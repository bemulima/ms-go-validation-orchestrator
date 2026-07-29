package engines

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type fakeNodeHTTPClient struct {
	lastURL     string
	lastPayload map[string]any
	response    []byte
}

func (client *fakeNodeHTTPClient) PostJSON(
	_ context.Context,
	url string,
	payload any,
) ([]byte, error) {
	client.lastURL = url
	if mapped, ok := payload.(map[string]any); ok {
		client.lastPayload = mapped
	}

	return client.response, nil
}

func TestNodeClientParsesNativeValidationResponse(t *testing.T) {
	t.Parallel()

	httpClient := &fakeNodeHTTPClient{
		response: []byte(`{
			"ok": false,
			"summary": {"staticOk": true, "structureOk": false, "runtimeOk": true},
			"errors": [
				{
					"code": "EXPRESS_ROUTE_MISSING",
					"level": "error",
					"message": "Express route not found: GET /health",
					"location": {"file": "src/main.ts", "line": 12, "column": 5},
					"meta": {"route": "GET /health", "hint": "define app.get('/health', ...)"}
				}
			]
		}`),
	}

	client := NewNodeClient("http://node-validator", httpClient, "node.express")
	result, err := client.Validate(context.Background(), domain.EngineValidationInput{
		TaskID: "task-1",
		Stage: domain.ValidationStage{
			ID:     "backend",
			Engine: "node.express",
			Targets: domain.StageTargets{
				Entrypoint: "src/main.ts",
			},
		},
		Workspace: domain.ValidationWorkspace{
			Files: []domain.WorkspaceFile{
				{Path: "src/main.ts", Content: "import express from 'express';"},
			},
		},
	})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}

	if result.Passed {
		t.Fatalf("expected failed result")
	}

	if len(result.Errors) != 1 {
		t.Fatalf("expected one error, got %d", len(result.Errors))
	}

	issue := result.Errors[0]
	if issue.File != "src/main.ts" || issue.Line != 12 || issue.Column != 5 {
		t.Fatalf("unexpected location: %+v", issue)
	}

	if issue.Route != "GET /health" {
		t.Fatalf("expected route to be propagated, got %q", issue.Route)
	}

	if issue.Hint != "define app.get('/health', ...)" {
		t.Fatalf("expected hint to be propagated, got %q", issue.Hint)
	}
}

func TestNodeClientSelectsModesForHTTPRuntime(t *testing.T) {
	t.Parallel()

	httpClient := &fakeNodeHTTPClient{
		response: []byte(`{"ok": true, "summary": {"staticOk": true, "structureOk": true, "runtimeOk": true}, "errors": []}`),
	}

	client := NewNodeClient("http://node-validator", httpClient, "http.runtime")
	_, err := client.Validate(context.Background(), domain.EngineValidationInput{
		TaskID: "task-runtime",
		Stage: domain.ValidationStage{
			ID:        "http",
			Engine:    "http.runtime",
			Framework: "express",
			Targets: domain.StageTargets{
				Entrypoint: "src/main.ts",
			},
			Checks: json.RawMessage(`{"requests":[{"name":"health","request":{"method":"GET","path":"/health"},"expect":{"status":200}}]}`),
		},
		Workspace: domain.ValidationWorkspace{
			Files: []domain.WorkspaceFile{
				{Path: "src/main.ts", Content: "console.log('server');"},
			},
		},
	})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}

	mode, ok := httpClient.lastPayload["mode"].(map[string]bool)
	if !ok {
		t.Fatalf("expected mode payload, got %#v", httpClient.lastPayload["mode"])
	}

	if mode["static"] || mode["structure"] || !mode["runtime"] {
		t.Fatalf("unexpected runtime stage mode: %#v", mode)
	}

	framework, ok := httpClient.lastPayload["framework"].(string)
	if !ok || framework != "express" {
		t.Fatalf("expected framework hint to be preserved, got %#v", httpClient.lastPayload["framework"])
	}
}

func TestNodeClientOmitsRuntimeRulesForStaticStage(t *testing.T) {
	t.Parallel()

	httpClient := &fakeNodeHTTPClient{
		response: []byte(`{"ok": true, "summary": {"staticOk": true, "structureOk": true, "runtimeOk": true}, "errors": []}`),
	}
	client := NewNodeClient("http://node-validator", httpClient, "ts.ast")
	_, err := client.Validate(context.Background(), domain.EngineValidationInput{
		TaskID: "task-ts-static",
		Stage: domain.ValidationStage{
			ID:     "typescript-static",
			Engine: "ts.ast",
			Targets: domain.StageTargets{
				Entrypoint: "src/main.ts",
			},
			Rules: json.RawMessage(`{"maxFileSizeKb":64}`),
		},
		Workspace: domain.ValidationWorkspace{
			Files: []domain.WorkspaceFile{{Path: "src/main.ts", Content: "console.log('ok');"}},
		},
	})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}

	rules, ok := httpClient.lastPayload["rules"].(map[string]any)
	if !ok {
		t.Fatalf("expected rules payload, got %#v", httpClient.lastPayload["rules"])
	}
	if _, exists := rules["runtime"]; exists {
		t.Fatalf("expected static stage to omit runtime rules, got %#v", rules)
	}
}

