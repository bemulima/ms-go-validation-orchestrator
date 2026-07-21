package engines

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type fakeFoundationHTTPClient struct {
	lastURL     string
	lastPayload map[string]any
	response    []byte
}

func TestWorkspaceFoundationClientForwardsJavaChecksAndDiagnostics(t *testing.T) {
	t.Parallel()

	httpClient := &fakeFoundationHTTPClient{
		response: []byte(`{
			"ok": false,
			"isValid": false,
			"errors": [{
				"code": "JAVA_COMPILE_ERROR",
				"message": "compiler.err.expected: ';'",
				"file": "Main.java",
				"line": 3,
				"column": 27,
				"hint": "Fix this compiler diagnostic in Main.java."
			}]
		}`),
	}
	client := NewWorkspaceFoundationClient(
		"http://code-validator",
		httpClient,
		"java.runtime",
	)

	result, err := client.Validate(context.Background(), domain.EngineValidationInput{
		TaskID: "java-task",
		Mode:   domain.ValidationModeFinal,
		Stage: domain.ValidationStage{
			ID:       "java-runtime",
			Engine:   "java.runtime",
			Language: "java",
			Mode:     domain.ValidationModeFinal,
			Targets: domain.StageTargets{
				Files:      []string{"Main.java"},
				Entrypoint: "Main.java",
			},
			Rules: json.RawMessage(`{}`),
			Checks: json.RawMessage(`{
				"kind":"cli",
				"expect":{"exitCode":0,"stdoutEquals":"Hello!\\n"}
			}`),
		},
		Workspace: domain.ValidationWorkspace{Files: []domain.WorkspaceFile{{
			Path:    "Main.java",
			Content: "public class Main {}",
		}}},
	})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if result.Passed || len(result.Errors) != 1 {
		t.Fatalf("expected one normalized Java diagnostic, got %+v", result)
	}
	issue := result.Errors[0]
	if issue.Code != "JAVA_COMPILE_ERROR" || issue.File != "Main.java" ||
		issue.Line != 3 || issue.Column != 27 || issue.Hint == "" {
		t.Fatalf("unexpected normalized Java diagnostic: %+v", issue)
	}

	stagePayload, ok := httpClient.lastPayload["stage"].(map[string]any)
	if !ok {
		t.Fatalf("expected stage payload, got %#v", httpClient.lastPayload["stage"])
	}
	checks, ok := stagePayload["checks"].(map[string]any)
	if !ok || checks["kind"] != "cli" {
		t.Fatalf("expected constrained Java CLI checks, got %#v", stagePayload["checks"])
	}
	if httpClient.lastURL != "http://code-validator/api/v1/validate" {
		t.Fatalf("unexpected Java validator URL %q", httpClient.lastURL)
	}
}

func TestWorkspaceFoundationClientForwardsKotlinChecksAndDiagnostics(t *testing.T) {
	t.Parallel()

	httpClient := &fakeFoundationHTTPClient{
		response: []byte(`{
			"ok": false,
			"isValid": false,
			"errors": [{
				"code": "KOTLIN_COMPILE_ERROR",
				"message": "unresolved reference 'prntln'.",
				"file": "Main.kt",
				"line": 3,
				"column": 9,
				"hint": "Fix this compiler diagnostic in Main.kt."
			}]
		}`),
	}
	client := NewWorkspaceFoundationClient(
		"http://code-validator",
		httpClient,
		"kotlin.runtime",
	)

	result, err := client.Validate(context.Background(), domain.EngineValidationInput{
		TaskID: "kotlin-task",
		Mode:   domain.ValidationModeFinal,
		Stage: domain.ValidationStage{
			ID:       "kotlin-runtime",
			Engine:   "kotlin.runtime",
			Language: "kotlin",
			Mode:     domain.ValidationModeFinal,
			Targets: domain.StageTargets{
				Files:      []string{"Main.kt"},
				Entrypoint: "Main.kt",
			},
			Rules: json.RawMessage(`{}`),
			Checks: json.RawMessage(`{
				"kind":"cli",
				"expect":{"exitCode":0,"stdoutEquals":"Hello!\n"}
			}`),
		},
		Workspace: domain.ValidationWorkspace{Files: []domain.WorkspaceFile{{
			Path:    "Main.kt",
			Content: "fun main() {}",
		}}},
	})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if result.Passed || len(result.Errors) != 1 {
		t.Fatalf("expected one normalized Kotlin diagnostic, got %+v", result)
	}
	issue := result.Errors[0]
	if issue.Code != "KOTLIN_COMPILE_ERROR" || issue.File != "Main.kt" ||
		issue.Line != 3 || issue.Column != 9 || issue.Hint == "" {
		t.Fatalf("unexpected normalized Kotlin diagnostic: %+v", issue)
	}

	stagePayload, ok := httpClient.lastPayload["stage"].(map[string]any)
	if !ok {
		t.Fatalf("expected stage payload, got %#v", httpClient.lastPayload["stage"])
	}
	checks, ok := stagePayload["checks"].(map[string]any)
	if !ok || checks["kind"] != "cli" {
		t.Fatalf("expected constrained Kotlin CLI checks, got %#v", stagePayload["checks"])
	}
	if httpClient.lastURL != "http://code-validator/api/v1/validate" {
		t.Fatalf("unexpected Kotlin validator URL %q", httpClient.lastURL)
	}
}

