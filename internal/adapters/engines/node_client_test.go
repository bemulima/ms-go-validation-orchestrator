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
