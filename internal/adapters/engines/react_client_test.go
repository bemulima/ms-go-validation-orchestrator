package engines

import (
	"context"
	"testing"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type fakeReactHTTPClient struct {
	responseBody []byte
}

func (client *fakeReactHTTPClient) PostJSON(
	_ context.Context,
	_ string,
	_ any,
) ([]byte, error) {
	return client.responseBody, nil
}

func TestReactClientValidateMapsHTTPContract(t *testing.T) {
	httpClient := &fakeReactHTTPClient{
		responseBody: []byte(`{
			"success": false,
			"isValid": false,
			"errors": [
				{
					"code": "REACT_JSX_NODE_NOT_FOUND",
					"message": "JSX node button was not found in component App.",
					"file": "src/App.tsx",
					"line": 3,
					"column": 5,
					"selector": "button",
					"symbol": "App",
					"hint": "Render button in the component tree."
				}
			]
		}`),
	}
	client := NewReactClient("http://react-validator", httpClient)

	result, err := client.Validate(context.Background(), domain.EngineValidationInput{
		TaskID: "task-1",
		Stage: domain.ValidationStage{
			ID:        "react",
			Engine:    "react.ast",
			Language:  "tsx",
			Framework: "react",
			Targets: domain.StageTargets{
				Files: []string{"src/App.tsx"},
			},
		},
		Workspace: domain.ValidationWorkspace{
			Files: []domain.WorkspaceFile{
				{Path: "src/App.tsx", Content: "export default function App(){ return <div/> }"},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Fatalf("expected validation to fail")
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected one error, got %d", len(result.Errors))
	}
	if result.Errors[0].File != "src/App.tsx" {
		t.Fatalf("expected file to be forwarded")
	}
	if result.Errors[0].Selector != "button" {
		t.Fatalf("expected selector to be forwarded")
	}
}
