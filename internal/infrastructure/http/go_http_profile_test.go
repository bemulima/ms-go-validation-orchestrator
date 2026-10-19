package engines

import (
	"context"
	"encoding/json"
	"github.com/example/ms-validation-orchestrator-service/internal/domain"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestHTTPModuleProjectionAndResponseBinding(t *testing.T) {
	raw, e := os.ReadFile("../../usecase/testdata/http_handler_capability.json")
	if e != nil {
		t.Fatal(e)
	}
	var m domain.GoTestManifest
	json.Unmarshal(raw, &m)
	mode := "valid"
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/internal/go-tests/capability" {
			if r.URL.Query().Get("profile_id") != m.ProfileID {
				t.Error("profile query missing")
			}
			json.NewEncoder(w).Encode(m)
			return
		}
		calls++
		var b domain.GoTestBinding
		json.NewDecoder(r.Body).Decode(&b)
		if len(b.Files) != 3 || b.Files["go.mod"] != m.StarterFiles["go.mod"] || b.Files["main.go"] != m.StarterFiles["main.go"] {
			t.Error("projection drift")
		}
		criteria := map[string]string{}
		for c := range m.CriterionTests {
			criteria[c] = "PASS"
		}
		o := domain.GoTestOutcome{Status: "PASS", Criteria: criteria, SourceDigest: domain.CanonicalDigestV2(b.Files)}
		result := domain.GoTestResult{Schema: "go-test-result.v1", Manifest: m, Outcomes: map[string]domain.GoTestOutcome{}}
		switch mode {
		case "error":
			o.Status = "ERROR"
		case "runner":
			result.Manifest.RunnerDigest = "wrong"
		case "source":
			o.SourceDigest = "wrong"
		case "missing":
			delete(criteria, "method")
		case "semantic":
			o.Status = "FAIL"
			criteria["method"] = "FAIL"
		}
		result.Outcomes["solution"] = o
		json.NewEncoder(w).Encode(result)
	}))
	defer server.Close()
	client := NewGoTestsClient(server.URL, "unit-token")
	if cap, e := client.CapabilityForProfile(context.Background(), m.ProfileID); e != nil || cap.ProfileID != m.ProfileID {
		t.Fatal(e)
	}
	b := domain.GoTestBinding{Schema: "go-test-execution.v1", ProfileID: m.ProfileID, BundleDigest: m.BundleDigest, RunnerDigest: m.RunnerDigest, ExecutionProfileDigest: m.ExecutionProfileDigest, StarterDigest: m.StarterDigest}
	checks, _ := json.Marshal(map[string]any{"profile_id": m.ProfileID, "private_binding": b})
	input := domain.EngineValidationInput{Mode: "final", Stage: domain.ValidationStage{Checks: checks}, Workspace: domain.ValidationWorkspace{Files: []domain.WorkspaceFile{{Path: "docker-compose.yml", Content: "managed"}}}}
	for _, p := range []string{"go.mod", "handler.go", "main.go"} {
		input.Workspace.Files = append(input.Workspace.Files, domain.WorkspaceFile{Path: p, Content: m.StarterFiles[p]})
	}
	for _, next := range []string{"valid", "semantic", "error", "runner", "source", "missing"} {
		mode = next
		result, e := client.Validate(context.Background(), input)
		if mode == "valid" {
			if e != nil || !result.Passed {
				t.Fatal("valid", e)
			}
		} else if mode == "semantic" {
			if e != nil || result.Passed || len(result.Errors) != 1 {
				t.Fatal("semantic", e)
			}
		} else if e == nil {
			t.Fatal("drift/ERROR accepted", mode)
		}
	}
	before := calls
	for _, files := range [][]domain.WorkspaceFile{input.Workspace.Files[:2], append(append([]domain.WorkspaceFile(nil), input.Workspace.Files...), domain.WorkspaceFile{Path: "extra.go"}), append(append([]domain.WorkspaceFile(nil), input.Workspace.Files...), domain.WorkspaceFile{Path: "main.go"})} {
		bad := input
		bad.Workspace.Files = files
		if _, e := client.Validate(context.Background(), bad); e == nil {
			t.Fatal("wrong source set")
		}
	}
	if calls != before {
		t.Fatal("invalid paths reached provider")
	}
}
