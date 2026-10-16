package engines

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type GoTestsClient struct {
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
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return errors.New("go executable provider unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New("go executable provider rejected request")
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20+1))
	if err != nil || len(b) > 1<<20 {
		return errors.New("invalid go executable response")
	}
	return json.Unmarshal(b, output)
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
	err := c.call(ctx, path, b, &result)
	return result, err
}

func (c *GoTestsClient) EngineID() string                                { return "go.tests.v1" }
func (c *GoTestsClient) NeedsWorkspaceFiles(domain.ValidationStage) bool { return true }
func (c *GoTestsClient) Validate(ctx context.Context, input domain.EngineValidationInput) (domain.StageExecutionResult, error) {
	var checks struct {
		ProfileID string               `json:"profile_id"`
		Binding   domain.GoTestBinding `json:"private_binding"`
	}
	if input.Mode != "final" || json.Unmarshal(input.Stage.Checks, &checks) != nil || checks.ProfileID != "go-double-numbers.v1" || checks.Binding.ProfileID != checks.ProfileID || checks.Binding.Schema != "go-test-execution.v1" {
		return domain.StageExecutionResult{}, errors.New("missing immutable Go evaluation binding")
	}
	b := checks.Binding
	b.Files = map[string]string{}
	for _, f := range input.Workspace.Files {
		if _, duplicate := b.Files[f.Path]; duplicate {
			return domain.StageExecutionResult{}, errors.New("duplicate Go input")
		}
		b.Files[f.Path] = f.Content
	}
	out, err := c.Run(ctx, b, false)
	if err != nil {
		return domain.StageExecutionResult{}, err
	}
	result, ok := out.Outcomes["solution"]
	if !ok || (result.Status != "PASS" && result.Status != "FAIL") {
		return domain.StageExecutionResult{}, errors.New("Go executable validation failed technically")
	}
	passed := result.Status == "PASS"
	if !passed {
		return domain.StageExecutionResult{Errors: []domain.ValidationIssue{{Code: "GO_TEST_CRITERIA_FAILED", Message: "Required executable criteria failed."}}}, nil
	}
	return domain.StageExecutionResult{Passed: true}, nil
}
