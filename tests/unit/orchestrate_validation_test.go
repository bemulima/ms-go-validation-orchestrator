package unit

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
	"github.com/example/ms-validation-orchestrator-service/internal/usecase"
)

type fakeEngine struct {
	id      string
	passed  bool
	message string
}

func (engine fakeEngine) EngineID() string {
	return engine.id
}

func (engine fakeEngine) Validate(
	_ context.Context,
	input domain.EngineValidationInput,
) (domain.StageExecutionResult, error) {
	result := domain.StageExecutionResult{
		Passed: engine.passed,
	}

	if !engine.passed {
		result.Errors = []domain.ValidationIssue{
			{
				Code:     "FAKE_FAILURE",
				Message:  engine.message,
				Severity: "error",
				StageID:  input.Stage.ID,
				Engine:   input.Stage.Engine,
			},
		}
	}

	return result, nil
}

func TestExecuteOrdersStagesAndAggregatesFailures(t *testing.T) {
	t.Parallel()

	contract := domain.ValidationContract{
		Version: 1,
		Kind:    "workspace_contract",
		Stages: []domain.ValidationStage{
			{ID: "html", Engine: "html.dom"},
			{ID: "css", Engine: "css.ast", DependsOn: []string{"html"}},
		},
	}

	codeStructure, err := json.Marshal(contract)
	if err != nil {
		t.Fatalf("marshal contract: %v", err)
	}

	parser := usecase.NewContractParser(usecase.NewDefaultLegacyContractAdapter())
	useCase := usecase.NewOrchestrateValidationUseCase(parser, []domain.EngineClient{
		fakeEngine{id: "html.dom", passed: false, message: "html failed"},
		fakeEngine{id: "css.ast", passed: true},
	})

	result, err := useCase.Execute(context.Background(), domain.ValidationRequest{
		TaskID:        "task-1",
		CodeStructure: codeStructure,
		Workspace: domain.ValidationWorkspace{
			Files: []domain.WorkspaceFile{{Path: "index.html", Content: "<html></html>"}},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if result.Passed {
		t.Fatalf("expected overall result to fail")
	}

	if len(result.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(result.Stages))
	}

	if result.Stages[0].StageID != "html" {
		t.Fatalf("expected first stage to be html, got %s", result.Stages[0].StageID)
	}

	if result.Stages[1].Status != "skipped" {
		t.Fatalf("expected css stage to be skipped, got %s", result.Stages[1].Status)
	}
}

func TestExecuteAdaptsLegacyContracts(t *testing.T) {
	t.Parallel()

	parser := usecase.NewContractParser(usecase.NewDefaultLegacyContractAdapter())
	useCase := usecase.NewOrchestrateValidationUseCase(parser, []domain.EngineClient{
		fakeEngine{id: "html.dom", passed: true},
	})

	result, err := useCase.Execute(context.Background(), domain.ValidationRequest{
		TaskID:                "task-legacy",
		CodeStructureTypeCode: "HTML_BASIC",
		CodeStructure:         json.RawMessage(`{"rules":{"doctype":true}}`),
		Workspace: domain.ValidationWorkspace{
			Files: []domain.WorkspaceFile{{Path: "index.html", Content: "<!doctype html>"}},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if !result.Legacy {
		t.Fatalf("expected legacy flag to be true")
	}

	if result.ContractKind != "legacy_contract" {
		t.Fatalf("expected legacy contract kind, got %s", result.ContractKind)
	}

	if result.Passed {
		t.Fatalf("expected legacy path to fail until integration is implemented")
	}
}

func TestExecuteFiltersStagesByMode(t *testing.T) {
	t.Parallel()

	contract := domain.ValidationContract{
		Version: 1,
		Kind:    "workspace_contract",
		Stages: []domain.ValidationStage{
			{ID: "html-live", Engine: "html.dom", Mode: domain.ValidationModeLive},
			{ID: "browser-final", Engine: "html.dom", Mode: domain.ValidationModeFinal},
		},
	}

	codeStructure, err := json.Marshal(contract)
	if err != nil {
		t.Fatalf("marshal contract: %v", err)
	}

	parser := usecase.NewContractParser(usecase.NewDefaultLegacyContractAdapter())
	useCase := usecase.NewOrchestrateValidationUseCase(parser, []domain.EngineClient{
		fakeEngine{id: "html.dom", passed: true},
	})

	result, err := useCase.Execute(context.Background(), domain.ValidationRequest{
		TaskID:        "task-live",
		Mode:          domain.ValidationModeLive,
		CodeStructure: codeStructure,
		Workspace: domain.ValidationWorkspace{
			Files: []domain.WorkspaceFile{{Path: "index.html", Content: "<html></html>"}},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if len(result.Stages) != 1 {
		t.Fatalf("expected 1 live stage, got %d", len(result.Stages))
	}

	if result.Stages[0].StageID != "html-live" {
		t.Fatalf("expected live stage to run, got %s", result.Stages[0].StageID)
	}
}
