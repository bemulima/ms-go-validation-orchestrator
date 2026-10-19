package engines

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

// GoRunnerRoute is deployment-owned configuration, never a request DTO.
// Binding has no schema/files: only the five immutable identity fields select a build.
type GoRunnerRoute struct {
	Binding domain.GoTestBinding `json:"binding"`
	URL     string               `json:"url"`
	Token   string               `json:"token"`
}
type goRunnerTarget struct {
	binding domain.GoTestBinding
	client  *GoTestsClient
}

var goRouteDigest = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

func NewConfiguredGoTestsClient(base, token, raw string) (*GoTestsClient, error) {
	c := NewGoTestsClient(base, token)
	if strings.TrimSpace(raw) == "" {
		return c, nil
	}
	var routes []GoRunnerRoute
	d := json.NewDecoder(strings.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&routes) != nil || len(routes) == 0 || len(routes) > 16 {
		return nil, errors.New("invalid Go runner routes")
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return nil, errors.New("invalid Go runner routes")
	}
	c.routes = map[string]goRunnerTarget{}
	for _, route := range routes {
		b := route.Binding
		u, e := url.Parse(route.URL)
		if e != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || strings.TrimSpace(route.Token) == "" || strings.ContainsAny(route.Token, "\r\n") || b.Schema != "" || len(b.Files) != 0 || !validGoRouteIdentity(b) {
			return nil, errors.New("invalid Go runner route")
		}
		if _, exists := c.routes[b.RunnerDigest]; exists {
			return nil, errors.New("duplicate Go runner digest route")
		}
		target := NewGoTestsClient(route.URL, route.Token)
		target.HTTP.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		c.routes[b.RunnerDigest] = goRunnerTarget{binding: b, client: target}
	}
	return c, nil
}
func validGoRouteIdentity(b domain.GoTestBinding) bool {
	if b.ProfileID != "go-double-numbers.v1" && b.ProfileID != "go-http-handler.v1" {
		return false
	}
	for _, d := range []string{b.RunnerDigest, b.BundleDigest, b.ExecutionProfileDigest, b.StarterDigest} {
		if !goRouteDigest.MatchString(d) {
			return false
		}
	}
	return true
}
func sameGoIdentity(b domain.GoTestBinding, m domain.GoTestManifest) bool {
	return m.Schema == "go-test-capability.v1" && b.ProfileID == m.ProfileID && b.RunnerDigest == m.RunnerDigest && b.BundleDigest == m.BundleDigest && b.ExecutionProfileDigest == m.ExecutionProfileDigest && b.StarterDigest == m.StarterDigest
}
func (c *GoTestsClient) boundTarget(b domain.GoTestBinding) (*GoTestsClient, error) {
	if c.routes == nil {
		return c, nil
	}
	target, ok := c.routes[b.RunnerDigest]
	if !ok || !validGoRouteIdentity(b) || b.ProfileID != target.binding.ProfileID || b.BundleDigest != target.binding.BundleDigest || b.ExecutionProfileDigest != target.binding.ExecutionProfileDigest || b.StarterDigest != target.binding.StarterDigest {
		return nil, errors.New("unknown or mismatched Go runner binding")
	}
	return target.client, nil
}
func (c *GoTestsClient) CapabilityForBinding(ctx context.Context, b domain.GoTestBinding) (domain.GoTestManifest, error) {
	target, e := c.boundTarget(b)
	if e != nil {
		return domain.GoTestManifest{}, e
	}
	m, e := target.CapabilityForProfile(ctx, b.ProfileID)
	if e != nil {
		return domain.GoTestManifest{}, e
	}
	if !sameGoIdentity(b, m) {
		return domain.GoTestManifest{}, errors.New("Go runner target capability drift")
	}
	return m, nil
}
func (c *GoTestsClient) runBound(ctx context.Context, b domain.GoTestBinding, verify bool) (domain.GoTestResult, error) {
	target, e := c.boundTarget(b)
	if e != nil {
		return domain.GoTestResult{}, e
	}
	// Check the actual pinned binary's capability before execution; never try another target.
	m, e := c.CapabilityForBinding(ctx, b)
	if e != nil {
		return domain.GoTestResult{}, e
	}
	out, e := target.Run(ctx, b, verify)
	if e != nil {
		return domain.GoTestResult{}, e
	}
	if out.Schema != "go-test-result.v1" || domain.CanonicalDigestV2(m) != domain.CanonicalDigestV2(out.Manifest) {
		return domain.GoTestResult{}, errors.New("Go runner response capability drift")
	}
	want := []string{"solution"}
	if verify {
		want = []string{"reference", "starter", "negative"}
	}
	if len(out.Outcomes) != len(want) || len(m.CriterionTests) == 0 {
		return domain.GoTestResult{}, errors.New("Go runner outcome coverage drift")
	}
	for _, name := range want {
		o, ok := out.Outcomes[name]
		expected := m.FixtureDigests[name]
		if !verify {
			expected = domain.CanonicalDigestV2(b.Files)
			if b.ProfileID == "go-double-numbers.v1" {
				sum := sha256.Sum256([]byte(b.Files["solution.go"]))
				expected = "sha256:" + hex.EncodeToString(sum[:])
			}
		}
		if !ok || o.SourceDigest != expected || (o.Status != "PASS" && o.Status != "FAIL") || len(o.Criteria) != len(m.CriterionTests) {
			return domain.GoTestResult{}, errors.New("Go execution technical error or source binding drift")
		}
		failed := false
		for criterion := range m.CriterionTests {
			status := o.Criteria[criterion]
			if status != "PASS" && status != "FAIL" {
				return domain.GoTestResult{}, errors.New("Go runner incomplete semantic checks")
			}
			failed = failed || status == "FAIL"
		}
		if (o.Status == "PASS" && failed) || (o.Status == "FAIL" && !failed) {
			return domain.GoTestResult{}, errors.New("Go runner inconsistent semantic result")
		}
	}
	return out, nil
}