func (client *fakeFoundationHTTPClient) PostJSON(
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

func TestWorkspaceFoundationClientBuildsExpectedPayload(t *testing.T) {
	t.Parallel()

	httpClient := &fakeFoundationHTTPClient{
		response: []byte(`{
			"ok": false,
			"errors": [
				{
					"code": "NEXTJS_PAGE_MISSING",
					"message": "Required page app/page.tsx is missing.",
					"file": "app/page.tsx",
					"hint": "Create app/page.tsx."
				}
			],
			"warnings": [
				{
					"code": "NEXTJS_API_OPTIONAL",
					"message": "API route is optional for this foundation.",
					"severity": "warning"
				}
			],
			"evidence": [
				{
					"file": "app/page.tsx",
					"message": "checked page entry"
				}
			]
		}`),
	}

	client := NewWorkspaceFoundationClient(
		"http://future-engine",
		httpClient,
		"nextjs.app",
	)

	result, err := client.Validate(context.Background(), domain.EngineValidationInput{
		TaskID: "task-1",
		Stage: domain.ValidationStage{
			ID:             "nextjs-app",
			Name:           "Next.js app foundation",
			Engine:         "nextjs.app",
			Language:       "ts",
			Framework:      "nextjs",
			DependsOn:      []string{"react"},
			TimeoutSeconds: 45,
			Targets: domain.StageTargets{
				Files:      []string{"app/page.tsx", "app/api/health/route.ts"},
				Entrypoint: "app/page.tsx",
			},
			Optional: false,
		},
		Workspace: domain.ValidationWorkspace{
			Files: []domain.WorkspaceFile{
				{Path: "app/page.tsx", Content: "export default function Page(){ return null }"},
			},
		},
		TaskMetadata: domain.TaskMetadata{
			TaskKind:      "frontend_preview",
			ExecutionMode: "WEB_BROWSER",
		},
		Locale: "ru",
		Mode:   "final",
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

	if len(result.Warnings) != 1 {
		t.Fatalf("expected one warning, got %d", len(result.Warnings))
	}

	if len(result.Evidence) != 1 {
		t.Fatalf("expected one evidence item, got %d", len(result.Evidence))
	}

	if httpClient.lastURL != "http://future-engine/api/v1/validate" {
		t.Fatalf("unexpected URL %q", httpClient.lastURL)
	}

	stagePayload, ok := httpClient.lastPayload["stage"].(map[string]any)
	if !ok {
		t.Fatalf("expected stage payload, got %#v", httpClient.lastPayload["stage"])
	}

	if stagePayload["engine"] != "nextjs.app" {
		t.Fatalf("unexpected engine %#v", stagePayload["engine"])
	}

	if stagePayload["timeoutSeconds"] != 45 {
		t.Fatalf("expected timeoutSeconds, got %#v", stagePayload["timeoutSeconds"])
	}

	taskMetadata, ok := httpClient.lastPayload["taskMetadata"].(domain.TaskMetadata)
	if !ok {
		t.Fatalf("expected task metadata payload, got %#v", httpClient.lastPayload["taskMetadata"])
	}

	if taskMetadata.ExecutionMode != "WEB_BROWSER" {
		t.Fatalf("unexpected task metadata %#v", taskMetadata)
	}
}
