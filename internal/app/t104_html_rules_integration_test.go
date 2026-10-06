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

func TestT104PinnedHTMLRulesUseActualProvider(t *testing.T) {
	fixturePath, htmlURL := os.Getenv("T104_PROVIDER_FIXTURE_FILE"), os.Getenv("T104_HTML_PROVIDER_URL")
	if fixturePath == "" {
		t.Skip("requires explicitly isolated actual HTML and Sandbox providers")
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
	if metadata.URL == "" || htmlURL == "" || len(metadata.Fixtures) != 2 {
		t.Fatal("actual providers incomplete")
	}
	client := &http.Client{Timeout: 20 * time.Second}
	proof := map[string]any{"full_live_e2e": false, "boundary": "actual retained Sandbox/Tarantool/S3 immutable genesis → production Orchestrator App HTTP/pinned reader/HTML adapter → actual HTML production DI/parser/rules/registered HTTP; public synthetic bindings, no learner journey, browser or live NATS engine"}
	validRules := json.RawMessage(`{"version":2,"head":{"doctype":true,"htmlLang":"en"},"body":{"roots":[{"tag":{"value":"h1","errorMessage":"Heading must be h1"},"attributes":[{"name":"class","value":"title","errorMessage":"class required"}],"text":{"value":"Hello World!","errorMessage":"Text must be Hello World!"}}]}}`)
	invalidRules := map[string]json.RawMessage{"missing": nil, "null": json.RawMessage(`null`), "empty_object": json.RawMessage(`{}`), "zero_criteria": json.RawMessage(`{"version":2}`), "disabled_criteria": json.RawMessage(`{"version":2,"head":{"doctype":false,"htmlLang":"","head":[]},"body":{"roots":[]}}`)}
	direct := map[string]any{}
	for _, kind := range []string{"positive", "negative", "missing", "empty_object", "zero_criteria", "disabled_criteria"} {
		t.Run("owner_"+kind, func(t *testing.T) {
			code := `<!DOCTYPE html><html lang="en"><body><h1 class="title">Hello World!</h1></body></html>`
			if kind == "negative" {
				code = strings.Replace(code, "Hello World!", "Wrong output", 1)
			}
			payload := map[string]any{"code": code, "taskId": "t104-public-missing-rules"}
			if kind == "positive" || kind == "negative" {
				payload["rules"] = validRules
			} else if kind != "missing" {
				payload["rules"] = invalidRules[kind]
			}
			body, e := json.Marshal(payload)
			if e != nil {
				t.Fatal(e)
			}
			res, e := client.Post(htmlURL+"/validate", "application/json", bytes.NewReader(body))
			if e != nil {
				t.Fatal("actual HTML unavailable")
			}
			defer res.Body.Close()
			var result struct {
				Valid  *bool `json:"isValid"`
				Errors []struct {
					Code string `json:"code"`
				} `json:"errors"`
			}
			if e = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&result); e != nil {
				t.Fatal(e)
			}
			codes := []string{}
			for _, issue := range result.Errors {
				codes = append(codes, issue.Code)
			}
			direct[kind] = map[string]any{"status": res.StatusCode, "isValid": result.Valid, "codes": codes}
			if kind == "positive" || kind == "negative" {
				want := kind == "positive"
				if res.StatusCode != 200 || result.Valid == nil || *result.Valid != want {
					t.Fatalf("actual semantic status=%d valid=%v", res.StatusCode, result.Valid)
				}
				if !want && (len(codes) == 0 || codes[0] != "TEXT_MISMATCH") {
					t.Fatal("actual negative text rule absent")
				}
			} else if res.StatusCode != 400 || result.Valid != nil {
				t.Fatalf("configuration must technical400 without learner outcome status=%d", res.StatusCode)
			}
		})
	}
	proof["actual_html_owner"] = direct
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	var entropy [24]byte
	if _, err = rand.Read(entropy[:]); err != nil {
		t.Fatal("generate isolated credential")
	}
	cfg.InternalAPIToken = hex.EncodeToString(entropy[:])
	cfg.Engines = config.EngineEndpoints{HTML: htmlURL}
	cfg.SandboxServiceBaseURL = metadata.URL
	cfg.SandboxServiceInternalToken = os.Getenv("T104_INTERNAL_TOKEN")
	if cfg.SandboxServiceInternalToken == "" {
		t.Fatal("missing scoped Sandbox credential")
	}
	application := New(cfg)
	server := httptest.NewServer(application.Server().Handler)
	defer server.Close()
	makeSpec := func(rules json.RawMessage, optional bool) (json.RawMessage, string) {
		t.Helper()
		stage := map[string]any{"id": "html", "engine": "html.dom", "mode": "final", "targets": map[string]any{"files": []string{"index.html"}}}
		if rules != nil {
			stage["rules"] = rules
		}
		stages := []map[string]any{stage}
		if optional {
			stage["optional"] = true
			stages = []map[string]any{{"id": "required-html", "engine": "html.dom", "mode": "final", "targets": map[string]any{"files": []string{"index.html"}}, "rules": validRules}, stage}
		}
		contract, e := json.Marshal(map[string]any{"version": 1, "kind": "workspace_contract", "workspace": map[string]any{"required_files": []string{"index.html"}}, "stages": stages})
		if e != nil {
			t.Fatal(e)
		}
		spec, e := json.Marshal(map[string]any{"schema": domain.PracticeValidationSpecificationSchemaV1, "runtime_profile": map[string]any{}, "contracts": []map[string]any{{"milestone_id": "html", "required": true, "validation_contract": json.RawMessage(contract), "evaluation_assets": map[string]any{}}}})
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
		spec, digest := makeSpec(validRules, false)
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
						found = found || issue.Code == "TEXT_MISMATCH"
					}
				}
				if !found {
					t.Fatal("retained negative text rule absent")
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
	for _, message := range []string{"", " "} {
		t.Run("pinned_semantic_message_"+map[bool]string{true: "blank", false: "whitespace"}[message == ""], func(t *testing.T) {
			rules, e := json.Marshal(map[string]any{"version": 2, "body": map[string]any{"roots": []map[string]any{{"tag": map[string]any{"value": "main", "errorMessage": message}}}}})
			if e != nil {
				t.Fatal(e)
			}
			input := positive
			input.ValidationSpecification, input.ValidationContractDigest = makeSpec(rules, false)
			status, result := call(input, true)
			proof["semantic_message_"+map[bool]string{true: "blank", false: "whitespace"}[message == ""]] = result
			if status != 200 || result.Outcome != domain.PracticeValidationFail {
				t.Fatalf("authored string message must remain semanticFAIL status=%d outcome=%s", status, result.Outcome)
			}
			if len(result.Stages) != 1 || len(result.Stages[0].Issues) != 1 || result.Stages[0].Issues[0].Code != "TAG_MISMATCH" || result.Stages[0].Issues[0].Message != "Validation criteria failed." {
				t.Fatal("authored message projection changed")
			}
		})
	}
	for name, rules := range invalidRules {
		t.Run("pinned_"+name, func(t *testing.T) {
			input := positive
			input.ValidationSpecification, input.ValidationContractDigest = makeSpec(rules, false)
			status, result := call(input, true)
			proof[name] = result
			if status != 200 || result.Outcome != domain.PracticeValidationError || result.ErrorClassification != domain.PracticeValidationValidatorProtocol {
				t.Fatalf("missing/zero rules must technicalERROR status=%d outcome=%s/%s", status, result.Outcome, result.ErrorClassification)
			}
		})
	}
	t.Run("pinned_optional_config", func(t *testing.T) {
		input := positive
		input.ValidationSpecification, input.ValidationContractDigest = makeSpec(nil, true)
		status, result := call(input, true)
		proof["optional_config"] = result
		if status != 200 || result.Outcome != domain.PracticeValidationError || result.ErrorClassification != domain.PracticeValidationValidatorProtocol {
			t.Fatalf("optional configuration must technicalERROR status=%d outcome=%s", status, result.Outcome)
		}
	})
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
	if output := os.Getenv("T104_RESULT_FILE"); output != "" {
		raw, e := json.MarshalIndent(proof, "", "  ")
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(output, append(raw, '\n'), 0600); e != nil {
			t.Fatal(e)
		}
	}
}
