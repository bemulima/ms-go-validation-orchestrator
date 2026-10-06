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

func TestT102PinnedNodeAssertionsUseActualProvider(t *testing.T) {
	fixturePath, nodeURL := os.Getenv("T102_PROVIDER_FIXTURE_FILE"), os.Getenv("T102_NODE_PROVIDER_URL")
	if fixturePath == "" {
		t.Skip("requires explicitly isolated actual Node and Sandbox providers")
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
	if metadata.URL == "" || nodeURL == "" || len(metadata.Fixtures) != 2 {
		t.Fatal("actual providers incomplete")
	}
	client := &http.Client{Timeout: 20 * time.Second}
	validChecks := json.RawMessage(`{"requests":[{"name":"status","request":{"method":"GET","path":"/status"},"expect":{"status":200,"jsonMatch":{"status":"ok"}}}]}`)
	invalidChecks := map[string]json.RawMessage{"empty_requests": json.RawMessage(`{"requests":[]}`), "empty_expect": json.RawMessage(`{"requests":[{"name":"status","request":{"method":"GET","path":"/status"},"expect":{}}]}`), "empty_body_fragment": json.RawMessage(`{"requests":[{"name":"status","request":{"method":"GET","path":"/status"},"expect":{"bodyContains":""}}]}`)}
	proof := map[string]any{"full_live_e2e": false, "boundary": "actual retained Sandbox/Tarantool/S3 immutable genesis → production Orchestrator App HTTP/pinned reader/Node adapter → actual Node DI/runtime/SandboxManager/registered HTTP; public synthetic bindings, no learner journey"}
	direct := map[string]any{}
	directCall := func(kind string, checks json.RawMessage, missing bool) (int, bool, []string) {
		t.Helper()
		value := "ok"
		if kind == "negative" {
			value = "wrong"
		}
		code := "import { createServer } from 'node:http'; createServer((req,res)=>{if(req.url!='/status'){res.writeHead(404).end();return;}res.writeHead(200,{'content-type':'application/json'});res.end(JSON.stringify({status:'" + value + "'}));}).listen(Number(process.env.PORT),'127.0.0.1');"
		rules := map[string]any{}
		if !missing {
			rules["runtime"] = checks
		}
		body, e := json.Marshal(map[string]any{"language": "js", "framework": "none", "mode": map[string]bool{"static": false, "structure": false, "runtime": true}, "code": map[string]any{"entrypoint": "index.js", "files": []map[string]string{{"path": "index.js", "content": code}}}, "rules": rules})
		if e != nil {
			t.Fatal(e)
		}
		res, e := client.Post(nodeURL+"/api/v1/validate-node", "application/json", bytes.NewReader(body))
		if e != nil {
			t.Fatal("actual Node provider unavailable")
		}
		defer res.Body.Close()
		var result struct {
			OK     bool `json:"ok"`
			Errors []struct {
				Code string `json:"code"`
			} `json:"errors"`
		}
		if e = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&result); e != nil {
			t.Fatal("invalid actual Node response")
		}
		codes := []string{}
		for _, issue := range result.Errors {
			codes = append(codes, issue.Code)
		}
		return res.StatusCode, result.OK, codes
	}
	for _, kind := range []string{"positive", "negative"} {
		t.Run("owner_"+kind, func(t *testing.T) {
			status, ok, codes := directCall(kind, validChecks, false)
			direct[kind] = map[string]any{"status": status, "ok": ok, "codes": codes}
			if status != 200 || ok != (kind == "positive") {
				t.Fatalf("actual owner %s status=%d ok=%v", kind, status, ok)
			}
			if kind == "negative" {
				found := false
				for _, code := range codes {
					found = found || code == "RUNTIME_JSON_MISMATCH"
				}
				if !found {
					t.Fatal("actual runtime expected JSON mismatch not demonstrated")
				}
			}
		})
	}
	for name, checks := range invalidChecks {
		t.Run("owner_"+name, func(t *testing.T) {
			status, ok, codes := directCall("positive", checks, false)
			direct[name] = map[string]any{"status": status, "ok": ok, "codes": codes}
			if status != 400 || ok {
				t.Fatalf("configuration should be400 not PASS: status=%d ok=%v", status, ok)
			}
		})
	}
	t.Run("owner_missing_runtime", func(t *testing.T) {
		status, ok, codes := directCall("positive", nil, true)
		direct["missing_runtime"] = map[string]any{"status": status, "ok": ok, "codes": codes}
		if status != 400 || ok {
			t.Fatalf("missing runtime criteria should be400 not PASS: status=%d ok=%v", status, ok)
		}
	})
	proof["actual_node_owner"] = direct
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	var entropy [24]byte
	if _, err = rand.Read(entropy[:]); err != nil {
		t.Fatal("generate isolated credential")
	}
	cfg.InternalAPIToken = hex.EncodeToString(entropy[:])
	cfg.Engines = config.EngineEndpoints{Node: nodeURL}
	cfg.SandboxServiceBaseURL = metadata.URL
	cfg.SandboxServiceInternalToken = os.Getenv("T102_INTERNAL_TOKEN")
	if cfg.SandboxServiceInternalToken == "" {
		t.Fatal("missing scoped Sandbox credential")
	}
	application := New(cfg)
	server := httptest.NewServer(application.Server().Handler)
	defer server.Close()
	makeSpec := func(checks json.RawMessage, optional bool) (json.RawMessage, string) {
		t.Helper()
		stage := map[string]any{"id": "http", "engine": "http.runtime", "language": "js", "framework": "none", "mode": "final", "targets": map[string]any{"entrypoint": "index.js", "files": []string{"index.js"}}, "checks": checks}
		stages := []map[string]any{stage}
		if optional {
			required := map[string]any{"id": "required-http", "engine": "http.runtime", "language": "js", "framework": "none", "mode": "final", "targets": map[string]any{"entrypoint": "index.js", "files": []string{"index.js"}}, "checks": validChecks}
			stage["optional"] = true
			stages = []map[string]any{required, stage}
		}
		contract, e := json.Marshal(map[string]any{"version": 1, "kind": "workspace_contract", "workspace": map[string]any{"required_files": []string{"index.js"}}, "stages": stages})
		if e != nil {
			t.Fatal(e)
		}
		spec, e := json.Marshal(map[string]any{"schema": domain.PracticeValidationSpecificationSchemaV1, "runtime_profile": map[string]any{}, "contracts": []map[string]any{{"milestone_id": "http", "required": true, "validation_contract": json.RawMessage(contract), "evaluation_assets": map[string]any{}}}})
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
	for _, f := range metadata.Fixtures {
		if f.Revision != 0 || f.LiveHeadRevision <= f.Revision || f.WorkspaceDigest == f.LiveHeadDigest || !f.RetainedContentUnchanged {
			t.Fatal("retained genesis/opposite live head proof absent")
		}
		spec, digest := makeSpec(validChecks, false)
		input := domain.PracticeValidationRequestV2{Schema: domain.PracticeValidationRequestSchemaV2, TaskInstanceID: f.TaskInstanceID, SubmissionID: "22222222-2222-4222-8222-222222222222", ValidationRunID: "33333333-3333-4333-8333-333333333333", RevisionDigest: "sha256:" + strings.Repeat("a", 64), ValidationContractDigest: digest, Snapshot: domain.PracticeSnapshotRefV2{SandboxID: f.SandboxID, SnapshotID: f.SnapshotID, GenerationID: f.GenerationID, WorkspaceDigest: f.WorkspaceDigest}, Scope: domain.PracticeValidationScopeFinal, ValidationSpecification: spec}
		if f.Kind == "positive" {
			positive = input
		}
		t.Run("pinned_"+f.Kind, func(t *testing.T) {
			status, result := call(input, true)
			want := domain.PracticeValidationFail
			if f.Kind == "positive" {
				want = domain.PracticeValidationPass
			}
			proof[f.Kind] = result
			if status != 200 || result.Outcome != want {
				t.Fatalf("pinned %s status=%d outcome=%s/%s want=%s", f.Kind, status, result.Outcome, result.ErrorClassification, want)
			}
			if f.Kind == "negative" {
				found := false
				for _, stage := range result.Stages {
					for _, issue := range stage.Issues {
						found = found || issue.Code == "RUNTIME_JSON_MISMATCH"
					}
				}
				if !found {
					t.Fatal("retained negative did not fail actual JSON assertion")
				}
			}
			if result.Snapshot != input.Snapshot || result.TaskInstanceID != input.TaskInstanceID || result.SubmissionID != input.SubmissionID || result.ValidationRunID != input.ValidationRunID || result.ValidationContractDigest != input.ValidationContractDigest || result.RevisionDigest != input.RevisionDigest {
				t.Fatal("correlation changed")
			}
		})
	}
	if positive.TaskInstanceID == "" {
		t.Fatal("positive fixture missing")
	}
	for name, checks := range invalidChecks {
		t.Run("pinned_"+name, func(t *testing.T) {
			input := positive
			input.ValidationSpecification, input.ValidationContractDigest = makeSpec(checks, false)
			status, result := call(input, true)
			proof[name] = result
			if status != 200 || result.Outcome != domain.PracticeValidationError || result.ErrorClassification != domain.PracticeValidationValidatorProtocol {
				t.Fatalf("invalid runtime criteria should technicalERROR: status=%d outcome=%s/%s", status, result.Outcome, result.ErrorClassification)
			}
		})
	}
	t.Run("pinned_optional_invalid", func(t *testing.T) {
		input := positive
		input.ValidationSpecification, input.ValidationContractDigest = makeSpec(invalidChecks["empty_requests"], true)
		status, result := call(input, true)
		proof["optional_invalid"] = result
		if status != 200 || result.Outcome != domain.PracticeValidationError || result.ErrorClassification != domain.PracticeValidationValidatorProtocol {
			t.Fatalf("optional technical config must not PASS: status=%d outcome=%s/%s", status, result.Outcome, result.ErrorClassification)
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
	if output := os.Getenv("T102_RESULT_FILE"); output != "" {
		raw, e := json.MarshalIndent(proof, "", "  ")
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(output, append(raw, '\n'), 0600); e != nil {
			t.Fatal(e)
		}
	}
}
