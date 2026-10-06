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

func TestT093PinnedReactArrayRulesUseActualProvider(t *testing.T) {
	fixturePath, reactURL := os.Getenv("T093_PROVIDER_FIXTURE_FILE"), os.Getenv("T093_REACT_PROVIDER_URL")
	if fixturePath == "" {
		t.Skip("requires explicitly owned actual Sandbox/React providers")
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
		t.Fatal("read safe provider metadata")
	}
	if err = json.Unmarshal(data, &metadata); err != nil {
		t.Fatal(err)
	}
	if reactURL == "" || metadata.URL == "" || len(metadata.Fixtures) != 2 {
		t.Fatal("actual provider configuration incomplete")
	}
	rules := json.RawMessage(`[{"kind":"component_exists","name":"App","export":"default","componentKind":"function"},{"kind":"jsx_tree","component":"App","selector":"button","required":true}]`)
	client := &http.Client{Timeout: 10 * time.Second}
	direct := map[string]any{}
	for _, kind := range []string{"positive", "negative"} {
		tag := "button"
		if kind == "negative" {
			tag = "div"
		}
		code := "export default function App(){ return <" + tag + ">Public ready</" + tag + ">; }\n"
		body, e := json.Marshal(map[string]any{"language": "tsx", "framework": "react", "code": code, "rules": rules})
		if e != nil {
			t.Fatal(e)
		}
		res, e := client.Post(reactURL+"/validate", "application/json", bytes.NewReader(body))
		if e != nil {
			t.Fatal("actual React owner unavailable")
		}
		var owner struct {
			Success bool `json:"success"`
			IsValid bool `json:"isValid"`
			Errors  []struct {
				Code string `json:"code"`
			} `json:"errors"`
			Meta struct {
				ParsedOK *bool `json:"parsedOk"`
			} `json:"meta"`
		}
		e = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&owner)
		res.Body.Close()
		want := kind == "positive"
		if e != nil || res.StatusCode != http.StatusOK || owner.Success != want || owner.IsValid != want {
			t.Fatalf("actual React %s fixture status=%d success=%v parsed_on_wire=%v valid=%v", kind, res.StatusCode, owner.Success, owner.Meta.ParsedOK, owner.IsValid)
		}
		codes := []string{}
		for _, issue := range owner.Errors {
			codes = append(codes, issue.Code)
		}
		if kind == "negative" {
			found := false
			for _, code := range codes {
				if code == "REACT_JSX_NODE_NOT_FOUND" {
					found = true
				}
			}
			if !found {
				t.Fatal("valid negative fixture did not fail the required JSX criterion")
			}
		}
		direct[kind] = map[string]any{"status": res.StatusCode, "is_valid": owner.IsValid, "parsed_ok_on_wire": owner.Meta.ParsedOK, "error_codes": codes}
	}
	if output := os.Getenv("T093_RESULT_FILE"); output != "" {
		raw, e := json.MarshalIndent(direct, "", "  ")
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(output+".provider-characterization.json", append(raw, '\n'), 0o600); e != nil {
			t.Fatal(e)
		}
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	var entropy [24]byte
	if _, err = rand.Read(entropy[:]); err != nil {
		t.Fatal("generate isolated credential")
	}
	cfg.InternalAPIToken = hex.EncodeToString(entropy[:])
	cfg.Engines = config.EngineEndpoints{React: reactURL}
	cfg.SandboxServiceBaseURL = metadata.URL
	cfg.SandboxServiceInternalToken = os.Getenv("T093_INTERNAL_TOKEN")
	if cfg.SandboxServiceInternalToken == "" {
		t.Fatal("missing scoped Sandbox credential")
	}
	application := New(cfg)
	server := httptest.NewServer(application.Server().Handler)
	defer server.Close()
	contract, err := json.Marshal(map[string]any{"version": 1, "kind": "workspace_contract", "workspace": map[string]any{"required_files": []string{"src/App.tsx"}}, "stages": []map[string]any{{"id": "react", "engine": "react.ast", "language": "tsx", "framework": "react", "mode": "final", "targets": map[string]any{"files": []string{"src/App.tsx"}}, "rules": rules}}})
	if err != nil {
		t.Fatal(err)
	}
	spec, err := json.Marshal(map[string]any{"schema": domain.PracticeValidationSpecificationSchemaV1, "runtime_profile": map[string]any{}, "contracts": []map[string]any{{"milestone_id": "react", "required": true, "validation_contract": json.RawMessage(contract), "evaluation_assets": map[string]any{}}}})
	if err != nil {
		t.Fatal(err)
	}
	digest, err := usecase.ComputePracticeValidationContractDigest(domain.PracticeValidationScopeFinal, "", spec)
	if err != nil {
		t.Fatal(err)
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
			t.Fatal("call production App HTTP")
		}
		defer res.Body.Close()
		var result domain.PracticeValidationResultV2
		if res.StatusCode == http.StatusOK {
			if e = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&result); e != nil {
				t.Fatal(e)
			}
		}
		return res.StatusCode, result
	}
	proofs := map[string]any{"actual_react_owner": direct, "full_live_e2e": false, "classification": "actual retained Sandbox/Tarantool/S3 genesis pin → production Orchestrator App HTTP/parser/adapter → actual React production DI/AST validators; synthetic bindings, not learnerE2E"}
	var positive domain.PracticeValidationRequestV2
	for _, f := range metadata.Fixtures {
		if f.Revision != 0 || f.LiveHeadRevision <= f.Revision || f.WorkspaceDigest == f.LiveHeadDigest || !f.RetainedContentUnchanged {
			t.Fatal("immutable pin/opposite live head proof absent")
		}
		request := domain.PracticeValidationRequestV2{Schema: domain.PracticeValidationRequestSchemaV2, TaskInstanceID: f.TaskInstanceID, SubmissionID: "22222222-2222-4222-8222-222222222222", ValidationRunID: "33333333-3333-4333-8333-333333333333", RevisionDigest: "sha256:" + strings.Repeat("a", 64), ValidationContractDigest: digest, Snapshot: domain.PracticeSnapshotRefV2{SandboxID: f.SandboxID, SnapshotID: f.SnapshotID, GenerationID: f.GenerationID, WorkspaceDigest: f.WorkspaceDigest}, Scope: domain.PracticeValidationScopeFinal, ValidationSpecification: spec}
		status, result := call(request, true)
		want := domain.PracticeValidationFail
		if f.Kind == "positive" {
			want = domain.PracticeValidationPass
			positive = request
		}
		if status != http.StatusOK || result.Outcome != want {
			t.Fatalf("actual pinned React %s status=%d outcome=%s/%s want=%s", f.Kind, status, result.Outcome, result.ErrorClassification, want)
		}
		if result.Snapshot != request.Snapshot || result.TaskInstanceID != request.TaskInstanceID || result.SubmissionID != request.SubmissionID || result.ValidationRunID != request.ValidationRunID || result.RevisionDigest != request.RevisionDigest || result.ValidationContractDigest != request.ValidationContractDigest {
			t.Fatal("correlation changed")
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
	wrong := positive
	wrong.Snapshot.GenerationID = "t093-wrong-generation"
	status, result := call(wrong, true)
	if status != http.StatusOK || result.Outcome != domain.PracticeValidationError {
		t.Fatalf("wrongref status=%d outcome=%s", status, result.Outcome)
	}
	proofs["wrong_generation"] = result
	proofs["assertions_passed"] = true
	if output := os.Getenv("T093_RESULT_FILE"); output != "" {
		raw, e := json.MarshalIndent(proofs, "", "  ")
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(output, append(raw, '\n'), 0o600); e != nil {
			t.Fatal(e)
		}
	}
}
