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

	"github.com/example/ms-validation-orchestrator-service/config"
	"github.com/example/ms-validation-orchestrator-service/internal/domain"
	"github.com/example/ms-validation-orchestrator-service/internal/usecase"
)

func TestT092OfficialSnapshotUsesActualProviders(t *testing.T) {
	fixturePath := os.Getenv("T092_PROVIDER_FIXTURE_FILE")
	if fixturePath == "" {
		t.Skip("requires explicitly owned actual Sandbox/HTML providers")
	}
	var fixture struct {
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
		t.Fatal("read owned provider metadata")
	}
	if err = json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	var entropy [24]byte
	if _, err = rand.Read(entropy[:]); err != nil {
		t.Fatal("generate isolated caller identity")
	}
	cfg.InternalAPIToken = hex.EncodeToString(entropy[:])
	cfg.Engines = config.EngineEndpoints{HTML: os.Getenv("T092_HTML_PROVIDER_URL"), Git: os.Getenv("T092_GIT_PROVIDER_URL"), HTTPRuntime: os.Getenv("T092_HTTP_PROVIDER_URL")}
	if cfg.Engines.HTML == "" || cfg.Engines.Git == "" || cfg.Engines.HTTPRuntime == "" || fixture.URL == "" || len(fixture.Fixtures) != 2 {
		t.Fatal("missing actual provider configuration")
	}
	cfg.SandboxServiceBaseURL = fixture.URL
	cfg.SandboxServiceInternalToken = os.Getenv("T092_INTERNAL_TOKEN")
	if cfg.SandboxServiceInternalToken == "" {
		t.Fatal("missing scoped service credential")
	}
	application := New(cfg)
	server := httptest.NewServer(application.Server().Handler)
	defer server.Close()
	call := func(request domain.PracticeValidationRequestV2, authorized bool) (int, domain.PracticeValidationResultV2) {
		t.Helper()
		body, e := json.Marshal(request)
		if e != nil {
			t.Fatal(e)
		}
		req, e := http.NewRequest(http.MethodPost, server.URL+"/api/v2/practice-validations", bytes.NewReader(body))
		if e != nil {
			t.Fatal(e)
		}
		req.Header.Set("Content-Type", "application/json")
		if authorized {
			req.Header.Set("X-Internal-Token", cfg.InternalAPIToken)
		}
		res, e := server.Client().Do(req)
		if e != nil {
			t.Fatal("call actual application")
		}
		defer res.Body.Close()
		raw, e := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		if e != nil {
			t.Fatal(e)
		}
		var result domain.PracticeValidationResultV2
		if res.StatusCode == http.StatusOK {
			if e = json.Unmarshal(raw, &result); e != nil {
				t.Fatal(e)
			}
		}
		return res.StatusCode, result
	}
	contract := json.RawMessage(`{"version":1,"kind":"workspace_contract","stages":[{"id":"html","engine":"html.dom","mode":"final","targets":{"files":["index.html"]},"rules":{"version":2,"head":{"doctype":true,"htmlLang":"en"},"body":{"roots":[{"tag":{"value":"h1"},"attributes":[{"name":"class","value":"title"}],"text":{"value":"Hello World!"}}]}}}]}`)
	makeRequest := func(taskID string, ref domain.PracticeSnapshotRefV2, c json.RawMessage) domain.PracticeValidationRequestV2 {
		t.Helper()
		spec, e := json.Marshal(map[string]any{"schema": domain.PracticeValidationSpecificationSchemaV1, "runtime_profile": map[string]any{}, "contracts": []map[string]any{{"milestone_id": "html-check", "required": true, "validation_contract": c, "evaluation_assets": map[string]any{}}}})
		if e != nil {
			t.Fatal(e)
		}
		digest, e := usecase.ComputePracticeValidationContractDigest(domain.PracticeValidationScopeFinal, "", spec)
		if e != nil {
			t.Fatal(e)
		}
		return domain.PracticeValidationRequestV2{Schema: domain.PracticeValidationRequestSchemaV2, TaskInstanceID: taskID, SubmissionID: "22222222-2222-4222-8222-222222222222", ValidationRunID: "33333333-3333-4333-8333-333333333333", RevisionDigest: "sha256:" + strings.Repeat("a", 64), ValidationContractDigest: digest, Snapshot: ref, Scope: domain.PracticeValidationScopeFinal, ValidationSpecification: spec}
	}
	proofs := map[string]any{"full_live_e2e": false, "classification": "actual Sandbox HTTP/Tarantool/S3 retained pin → production Orchestrator App HTTP → actual HTML provider; preseeded isolated bindings, not Start/RuntimeSubmit/learnerE2E"}
	// Characterize both actual strict-root providers before the Orchestrator
	// assertion. They reject before Git inspection/runtime process launch.
	direct := map[string]any{}
	for _, item := range []struct{ engine, endpoint, code string }{
		{"git.core", cfg.Engines.Git, "GIT_WORKSPACE_ROOT_REQUIRED"},
		{"http.runtime", cfg.Engines.HTTPRuntime, "HTTP_RUNTIME_ROOT_REQUIRED"},
	} {
		body, e := json.Marshal(map[string]any{"stage": map[string]any{"id": "t092-prerequisite", "engine": item.engine, "rules": map[string]any{}, "checks": map[string]any{"command": []string{"true"}}}, "workspace": map[string]any{"files": []domain.WorkspaceFile{{Path: "index.html", Content: "synthetic text only"}}}})
		if e != nil {
			t.Fatal(e)
		}
		res, e := server.Client().Post(item.endpoint+"/api/v1/validate", "application/json", bytes.NewReader(body))
		if e != nil {
			t.Fatal("actual root-prerequisite provider unavailable")
		}
		var owner struct {
			OK      bool `json:"ok"`
			IsValid bool `json:"isValid"`
			Errors  []struct {
				Code string `json:"code"`
			} `json:"errors"`
		}
		e = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&owner)
		res.Body.Close()
		if e != nil || res.StatusCode != http.StatusOK || owner.IsValid || len(owner.Errors) == 0 || owner.Errors[0].Code != item.code {
			t.Fatalf("actual %s prerequisite response invalid status=%d", item.engine, res.StatusCode)
		}
		direct[item.engine] = map[string]any{"status": res.StatusCode, "is_valid": owner.IsValid, "error_code": owner.Errors[0].Code, "execution_root_supplied": false}
	}
	proofs["actual_provider_prerequisites"] = direct
	if output := os.Getenv("T092_RESULT_FILE"); output != "" {
		raw, e := json.MarshalIndent(direct, "", "  ")
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(output+".provider-prerequisites.json", append(raw, '\n'), 0o600); e != nil {
			t.Fatal(e)
		}
	}
	var positive domain.PracticeValidationRequestV2
	for _, f := range fixture.Fixtures {
		if f.Revision != 0 || f.LiveHeadRevision <= f.Revision || f.LiveHeadDigest == f.WorkspaceDigest || !f.RetainedContentUnchanged {
			t.Fatal("provider has not proven actual genesis pin/live divergence")
		}
		ref := domain.PracticeSnapshotRefV2{SandboxID: f.SandboxID, SnapshotID: f.SnapshotID, GenerationID: f.GenerationID, WorkspaceDigest: f.WorkspaceDigest}
		request := makeRequest(f.TaskInstanceID, ref, contract)
		status, result := call(request, true)
		want := domain.PracticeValidationFail
		if f.Kind == "positive" {
			want = domain.PracticeValidationPass
			positive = request
		}
		if status != http.StatusOK || result.Outcome != want {
			t.Fatalf("actual immutable %s genesis0 outcome=%s/%s status=%d want=%s", f.Kind, result.Outcome, result.ErrorClassification, status, want)
		}
		if result.Snapshot != ref || result.TaskInstanceID != request.TaskInstanceID || result.SubmissionID != request.SubmissionID || result.ValidationRunID != request.ValidationRunID || result.RevisionDigest != request.RevisionDigest || result.ValidationContractDigest != request.ValidationContractDigest {
			t.Fatal("actual result lost correlation")
		}
		proofs[f.Kind] = result
	}
	if positive.TaskInstanceID == "" {
		t.Fatal("positive fixture absent")
	}
	status, _ := call(positive, false)
	if status != http.StatusUnauthorized {
		t.Fatalf("auth status=%d", status)
	}
	proofs["unauthenticated_status"] = status
	for _, kind := range []string{"digest", "generation", "missing-pin"} {
		req := positive
		switch kind {
		case "digest":
			req.Snapshot.WorkspaceDigest = "sha256:" + strings.Repeat("f", 64)
		case "generation":
			req.Snapshot.GenerationID = "t092-wrong-generation"
		case "missing-pin":
			req.Snapshot.SnapshotID = "t092-missing-pin"
		}
		status, result := call(req, true)
		if status != http.StatusOK || result.Outcome != domain.PracticeValidationError {
			t.Fatalf("%s status=%d outcome=%s", kind, status, result.Outcome)
		}
		proofs[kind] = result
	}
	proofs["proof_status"] = "supported_complete_unsupported_pending"
	if output := os.Getenv("T092_RESULT_FILE"); output != "" {
		raw, e := json.MarshalIndent(proofs, "", "  ")
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(output, append(raw, '\n'), 0o600); e != nil {
			t.Fatal(e)
		}
	}
	for _, engine := range []string{"git.core", "http.runtime"} {
		endpoint := cfg.Engines.Git
		if engine == "http.runtime" {
			endpoint = cfg.Engines.HTTPRuntime
		}
		if endpoint == "" {
			t.Fatal("required actual root-prerequisite provider missing")
		}
		c := json.RawMessage(`{"version":1,"kind":"workspace_contract","stages":[{"id":"unsupported-input","engine":"` + engine + `","mode":"final","rules":{},"checks":{"command":["true"]}}]}`)
		req := makeRequest(positive.TaskInstanceID, positive.Snapshot, c)
		status, result := call(req, true)
		if status != http.StatusOK || result.Outcome != domain.PracticeValidationError || result.ErrorClassification != domain.PracticeValidationUnsupportedConfiguration {
			t.Errorf("files-only %s status=%d outcome=%s classification=%s; must be technical unsupported", engine, status, result.Outcome, result.ErrorClassification)
		}
		proofs[engine] = result
	}
	proofs["proof_status"] = "all_configured_checks_executed"
	proofs["assertions_passed"] = !t.Failed()
	if output := os.Getenv("T092_RESULT_FILE"); output != "" {
		raw, e := json.MarshalIndent(proofs, "", "  ")
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(output, append(raw, '\n'), 0o600); e != nil {
			t.Fatal(e)
		}
	}
}
