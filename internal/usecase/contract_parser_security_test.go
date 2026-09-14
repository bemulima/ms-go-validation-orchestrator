package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

func TestContractParserRejectsUnknownV1Field(t *testing.T) {
	parser := NewContractParser(NewDefaultLegacyContractAdapter())
	_, _, err := parser.Parse(domain.ValidationRequest{
		CodeStructure: json.RawMessage(`{
			"version":1,
			"kind":"workspace_contract",
			"stages":[{"id":"go","engine":"go.core"}],
			"unexpected":true
		}`),
	})
	if !errors.Is(err, domain.ErrInvalidContract) {
		t.Fatalf("expected invalid contract, got %v", err)
	}
}

func TestContractParserRejectsIncompleteV1InsteadOfAdaptingAsLegacy(t *testing.T) {
	parser := NewContractParser(NewDefaultLegacyContractAdapter())
	_, legacy, err := parser.Parse(domain.ValidationRequest{
		CodeStructure: json.RawMessage(`{"version":1,"kind":"workspace_contract","stages":[]}`),
	})
	if !errors.Is(err, domain.ErrInvalidContract) {
		t.Fatalf("expected invalid contract, got %v", err)
	}
	if legacy {
		t.Fatal("incomplete v1 contract must not fall back to legacy")
	}
}

func TestContractParserRejectsUnsafeRequiredFile(t *testing.T) {
	parser := NewContractParser(NewDefaultLegacyContractAdapter())
	_, _, err := parser.Parse(domain.ValidationRequest{
		CodeStructure: json.RawMessage(`{
			"version":1,
			"kind":"workspace_contract",
			"workspace":{"required_files":["../solution.go"]},
			"stages":[{"id":"go","engine":"go.core"}]
		}`),
	})
	if !errors.Is(err, domain.ErrInvalidContract) {
		t.Fatalf("expected invalid contract, got %v", err)
	}
}

func TestExecuteRejectsMissingRequiredFileInInlineWorkspace(t *testing.T) {
	useCase := NewOrchestrateValidationUseCase(
		NewContractParser(NewDefaultLegacyContractAdapter()),
		[]domain.EngineClient{capabilityTestEngine{id: "go.core"}},
	)
	_, err := useCase.Execute(context.Background(), domain.ValidationRequest{
		CodeStructure: json.RawMessage(`{
			"version":1,
			"kind":"workspace_contract",
			"workspace":{"required_files":["go.mod","main.go"]},
			"stages":[{"id":"go","engine":"go.core"}]
		}`),
		Workspace: domain.ValidationWorkspace{
			Files: []domain.WorkspaceFile{{Path: "main.go", Content: "package main"}},
		},
	})
	if !errors.Is(err, domain.ErrInvalidRequest) {
		t.Fatalf("expected invalid request, got %v", err)
	}
}
