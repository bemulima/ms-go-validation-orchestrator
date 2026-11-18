package engines

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type GoTestsClient struct {
	HTTPGoToken    string
	prefix         string
	BaseURL, Token string
	HTTP           *http.Client
}

func NewGoTestsClient(base, token string) *GoTestsClient {
	return &GoTestsClient{BaseURL: strings.TrimRight(base, "/"), Token: token, HTTP: &http.Client{Timeout: 300 * time.Second}}
}
func (c *GoTestsClient) call(ctx context.Context, path string, input, output any) error {
	var body io.Reader
	method := http.MethodGet
	if input != nil {
		b, e := json.Marshal(input)
		if e != nil {
			return e
		}
		body = bytes.NewReader(b)
		method = http.MethodPost
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+strings.Replace(path, "/internal/go-tests/", c.routePrefix(), 1), body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	client := *c.HTTP
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := client.Do(req)
	if err != nil {
		return domain.GoTestProviderError{Cause: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		// Invalid authored definitions remain an authoring failure. Service,
		// auth and protocol failures preserve their existing typed classification.
		if input != nil && (resp.StatusCode == 400 || resp.StatusCode == 422) {
			return domain.ErrInvalidContract
		}
		return domain.GoTestProviderError{Cause: domain.ValidatorHTTPStatusError{StatusCode: resp.StatusCode}}
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20+1))
	if err != nil {
		var timeout interface{ Timeout() bool }
		if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &timeout) && timeout.Timeout()) {
			return domain.GoTestProviderError{Cause: err}
		}
		return domain.GoTestProviderError{Cause: domain.ErrValidatorProtocol}
	}
	if len(b) > 1<<20 {
		return domain.GoTestProviderError{Cause: domain.ErrValidatorProtocol}
	}
	if err := json.Unmarshal(b, output); err != nil {
		return domain.GoTestProviderError{Cause: domain.ErrValidatorProtocol}
	}
	return nil
}
func (c *GoTestsClient) Capability(ctx context.Context) (domain.GoTestManifest, error) {
	var m domain.GoTestManifest
	err := c.call(ctx, "/internal/go-tests/capability", nil, &m)
	return m, err
}
func (c *GoTestsClient) Run(ctx context.Context, b domain.GoTestBinding, verify bool) (domain.GoTestResult, error) {
	path := "/internal/go-tests/execute"
	if verify {
		path = "/internal/go-tests/verify"
	}
	var result domain.GoTestResult
	client := c
	if b.ProfileID == httpGoProfile {
		var e error
		client, e = c.httpProfile()
		if e != nil {
			return result, e
		}
	}
	err := client.call(ctx, path, b, &result)
	return result, err
}

