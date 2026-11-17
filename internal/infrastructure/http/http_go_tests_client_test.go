package engines

import (
	"context"
	"encoding/json"
	"github.com/example/ms-validation-orchestrator-service/internal/domain"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPGoExecutionUsesDedicatedOwnerAndMainFile(t *testing.T) {
	m := domain.GoTestManifest{Schema: "go-test-capability.v1", ProfileID: "go-http-health-main.v1", BundleDigest: "bundle", RunnerDigest: "runner", ExecutionProfileDigest: "execution", StarterDigest: "starter", TestFileDigests: map[string]string{"exercise_test.go": "digest"}, CriterionTests: map[string]string{"server_starts": "A", "health_status": "B", "health_body": "C", "health_content_type": "D", "unknown_path": "E", "method_rejected": "F", "repeat_request": "G"}}
	calls := 0
	source := "package main\nfunc main(){}"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/internal/go-http-tests/execute" || r.Header.Get("Authorization") != "Bearer http-only" {
			t.Error("cross-capability dispatch")
		}
		var b domain.GoTestBinding
		json.NewDecoder(r.Body).Decode(&b)
		if len(b.Files) != 1 || b.Files["main.go"] != source {
			t.Error("wrong source projection")
		}
		criteria := map[string]string{}
		for k := range m.CriterionTests {
			criteria[k] = "PASS"
		}
		criteria["health_body"] = "FAIL"
		out := domain.GoTestResult{Schema: "go-test-result.v1", Manifest: m, Outcomes: map[string]domain.GoTestOutcome{"solution": {Status: "FAIL", SourceDigest: goSolutionDigest(source), Criteria: criteria}}}
		raw, _ := json.Marshal(out)
		out.Digest = goSolutionDigest(string(raw))
		json.NewEncoder(w).Encode(out)
	}))
	defer server.Close()
	c := NewGoTestsClient(server.URL, "legacy-only")
	c.HTTPGoToken = "http-only"
	checks, _ := json.Marshal(map[string]any{"profile_id": m.ProfileID, "private_manifest": m, "private_binding": domain.GoTestBinding{Schema: "go-test-execution.v1", ProfileID: m.ProfileID, BundleDigest: m.BundleDigest, RunnerDigest: m.RunnerDigest, ExecutionProfileDigest: m.ExecutionProfileDigest, StarterDigest: m.StarterDigest}})
	in := domain.EngineValidationInput{Mode: "final", Stage: domain.ValidationStage{Checks: checks}, Workspace: domain.ValidationWorkspace{Files: []domain.WorkspaceFile{{Path: "main.go", Content: source}}}}
	out, e := c.Validate(context.Background(), in)
	if e != nil || out.Passed || len(out.Errors) != 1 || out.Errors[0].Code != "HTTP_HEALTH_BODY" || calls != 1 {
		t.Fatalf("out=%+v err=%v calls=%d", out, e, calls)
	}
	c.HTTPGoToken = ""
	if _, e = c.Validate(context.Background(), in); e == nil || calls != 1 {
		t.Fatal("default-off profile reached provider")
	}
}
