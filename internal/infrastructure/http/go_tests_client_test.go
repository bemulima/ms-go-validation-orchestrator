package engines

import (
	"context"
	"encoding/json"
	"github.com/example/ms-validation-orchestrator-service/internal/domain"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGoExecutionProjectsOnlyExactSolution(t *testing.T) {
	calls := 0
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
		json.NewEncoder(w).Encode(domain.GoTestResult{Outcomes: map[string]domain.GoTestOutcome{"solution": {Status: "FAIL"}}})
	}))
	defer server.Close()
	c := NewGoTestsClient(server.URL, "test-token")
	checks := json.RawMessage(`{"profile_id":"go-double-numbers.v1","private_binding":{"schema":"go-test-execution.v1","profile_id":"go-double-numbers.v1","bundle_digest":"bundle","runner_digest":"runner"}}`)
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
