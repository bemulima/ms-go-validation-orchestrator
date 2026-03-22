package engines

import (
	"context"
	"testing"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type fakeFoundationHTTPClient struct {
	lastURL     string
	lastPayload map[string]any
	response    []byte
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
