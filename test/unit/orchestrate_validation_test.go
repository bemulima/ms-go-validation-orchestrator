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

type trackingEngine struct {
	id     string
	passed bool
	calls  *[]string
}

func (engine trackingEngine) EngineID() string {
	return engine.id
}

func (engine trackingEngine) Validate(
	_ context.Context,
	input domain.EngineValidationInput,
) (domain.StageExecutionResult, error) {
	*engine.calls = append(*engine.calls, input.Stage.ID)
	return domain.StageExecutionResult{Passed: engine.passed}, nil
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

func TestExecuteTypeScriptCompositeByMode(t *testing.T) {
	t.Parallel()

	contract := domain.ValidationContract{
		Version: 1,
		Kind:    "workspace_contract",
		Stages: []domain.ValidationStage{
			{ID: "typescript-static", Engine: "ts.ast", Mode: domain.ValidationModeBoth},
			{
				ID:        "typescript-runtime",
				Engine:    "ts.runtime",
				Mode:      domain.ValidationModeFinal,
				DependsOn: []string{"typescript-static"},
			},
		},
	}
	codeStructure, err := json.Marshal(contract)
	if err != nil {
		t.Fatalf("marshal contract: %v", err)
	}

	parser := usecase.NewContractParser(usecase.NewDefaultLegacyContractAdapter())
	request := domain.ValidationRequest{
		TaskID:        "task-ts-runtime",
		CodeStructure: codeStructure,
		Workspace: domain.ValidationWorkspace{
			Files: []domain.WorkspaceFile{{Path: "src/main.ts", Content: "console.log('ok');"}},
		},
	}

	liveCalls := []string{}
	liveUseCase := usecase.NewOrchestrateValidationUseCase(parser, []domain.EngineClient{
		trackingEngine{id: "ts.ast", passed: true, calls: &liveCalls},
		trackingEngine{id: "ts.runtime", passed: true, calls: &liveCalls},
	})
	request.Mode = domain.ValidationModeLive
	liveResult, err := liveUseCase.Execute(context.Background(), request)
	if err != nil {
		t.Fatalf("execute live: %v", err)
	}
	if !liveResult.Passed || len(liveCalls) != 1 || liveCalls[0] != "typescript-static" {
		t.Fatalf("expected only TypeScript static stage during live validation, calls=%v result=%+v", liveCalls, liveResult)
	}

	finalCalls := []string{}
	finalUseCase := usecase.NewOrchestrateValidationUseCase(parser, []domain.EngineClient{
		trackingEngine{id: "ts.ast", passed: true, calls: &finalCalls},
		trackingEngine{id: "ts.runtime", passed: true, calls: &finalCalls},
	})
	request.Mode = domain.ValidationModeFinal
	finalResult, err := finalUseCase.Execute(context.Background(), request)
	if err != nil {
		t.Fatalf("execute final: %v", err)
	}
	if !finalResult.Passed || len(finalCalls) != 2 {
		t.Fatalf("expected both TypeScript stages during final validation, calls=%v result=%+v", finalCalls, finalResult)
	}
	if finalCalls[0] != "typescript-static" || finalCalls[1] != "typescript-runtime" {
		t.Fatalf("expected dependency order, got %v", finalCalls)
	}
}

func TestExecuteNeverRunsTypeScriptRuntimeDuringLiveValidation(t *testing.T) {
	t.Parallel()

	contract := domain.ValidationContract{
		Version: 1,
		Kind:    "workspace_contract",
		Stages: []domain.ValidationStage{
			{ID: "runtime", Engine: "ts.runtime", Mode: domain.ValidationModeBoth},
		},
	}
	codeStructure, err := json.Marshal(contract)
	if err != nil {
		t.Fatalf("marshal contract: %v", err)
	}

	calls := []string{}
	parser := usecase.NewContractParser(usecase.NewDefaultLegacyContractAdapter())
	useCase := usecase.NewOrchestrateValidationUseCase(parser, []domain.EngineClient{
		trackingEngine{id: "ts.runtime", passed: true, calls: &calls},
	})
	result, err := useCase.Execute(context.Background(), domain.ValidationRequest{
		TaskID:        "task-ts-live-safety",
		Mode:          domain.ValidationModeLive,
		CodeStructure: codeStructure,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !result.Passed || len(calls) != 0 || len(result.Stages) != 0 {
		t.Fatalf("expected TypeScript runtime to be skipped during live validation, calls=%v result=%+v", calls, result)
	}
}

func TestExecuteFailsClosedWhenTypeScriptRuntimeIsUnavailable(t *testing.T) {
	t.Parallel()

	contract := domain.ValidationContract{
		Version: 1,
		Kind:    "workspace_contract",
		Stages: []domain.ValidationStage{
			{ID: "runtime", Engine: "ts.runtime", Mode: domain.ValidationModeFinal},
		},
	}
	codeStructure, err := json.Marshal(contract)
	if err != nil {
		t.Fatalf("marshal contract: %v", err)
	}

	parser := usecase.NewContractParser(usecase.NewDefaultLegacyContractAdapter())
	useCase := usecase.NewOrchestrateValidationUseCase(parser, nil)
	result, err := useCase.Execute(context.Background(), domain.ValidationRequest{
		TaskID:        "task-ts-runtime-unavailable",
		Mode:          domain.ValidationModeFinal,
		CodeStructure: codeStructure,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Passed || len(result.Stages) != 1 || result.Stages[0].Status != "failed" {
		t.Fatalf("expected unavailable TypeScript runtime to fail closed, got %+v", result)
	}
}

func TestExecuteJavaCompositeByMode(t *testing.T) {
	t.Parallel()

	contract := domain.ValidationContract{
		Version: 1,
		Kind:    "workspace_contract",
		Stages: []domain.ValidationStage{
			{ID: "java-compile", Engine: "java.compile", Mode: domain.ValidationModeBoth},
			{
				ID:        "java-runtime",
				Engine:    "java.runtime",
				Mode:      domain.ValidationModeFinal,
				DependsOn: []string{"java-compile"},
			},
		},
	}
	codeStructure, err := json.Marshal(contract)
	if err != nil {
		t.Fatalf("marshal contract: %v", err)
	}

	request := domain.ValidationRequest{
		TaskID:        "task-java-runtime",
		CodeStructure: codeStructure,
		Workspace: domain.ValidationWorkspace{
			Files: []domain.WorkspaceFile{{Path: "Main.java", Content: "public class Main {}"}},
		},
	}
	parser := usecase.NewContractParser(usecase.NewDefaultLegacyContractAdapter())

	liveCalls := []string{}
	liveUseCase := usecase.NewOrchestrateValidationUseCase(parser, []domain.EngineClient{
		trackingEngine{id: "java.compile", passed: true, calls: &liveCalls},
		trackingEngine{id: "java.runtime", passed: true, calls: &liveCalls},
	})
	request.Mode = domain.ValidationModeLive
	liveResult, err := liveUseCase.Execute(context.Background(), request)
	if err != nil {
		t.Fatalf("execute live: %v", err)
	}
	if !liveResult.Passed || len(liveCalls) != 1 || liveCalls[0] != "java-compile" {
		t.Fatalf("expected only Java compile during live validation, calls=%v result=%+v", liveCalls, liveResult)
	}

	finalCalls := []string{}
	finalUseCase := usecase.NewOrchestrateValidationUseCase(parser, []domain.EngineClient{
		trackingEngine{id: "java.compile", passed: true, calls: &finalCalls},
		trackingEngine{id: "java.runtime", passed: true, calls: &finalCalls},
	})
	request.Mode = domain.ValidationModeFinal
	finalResult, err := finalUseCase.Execute(context.Background(), request)
	if err != nil {
		t.Fatalf("execute final: %v", err)
	}
	if !finalResult.Passed || len(finalCalls) != 2 ||
		finalCalls[0] != "java-compile" || finalCalls[1] != "java-runtime" {
		t.Fatalf("expected Java compile then runtime, calls=%v result=%+v", finalCalls, finalResult)
	}
}

func TestExecuteNeverRunsJavaRuntimeDuringLiveValidation(t *testing.T) {
	t.Parallel()

	contract := domain.ValidationContract{
		Version: 1,
		Kind:    "workspace_contract",
		Stages: []domain.ValidationStage{
			{ID: "runtime", Engine: "java.runtime", Mode: domain.ValidationModeBoth},
		},
	}
	codeStructure, err := json.Marshal(contract)
	if err != nil {
		t.Fatalf("marshal contract: %v", err)
	}

	calls := []string{}
	parser := usecase.NewContractParser(usecase.NewDefaultLegacyContractAdapter())
	useCase := usecase.NewOrchestrateValidationUseCase(parser, []domain.EngineClient{
		trackingEngine{id: "java.runtime", passed: true, calls: &calls},
	})
	result, err := useCase.Execute(context.Background(), domain.ValidationRequest{
		TaskID:        "task-java-live-safety",
		Mode:          domain.ValidationModeLive,
		CodeStructure: codeStructure,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !result.Passed || len(calls) != 0 || len(result.Stages) != 0 {
		t.Fatalf("expected Java runtime to be skipped during live validation, calls=%v result=%+v", calls, result)
	}
}

func TestExecuteFailsClosedWhenJavaRuntimeIsUnavailable(t *testing.T) {
	t.Parallel()

	contract := domain.ValidationContract{
		Version: 1,
		Kind:    "workspace_contract",
		Stages: []domain.ValidationStage{
			{ID: "runtime", Engine: "java.runtime", Mode: domain.ValidationModeFinal},
		},
	}
	codeStructure, err := json.Marshal(contract)
	if err != nil {
		t.Fatalf("marshal contract: %v", err)
	}

	parser := usecase.NewContractParser(usecase.NewDefaultLegacyContractAdapter())
	useCase := usecase.NewOrchestrateValidationUseCase(parser, nil)
	result, err := useCase.Execute(context.Background(), domain.ValidationRequest{
		TaskID:        "task-java-runtime-unavailable",
		Mode:          domain.ValidationModeFinal,
		CodeStructure: codeStructure,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Passed || len(result.Stages) != 1 || result.Stages[0].Status != "failed" {
		t.Fatalf("expected unavailable Java runtime to fail closed, got %+v", result)
	}
}

func TestExecuteKotlinCompositeByMode(t *testing.T) {
	t.Parallel()

	contract := domain.ValidationContract{
		Version: 1,
		Kind:    "workspace_contract",
		Stages: []domain.ValidationStage{
			{ID: "kotlin-compile", Engine: "kotlin.compile", Mode: domain.ValidationModeBoth},
			{
				ID:        "kotlin-runtime",
				Engine:    "kotlin.runtime",
				Mode:      domain.ValidationModeFinal,
				DependsOn: []string{"kotlin-compile"},
			},
		},
	}
	codeStructure, err := json.Marshal(contract)
	if err != nil {
		t.Fatalf("marshal contract: %v", err)
	}

	request := domain.ValidationRequest{
		TaskID:        "task-kotlin-runtime",
		CodeStructure: codeStructure,
		Workspace: domain.ValidationWorkspace{
			Files: []domain.WorkspaceFile{{Path: "Main.kt", Content: "fun main() {}"}},
		},
	}
	parser := usecase.NewContractParser(usecase.NewDefaultLegacyContractAdapter())

	liveCalls := []string{}
	liveUseCase := usecase.NewOrchestrateValidationUseCase(parser, []domain.EngineClient{
		trackingEngine{id: "kotlin.compile", passed: true, calls: &liveCalls},
		trackingEngine{id: "kotlin.runtime", passed: true, calls: &liveCalls},
	})
	request.Mode = domain.ValidationModeLive
	liveResult, err := liveUseCase.Execute(context.Background(), request)
	if err != nil {
		t.Fatalf("execute live: %v", err)
	}
	if !liveResult.Passed || len(liveCalls) != 1 || liveCalls[0] != "kotlin-compile" {
		t.Fatalf("expected only Kotlin compile during live validation, calls=%v result=%+v", liveCalls, liveResult)
	}

	finalCalls := []string{}
	finalUseCase := usecase.NewOrchestrateValidationUseCase(parser, []domain.EngineClient{
		trackingEngine{id: "kotlin.compile", passed: true, calls: &finalCalls},
		trackingEngine{id: "kotlin.runtime", passed: true, calls: &finalCalls},
	})
	request.Mode = domain.ValidationModeFinal
	finalResult, err := finalUseCase.Execute(context.Background(), request)
	if err != nil {
		t.Fatalf("execute final: %v", err)
	}
	if !finalResult.Passed || len(finalCalls) != 2 ||
		finalCalls[0] != "kotlin-compile" || finalCalls[1] != "kotlin-runtime" {
		t.Fatalf("expected Kotlin compile then runtime, calls=%v result=%+v", finalCalls, finalResult)
	}
}

func TestExecuteNeverRunsKotlinRuntimeDuringLiveValidation(t *testing.T) {
	t.Parallel()

	contract := domain.ValidationContract{
		Version: 1,
		Kind:    "workspace_contract",
		Stages: []domain.ValidationStage{
			{ID: "runtime", Engine: "kotlin.runtime", Mode: domain.ValidationModeBoth},
		},
	}
	codeStructure, err := json.Marshal(contract)
	if err != nil {
		t.Fatalf("marshal contract: %v", err)
	}

	calls := []string{}
	parser := usecase.NewContractParser(usecase.NewDefaultLegacyContractAdapter())
	useCase := usecase.NewOrchestrateValidationUseCase(parser, []domain.EngineClient{
		trackingEngine{id: "kotlin.runtime", passed: true, calls: &calls},
	})
	result, err := useCase.Execute(context.Background(), domain.ValidationRequest{
		TaskID:        "task-kotlin-live-safety",
		Mode:          domain.ValidationModeLive,
		CodeStructure: codeStructure,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !result.Passed || len(calls) != 0 || len(result.Stages) != 0 {
		t.Fatalf("expected Kotlin runtime to be skipped during live validation, calls=%v result=%+v", calls, result)
	}
}

func TestExecuteFailsClosedWhenKotlinRuntimeIsUnavailable(t *testing.T) {
	t.Parallel()

	contract := domain.ValidationContract{
		Version: 1,
		Kind:    "workspace_contract",
		Stages: []domain.ValidationStage{
			{ID: "runtime", Engine: "kotlin.runtime", Mode: domain.ValidationModeFinal},
		},
	}
	codeStructure, err := json.Marshal(contract)
	if err != nil {
		t.Fatalf("marshal contract: %v", err)
	}

	parser := usecase.NewContractParser(usecase.NewDefaultLegacyContractAdapter())
	useCase := usecase.NewOrchestrateValidationUseCase(parser, nil)
	result, err := useCase.Execute(context.Background(), domain.ValidationRequest{
		TaskID:        "task-kotlin-runtime-unavailable",
		Mode:          domain.ValidationModeFinal,
		CodeStructure: codeStructure,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Passed || len(result.Stages) != 1 || result.Stages[0].Status != "failed" {
		t.Fatalf("expected unavailable Kotlin runtime to fail closed, got %+v", result)
	}
}
