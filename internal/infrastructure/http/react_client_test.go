package engines

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type fakeReactHTTPClient struct {
	responseBody []byte
	lastURL      string
	requestBody  []byte
}

func (client *fakeReactHTTPClient) PostJSON(
	_ context.Context,
	url string,
	payload any,
) ([]byte, error) {
	client.lastURL = url
	var err error
	client.requestBody, err = json.Marshal(payload)
	if err != nil {
		return nil, err
	}
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

func TestReactClientPreservesAuthoredRulesArrayInOutboundJSON(t *testing.T) {
	rules := json.RawMessage(`[
		{"kind":"component_exists","name":"App","export":"default","componentKind":"function"},
		{"kind":"jsx_tree","component":"App","selector":"button","required":true,"textIncludes":"Save","attributesIncludes":{"id":"save-button"},"children":[{"selector":"span","textIncludes":"Saved"}]}
	]`)
	code := `export default function App(){ return <button id="save-button">Save<span>Saved</span></button>; }`
	poster := &fakeReactHTTPClient{responseBody: []byte(`{"success":true,"isValid":true,"errors":[]}`)}
	_, err := NewReactClient("http://react.test", poster).Validate(context.Background(), domain.EngineValidationInput{
		TaskID:    "synthetic-task",
		Stage:     domain.ValidationStage{ID: "react", Engine: "react.ast", Language: "tsx", Framework: "react", Targets: domain.StageTargets{Files: []string{"src/App.tsx"}}, Rules: rules},
		Workspace: domain.ValidationWorkspace{Files: []domain.WorkspaceFile{{Path: "src/App.tsx", Content: code}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		TaskID, Code, Language, Framework string
		Rules                             json.RawMessage
		Meta                              struct{ Path string }
	}
	if err := json.Unmarshal(poster.requestBody, &body); err != nil {
		t.Fatal(err)
	}
	if poster.lastURL != "http://react.test/validate" || body.TaskID != "synthetic-task" || body.Code != code || body.Language != "tsx" || body.Framework != "react" || body.Meta.Path != "src/App.tsx" {
		t.Fatal("React HTTP routing or exact target payload changed")
	}
	var expected, actual []any
	if err := json.Unmarshal(rules, &expected); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body.Rules, &actual); err != nil {
		t.Fatal(err)
	}
	if len(actual) != 2 || !reflect.DeepEqual(actual, expected) {
		t.Fatal("authored Rules array order, fields or nested values changed")
	}
}