func (c *GoTestsClient) EngineID() string                                { return "go.tests.v1" }
func (c *GoTestsClient) NeedsWorkspaceFiles(domain.ValidationStage) bool { return true }
func (c *GoTestsClient) Validate(ctx context.Context, input domain.EngineValidationInput) (domain.StageExecutionResult, error) {
	var checks struct {
		ProfileID string                `json:"profile_id"`
		Binding   domain.GoTestBinding  `json:"private_binding"`
		Manifest  domain.GoTestManifest `json:"private_manifest"`
	}
	if input.Mode != "final" || json.Unmarshal(input.Stage.Checks, &checks) != nil || checks.Binding.ProfileID != checks.ProfileID || checks.Binding.Schema != "go-test-execution.v1" {
		return domain.StageExecutionResult{}, errors.New("missing immutable Go evaluation binding")
	}
	allowedProfile := checks.ProfileID == "go-double-numbers.v1" || checks.ProfileID == "go-double-numbers-cases.v1" || checks.ProfileID == safeDivisionProfile || (checks.ProfileID == httpGoProfile && c.HTTPGoToken != "")
	if !allowedProfile {
		return domain.StageExecutionResult{}, errors.New("missing immutable Go evaluation binding")
	}
	sourceFile := "solution.go"
	if checks.ProfileID == httpGoProfile || checks.ProfileID == safeDivisionProfile {
		sourceFile = "main.go"
	}
	b := checks.Binding
	b.Files = map[string]string{}
	seen := map[string]bool{}
	for _, f := range input.Workspace.Files {
		if seen[f.Path] {
			return domain.StageExecutionResult{}, errors.New("duplicate Go input")
		}
		seen[f.Path] = true
		switch f.Path {
		case "README.md": // Published bounded public documentation, not executable source.
		case "docker-compose.yml": // Sandbox-managed operational file, outside executable source.
		case sourceFile:
			b.Files[f.Path] = f.Content
		default:
			return domain.StageExecutionResult{}, errors.New("unsupported Go input path")
		}
	}
	if len(b.Files) != 1 {
		return domain.StageExecutionResult{}, errors.New("one solution.go required")
	}
	out, err := c.Run(ctx, b, false)
	if err != nil {
		return domain.StageExecutionResult{}, err
	}
	protocolError := func() (domain.StageExecutionResult, error) {
		return domain.StageExecutionResult{}, domain.GoTestProviderError{Cause: domain.ErrValidatorProtocol}
	}
	if out.Schema != "go-test-result.v1" || !domain.MatchingGoTestBinding(b, checks.Manifest) || domain.CanonicalDigestV2(out.Manifest) != domain.CanonicalDigestV2(checks.Manifest) || len(out.Outcomes) != 1 {
		return protocolError()
	}
	receivedDigest := out.Digest
	out.Digest = ""
	rawResult, _ := json.Marshal(out)
	out.Digest = receivedDigest
	if receivedDigest != goSolutionDigest(string(rawResult)) {
		return protocolError()
	}
	result, ok := out.Outcomes["solution"]
	if !ok || (result.Status != "PASS" && result.Status != "FAIL") || result.SourceDigest != goSolutionDigest(b.Files[sourceFile]) {
		return protocolError()
	}
	if len(result.Criteria) != len(checks.Manifest.CriterionTests) {
		return protocolError()
	}
	failed := false
	for criterion := range checks.Manifest.CriterionTests {
		status := result.Criteria[criterion]
		if status != "PASS" && status != "FAIL" {
			return protocolError()
		}
		failed = failed || status == "FAIL"
	}
	passed := result.Status == "PASS"
	if passed == failed {
		return protocolError()
	}
	if !passed {
		if checks.ProfileID == httpGoProfile {
			issues := []domain.ValidationIssue{}
			for _, criterion := range []string{"server_starts", "health_status", "health_body", "health_content_type", "unknown_path", "method_rejected", "repeat_request"} {
				if result.Criteria[criterion] == "FAIL" {
					issues = append(issues, domain.ValidationIssue{Code: "HTTP_" + strings.ToUpper(criterion), Message: "Public HTTP criterion failed."})
				}
			}
			return domain.StageExecutionResult{Errors: issues}, nil
		}
		return domain.StageExecutionResult{Errors: []domain.ValidationIssue{{Code: "GO_TEST_CRITERIA_FAILED", Message: "Required executable criteria failed."}}}, nil
	}
	return domain.StageExecutionResult{Passed: true}, nil
}

func (c *GoTestsClient) Prepare(ctx context.Context, definitions json.RawMessage) (domain.GoTestManifest, error) {
	var m domain.GoTestManifest
	err := c.call(ctx, "/internal/go-tests/prepare", map[string]any{"schema": "go-test-prepare.v1", "profile_id": "go-double-numbers-cases.v1", "test_definitions": definitions}, &m)
	return m, err
}

func goSolutionDigest(source string) string {
	h := sha256.Sum256([]byte(source))
	return "sha256:" + hex.EncodeToString(h[:])
}

const httpGoProfile = "go-http-health-main.v1"
const safeDivisionProfile = "go-safe-division.v1"

func (c *GoTestsClient) routePrefix() string {
	if c.prefix != "" {
		return c.prefix
	}
	return "/internal/go-tests/"
}
func (c *GoTestsClient) httpProfile() (*GoTestsClient, error) {
	if c.HTTPGoToken == "" || c.HTTPGoToken == c.Token {
		return nil, errors.New("HTTP Go profile unavailable")
	}
	copy := *c
	copy.Token = c.HTTPGoToken
	copy.prefix = "/internal/go-http-tests/"
	return &copy, nil
}
func (c *GoTestsClient) CapabilityForProfile(ctx context.Context, profile string) (domain.GoTestManifest, error) {
	if profile == httpGoProfile {
		copy, e := c.httpProfile()
		if e != nil {
			return domain.GoTestManifest{}, e
		}
		return copy.Capability(ctx)
	}
	if profile == safeDivisionProfile {
		var m domain.GoTestManifest
		err := c.call(ctx, "/internal/go-tests/capability?profile_id="+safeDivisionProfile, nil, &m)
		if err == nil && m.ProfileID != safeDivisionProfile {
			return domain.GoTestManifest{}, domain.GoTestProviderError{Cause: domain.ErrValidatorProtocol}
		}
		return m, err
	}
	if profile != "go-double-numbers.v1" && profile != "go-double-numbers-cases.v1" {
		return domain.GoTestManifest{}, errors.New("unsupported Go profile")
	}
	return c.Capability(ctx)
}