func TestNodeClientNormalizesJavaScriptAuthoringLanguage(t *testing.T) {
	t.Parallel()

	httpClient := &fakeNodeHTTPClient{
		response: []byte(`{"ok": true, "summary": {"staticOk": true, "structureOk": true, "runtimeOk": true}, "errors": []}`),
	}
	client := NewNodeClient("http://node-validator", httpClient, "js.ast")
	_, err := client.Validate(context.Background(), domain.EngineValidationInput{
		Stage: domain.ValidationStage{
			ID:       "javascript-static",
			Engine:   "js.ast",
			Language: "javascript",
			Targets: domain.StageTargets{
				Entrypoint: "main.js",
			},
		},
		Workspace: domain.ValidationWorkspace{
			Files: []domain.WorkspaceFile{{Path: "main.js", Content: "console.log('ok');"}},
		},
	})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}

	if got := httpClient.lastPayload["language"]; got != "js" {
		t.Fatalf("expected JavaScript alias to normalize to js, got %#v", got)
	}
}

func TestNodeClientMapsTypeScriptRuntimeContractAndFailure(t *testing.T) {
	t.Parallel()

	httpClient := &fakeNodeHTTPClient{
		response: []byte(`{
			"ok": false,
			"summary": {"staticOk": true, "structureOk": true, "runtimeOk": false},
			"errors": [
				{
					"code": "RUNTIME_STDOUT_MISSING",
					"level": "error",
					"message": "Program stdout is missing the expected fragment.",
					"location": {"file": "src/main.ts"},
					"meta": {"hint": "Print the requested greeting."}
				}
			]
		}`),
	}

	checks := json.RawMessage(`{
		"kind":"cli",
		"args":["Ada"],
		"timeoutMs":2500,
		"expect":{"exitCode":0,"stdoutContains":["Hello, Ada!"]}
	}`)
	client := NewNodeClient("http://node-validator", httpClient, "ts.runtime")
	result, err := client.Validate(context.Background(), domain.EngineValidationInput{
		TaskID: "task-ts-runtime",
		Stage: domain.ValidationStage{
			ID:     "typescript-runtime",
			Engine: "ts.runtime",
			Targets: domain.StageTargets{
				Entrypoint: "src/main.ts",
			},
			Checks: checks,
		},
		Workspace: domain.ValidationWorkspace{
			Files: []domain.WorkspaceFile{
				{Path: "src/main.ts", Content: "console.log('TODO');"},
			},
		},
	})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}

	if result.Passed {
		t.Fatalf("expected failed runtime result")
	}
	if len(result.Errors) != 1 || result.Errors[0].Hint != "Print the requested greeting." {
		t.Fatalf("expected normalized runtime hint, got %+v", result.Errors)
	}

	if got := httpClient.lastPayload["language"]; got != "ts" {
		t.Fatalf("expected TypeScript language, got %#v", got)
	}
	if got := httpClient.lastPayload["framework"]; got != "none" {
		t.Fatalf("expected no framework, got %#v", got)
	}

	mode, ok := httpClient.lastPayload["mode"].(map[string]bool)
	if !ok || mode["static"] || mode["structure"] || !mode["runtime"] {
		t.Fatalf("unexpected TypeScript runtime mode: %#v", httpClient.lastPayload["mode"])
	}

	code, ok := httpClient.lastPayload["code"].(map[string]any)
	if !ok || code["entrypoint"] != "src/main.ts" {
		t.Fatalf("expected entrypoint to be preserved, got %#v", httpClient.lastPayload["code"])
	}
	files, ok := code["files"].([]map[string]string)
	if !ok || len(files) != 1 || files[0]["path"] != "src/main.ts" {
		t.Fatalf("expected workspace files to be preserved, got %#v", code["files"])
	}

	rules, ok := httpClient.lastPayload["rules"].(map[string]any)
	if !ok {
		t.Fatalf("expected rules payload, got %#v", httpClient.lastPayload["rules"])
	}
	runtimeRules, ok := rules["runtime"].(map[string]any)
	if !ok || runtimeRules["kind"] != "cli" || runtimeRules["timeoutMs"] != float64(2500) {
		t.Fatalf("expected runtime checks to be preserved, got %#v", rules["runtime"])
	}
}
