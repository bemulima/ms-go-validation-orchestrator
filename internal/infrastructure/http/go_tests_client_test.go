package engines

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/example/ms-validation-orchestrator-service/internal/domain"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestGoExecutionProjectsOnlyExactSolution(t *testing.T) {
	calls := 0
	m := domain.GoTestManifest{Schema: "go-test-capability.v1", ProfileID: "go-double-numbers.v1", BundleDigest: "bundle", RunnerDigest: "runner", ExecutionProfileDigest: "execution", StarterDigest: "starter", TestFileDigests: map[string]string{"test": "digest"}, CriterionTests: map[string]string{"a": "A", "b": "B", "c": "C", "d": "D", "e": "E", "f": "F", "g": "G"}}
	source := "package exercise\nfunc DoubleNumbers(numbers []int) []int { return nil }\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var b domain.GoTestBinding
		if json.NewDecoder(r.Body).Decode(&b) != nil {
			t.Fatal("decode")
		}
		if len(b.Files) != 1 || b.Files["solution.go"] != source || b.BundleDigest != "bundle" || b.RunnerDigest != "runner" {
			t.Errorf("incorrect projection/binding")
		}
		out := domain.GoTestResult{Schema: "go-test-result.v1", Manifest: m, Outcomes: map[string]domain.GoTestOutcome{"solution": {Status: "FAIL", SourceDigest: goSolutionDigest(source)}}}
		signGoResultFixture(&out)
		json.NewEncoder(w).Encode(out)
	}))
	defer server.Close()
	c := NewGoTestsClient(server.URL, "test-token")
	checks, _ := json.Marshal(map[string]any{"profile_id": m.ProfileID, "private_manifest": m, "private_binding": domain.GoTestBinding{Schema: "go-test-execution.v1", ProfileID: m.ProfileID, BundleDigest: m.BundleDigest, RunnerDigest: m.RunnerDigest, ExecutionProfileDigest: m.ExecutionProfileDigest, StarterDigest: m.StarterDigest}})
	base := domain.EngineValidationInput{Mode: "final", Stage: domain.ValidationStage{Checks: checks}, Workspace: domain.ValidationWorkspace{Files: []domain.WorkspaceFile{{Path: "solution.go", Content: source}, {Path: "docker-compose.yml", Content: "managed"}}}}
	out, err := c.Validate(context.Background(), base)
	if err != nil || out.Passed || len(out.Errors) != 1 {
		t.Fatalf("out=%+v err=%v", out, err)
	}
	if calls != 1 {
		t.Fatal("not called")
	}
	for _, files := range [][]domain.WorkspaceFile{{{Path: "docker-compose.yml"}}, {{Path: "solution.go", Content: source}, {Path: "private_test.go"}}, {{Path: "solution.go", Content: source}, {Path: "solution.go", Content: source}}, {{Path: "solution.go", Content: source}, {Path: "docker-compose.yml"}, {Path: "docker-compose.yml"}}, {{Path: "./solution.go", Content: source}}} {
		in := base
		in.Workspace.Files = files
		if _, err := c.Validate(context.Background(), in); err == nil {
			t.Fatal("invalid input accepted")
		}
	}
	if calls != 1 {
		t.Fatal("invalid paths reached provider")
	}
}

func TestGoExecutionRejectsEveryReturnedManifestDrift(t *testing.T) {
	raw, e := os.ReadFile("../../usecase/testdata/go_executable_gate.json")
	if e != nil {
		t.Fatal(e)
	}
	var fixture domain.GoTestResult
	json.Unmarshal(raw, &fixture)
	m := fixture.Manifest
	b := domain.GoTestBinding{Schema: "go-test-execution.v1", ProfileID: m.ProfileID, BundleDigest: m.BundleDigest, RunnerDigest: m.RunnerDigest, ExecutionProfileDigest: m.ExecutionProfileDigest, StarterDigest: m.StarterDigest}
	checks, _ := json.Marshal(map[string]any{"profile_id": m.ProfileID, "private_binding": b, "private_manifest": m})
	source := "package exercise"
	in := domain.EngineValidationInput{Mode: "final", Stage: domain.ValidationStage{Checks: checks}, Workspace: domain.ValidationWorkspace{Files: []domain.WorkspaceFile{{Path: "solution.go", Content: source}}}}
	for _, field := range []string{"valid", "profile", "bundle", "runner", "execution", "starter", "definitions", "compiler", "tests", "criteria", "schema", "source"} {
		t.Run(field, func(t *testing.T) {
			var out domain.GoTestResult
			raw, _ := json.Marshal(m)
			json.Unmarshal(raw, &out.Manifest)
			out.Schema = "go-test-result.v1"
			out.Outcomes = map[string]domain.GoTestOutcome{"solution": {Status: "PASS", SourceDigest: goSolutionDigest(source)}}
			switch field {
			case "profile":
				out.Manifest.ProfileID = "other"
			case "bundle":
				out.Manifest.BundleDigest = "other"
			case "runner":
				out.Manifest.RunnerDigest = "other"
			case "execution":
				out.Manifest.ExecutionProfileDigest = "other"
			case "starter":
				out.Manifest.StarterDigest = "other"
			case "definitions":
				out.Manifest.DefinitionsDigest = "other"
			case "compiler":
				out.Manifest.CompilerImage = "other"
			case "tests":
				out.Manifest.TestFileDigests = nil
			case "criteria":
				out.Manifest.CriterionTests = nil
			case "schema":
				out.Schema = "other"
			case "source":
				out.Outcomes["solution"] = domain.GoTestOutcome{Status: "PASS", SourceDigest: "wrong"}
			}
			client := NewGoTestsClient("http://unit.invalid", "unit")
			client.HTTP = &http.Client{Transport: goProviderRoundTrip(func(*http.Request) (*http.Response, error) {
				signGoResultFixture(&out)
				raw, _ := json.Marshal(out)
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(raw))}, nil
			})}
			result, e := client.Validate(context.Background(), in)
			if field == "valid" {
				if e != nil || !result.Passed {
					t.Fatal(e)
				}
				return
			}
			var pe domain.GoTestProviderError
			if !errors.As(e, &pe) || !errors.Is(e, domain.ErrValidatorProtocol) || result.Passed {
				t.Fatalf("drift accepted: %v", e)
			}
		})
	}
}

func signGoResultFixture(out *domain.GoTestResult) {
	for name, result := range out.Outcomes {
		result.Criteria = map[string]string{}
		for criterion := range out.Manifest.CriterionTests {
			result.Criteria[criterion] = "PASS"
		}
		if result.Status == "FAIL" {
			for criterion := range result.Criteria {
				result.Criteria[criterion] = "FAIL"
				break
			}
		}
		out.Outcomes[name] = result
	}
	out.Digest = ""
	raw, _ := json.Marshal(out)
	out.Digest = goSolutionDigest(string(raw))
}
