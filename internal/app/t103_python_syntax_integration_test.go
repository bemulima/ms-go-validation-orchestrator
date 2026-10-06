//go:build integration

package app

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/example/ms-validation-orchestrator-service/config"
	"github.com/example/ms-validation-orchestrator-service/internal/domain"
	"github.com/example/ms-validation-orchestrator-service/internal/usecase"
)

func TestT103PinnedPythonSyntaxUseActualProvider(t *testing.T) {
	fixturePath, pythonURL := os.Getenv("T103_PROVIDER_FIXTURE_FILE"), os.Getenv("T103_PYTHON_PROVIDER_URL")
	if fixturePath == "" {
		t.Skip("requires explicitly isolated actual Python and Sandbox providers")
	}
	var metadata struct {
		URL      string `json:"url"`
		Fixtures []struct {
			Kind                     string `json:"kind"`
			SandboxID                string `json:"sandbox_id"`
			TaskInstanceID           string `json:"task_instance_id"`
			SnapshotID               string `json:"snapshot_id"`
			GenerationID             string `json:"generation_id"`
			WorkspaceDigest          string `json:"workspace_digest"`
			Revision                 uint64 `json:"revision"`
			LiveHeadRevision         uint64 `json:"live_head_revision"`
			LiveHeadDigest           string `json:"live_head_digest"`
			RetainedContentUnchanged bool   `json:"retained_content_unchanged"`
		} `json:"fixtures"`
	}
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal("read safe fixture metadata")
	}
	if err = json.Unmarshal(data, &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata.URL == "" || pythonURL == "" || len(metadata.Fixtures) != 2 {
		t.Fatal("actual providers incomplete")
	}
	client := &http.Client{Timeout: 20 * time.Second}
	proof := map[string]any{"full_live_e2e": false, "boundary": "actual retained Sandbox/Tarantool/S3 immutable genesis → production Orchestrator App HTTP/pinned reader/Python adapter → actual Python create_server registered AST handler; public synthetic bindings, no learner journey or Django runtime"}
	coreRules := json.RawMessage(`{"requiredFunctions":[{"name":"greet","argsCount":0}]}`)
	mainSource := "raise RuntimeError('PUBLIC_NEVER_EXECUTE')\ndef greet():\n    return 'hello'\n"
	direct := map[string]any{}
	for _, engine := range []string{"python.core", "python.django"} {
		for _, kind := range []string{"positive", "negative", "broken_only", "excluded_bad"} {
			t.Run("owner_"+engine+"_"+kind, func(t *testing.T) {
				content := "value = 1\n"
				if kind != "positive" {
					content = "def broken(:\n    pass\n"
				}
				files := []map[string]string{{"path": "main.py", "content": mainSource}, {"path": "syntax.py", "content": content}}
				targets := []string{"main.py", "syntax.py"}
				rules := json.RawMessage(`{}`)
				if engine == "python.core" && kind != "broken_only" {
					rules = coreRules
				}
				if kind == "broken_only" {
					targets = []string{"syntax.py"}
				}
				if kind == "excluded_bad" {
					targets = []string{"main.py"}
				}
				body, e := json.Marshal(map[string]any{"stage": map[string]any{"engine": engine, "rules": rules, "targets": map[string]any{"files": targets}}, "workspace": map[string]any{"files": files}})
				if e != nil {
					t.Fatal(e)
				}
				res, e := client.Post(pythonURL+"/api/v1/validate", "application/json", bytes.NewReader(body))
				if e != nil {
					t.Fatal("actual Python unavailable")
				}
				defer res.Body.Close()
				var result struct {
					OK     bool `json:"ok"`
					Valid  bool `json:"isValid"`
					Errors []struct {
						Code    string `json:"code"`
						File    string `json:"file"`
						Message string `json:"message"`
					} `json:"errors"`
				}
				if e = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&result); e != nil {
					t.Fatal(e)
				}
				codes := []string{}
				found := false
				for _, issue := range result.Errors {
					codes = append(codes, issue.Code)
					found = found || issue.Code == "PYTHON_SYNTAX_ERROR" && issue.File == "syntax.py"
					if strings.Contains(issue.Message, "def broken") || strings.Contains(issue.Message, "invalid syntax") {
						t.Fatal("syntax source projection")
					}
				}
				direct[engine+"/"+kind] = map[string]any{"status": res.StatusCode, "ok": result.OK, "isValid": result.Valid, "codes": codes}
				want := kind == "positive" || kind == "excluded_bad"
				if res.StatusCode != 200 || result.OK != want || result.Valid != want {
					t.Fatalf("actual owner status=%d ok=%v valid=%v want=%v", res.StatusCode, result.OK, result.Valid, want)
				}
				if !want && !found {
					t.Fatal("selected syntax issue absent")
				}
			})
		}
	}
	proof["actual_python_owner"] = direct
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	var entropy [24]byte
	if _, err = rand.Read(entropy[:]); err != nil {
		t.Fatal("generate isolated credential")
	}
	cfg.InternalAPIToken = hex.EncodeToString(entropy[:])
	cfg.Engines = config.EngineEndpoints{Python: pythonURL}
	cfg.SandboxServiceBaseURL = metadata.URL
	cfg.SandboxServiceInternalToken = os.Getenv("T103_INTERNAL_TOKEN")
	if cfg.SandboxServiceInternalToken == "" {
		t.Fatal("missing scoped Sandbox credential")
	}
	application := New(cfg)
	server := httptest.NewServer(application.Server().Handler)
	defer server.Close()
	makeSpec := func(engine string) (json.RawMessage, string) {
		t.Helper()
		rules := json.RawMessage(`{}`)
		if engine == "python.core" {
			rules = coreRules
		}
		stage := map[string]any{"id": "python", "engine": engine, "language": "python", "mode": "final", "targets": map[string]any{"files": []string{"main.py", "syntax.py"}}, "rules": rules}
		contract, e := json.Marshal(map[string]any{"version": 1, "kind": "workspace_contract", "workspace": map[string]any{"required_files": []string{"main.py", "syntax.py"}}, "stages": []map[string]any{stage}})
		if e != nil {
			t.Fatal(e)
		}
		spec, e := json.Marshal(map[string]any{"schema": domain.PracticeValidationSpecificationSchemaV1, "runtime_profile": map[string]any{}, "contracts": []map[string]any{{"milestone_id": "python", "required": true, "validation_contract": json.RawMessage(contract), "evaluation_assets": map[string]any{}}}})
		if e != nil {
			t.Fatal(e)
		}
		digest, e := usecase.ComputePracticeValidationContractDigest(domain.PracticeValidationScopeFinal, "", spec)
		if e != nil {
			t.Fatal(e)
		}
		return spec, digest
	}
	call := func(input domain.PracticeValidationRequestV2, auth bool) (int, domain.PracticeValidationResultV2) {
		t.Helper()
		body, e := json.Marshal(input)
		if e != nil {
			t.Fatal(e)
		}
		req, e := http.NewRequest(http.MethodPost, server.URL+"/api/v2/practice-validations", bytes.NewReader(body))
		if e != nil {
			t.Fatal(e)
		}
		req.Header.Set("Content-Type", "application/json")
		if auth {
			req.Header.Set("X-Internal-Token", cfg.InternalAPIToken)
		}
		res, e := client.Do(req)
		if e != nil {
			t.Fatal("production App HTTP unavailable")
		}
		defer res.Body.Close()
		var result domain.PracticeValidationResultV2
		if res.StatusCode == 200 {
			if e = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&result); e != nil {
				t.Fatal(e)
			}
		}
		return res.StatusCode, result
	}
	var positive domain.PracticeValidationRequestV2
	for _, engine := range []string{"python.core", "python.django"} {
		for _, f := range metadata.Fixtures {
			if f.Revision != 0 || f.LiveHeadRevision <= f.Revision || f.WorkspaceDigest == f.LiveHeadDigest || !f.RetainedContentUnchanged {
				t.Fatal("retained genesis/opposite live head proof absent")
			}
			spec, digest := makeSpec(engine)
			input := domain.PracticeValidationRequestV2{Schema: domain.PracticeValidationRequestSchemaV2, TaskInstanceID: f.TaskInstanceID, SubmissionID: "22222222-2222-4222-8222-222222222222", ValidationRunID: "33333333-3333-4333-8333-333333333333", RevisionDigest: "sha256:" + strings.Repeat("a", 64), ValidationContractDigest: digest, Snapshot: domain.PracticeSnapshotRefV2{SandboxID: f.SandboxID, SnapshotID: f.SnapshotID, GenerationID: f.GenerationID, WorkspaceDigest: f.WorkspaceDigest}, Scope: domain.PracticeValidationScopeFinal, ValidationSpecification: spec}
			if f.Kind == "positive" {
				positive = input
			}
			t.Run("pinned_"+engine+"_"+f.Kind, func(t *testing.T) {
				status, result := call(input, true)
				want := domain.PracticeValidationFail
				if f.Kind == "positive" {
					want = domain.PracticeValidationPass
				}
				proof[engine+"/"+f.Kind] = result
				if status != 200 || result.Outcome != want {
					t.Fatalf("pinned %s status=%d outcome=%s/%s want=%s", f.Kind, status, result.Outcome, result.ErrorClassification, want)
				}
				if f.Kind == "negative" {
					found := false
					for _, stage := range result.Stages {
						for _, issue := range stage.Issues {
							found = found || issue.Code == "PYTHON_SYNTAX_ERROR"
						}
					}
					if !found {
						t.Fatal("retained negative selected syntax issue absent")
					}
				}
				if result.Snapshot != input.Snapshot || result.TaskInstanceID != input.TaskInstanceID || result.SubmissionID != input.SubmissionID || result.ValidationRunID != input.ValidationRunID || result.ValidationContractDigest != input.ValidationContractDigest || result.RevisionDigest != input.RevisionDigest {
					t.Fatal("correlation changed")
				}
			})
		}
	}
	if positive.TaskInstanceID == "" {
		t.Fatal("positive fixture missing")
	}
	t.Run("wrong_generation", func(t *testing.T) {
		input := positive
		input.Snapshot.GenerationID = "wrong-generation"
		status, result := call(input, true)
		proof["wrong_generation"] = result
		if status != 200 || result.Outcome != domain.PracticeValidationError {
			t.Fatalf("wrong pin must be technical error status=%d outcome=%s", status, result.Outcome)
		}
	})
	t.Run("anonymous", func(t *testing.T) {
		status, _ := call(positive, false)
		proof["unauthenticated_status"] = status
		if status != 401 {
			t.Fatalf("status=%d", status)
		}
	})
	proof["assertions_passed"] = !t.Failed()
	if output := os.Getenv("T103_RESULT_FILE"); output != "" {
		raw, e := json.MarshalIndent(proof, "", "  ")
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(output, append(raw, '\n'), 0600); e != nil {
			t.Fatal(e)
		}
	}
}
