package engines

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

func TestGoSafeDivisionValidationProjectsOnlyMainGo(t *testing.T) {
	const source = "package exercise\nimport \"errors\"\nfunc SafeDivide(a, b int) (int, error) { if b == 0 { return 0, errors.New(\"division by zero\") }; return a / b, nil }\n"
	manifest := domain.GoTestManifest{
		Schema:                 "go-test-capability.v1",
		ProfileID:              "go-safe-division.v1",
		BundleVersion:          "safe-division.v1",
		BundleDigest:           "sha256:bundle",
		RunnerDigest:           "sha256:runner",
		ExecutionProfileDigest: "sha256:execution",
		StarterDigest:          "sha256:starter",
		CompilerImage:          "golang@sha256:" + strings.Repeat("a", 64),
		TestFileDigests:        map[string]string{"exercise_test.go": "sha256:tests"},
		CriterionTests: map[string]string{
			"positive_exact":     "TestPositiveExact",
			"quotient_truncates": "TestQuotientTruncates",
			"negative_numerator": "TestNegativeNumerator",
			"negative_divisor":   "TestNegativeDivisor",
			"both_negative":      "TestBothNegative",
			"zero_numerator":     "TestZeroNumerator",
			"zero_divisor_error": "TestZeroDivisorError",
		},
	}
	seen := false
	client := NewGoTestsClient("http://unit.invalid", "owner-token")
	client.HTTP = &http.Client{Transport: goProviderRoundTrip(func(r *http.Request) (*http.Response, error) {
		seen = true
		if r.URL.Path != "/internal/go-tests/execute" {
			t.Errorf("unexpected execution path: %q", r.URL.Path)
		}
		var binding domain.GoTestBinding
		if json.NewDecoder(r.Body).Decode(&binding) != nil || len(binding.Files) != 1 || binding.Files["main.go"] != source {
			t.Errorf("SafeDivide source was not projected under main.go: %+v", binding.Files)
		}
		result := domain.GoTestResult{Schema: "go-test-result.v1", Manifest: manifest, Outcomes: map[string]domain.GoTestOutcome{"solution": {Status: "PASS", SourceDigest: goSolutionDigest(source)}}}
		signGoResultFixture(&result)
		b, _ := json.Marshal(result)
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(b))}, nil
	})}

	checks, _ := json.Marshal(map[string]any{
		"profile_id":       manifest.ProfileID,
		"private_binding":  domain.GoTestBinding{Schema: "go-test-execution.v1", ProfileID: manifest.ProfileID, BundleDigest: manifest.BundleDigest, RunnerDigest: manifest.RunnerDigest, ExecutionProfileDigest: manifest.ExecutionProfileDigest, StarterDigest: manifest.StarterDigest},
		"private_manifest": manifest,
	})
	input := domain.EngineValidationInput{Mode: "final", Stage: domain.ValidationStage{Checks: checks}, Workspace: domain.ValidationWorkspace{Files: []domain.WorkspaceFile{{Path: "main.go", Content: source}}}}
	out, err := client.Validate(context.Background(), input)
	if err != nil || !seen || !out.Passed {
		t.Fatalf("SafeDivide validation failed: out=%+v seen=%v err=%v", out, seen, err)
	}
}
