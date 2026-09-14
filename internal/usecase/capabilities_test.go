package usecase

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type capabilityTestEngine struct {
	id string
}

func (engine capabilityTestEngine) EngineID() string { return engine.id }

func (engine capabilityTestEngine) Validate(
	context.Context,
	domain.EngineValidationInput,
) (domain.StageExecutionResult, error) {
	return domain.StageExecutionResult{Passed: true}, nil
}

func TestConfiguredEngineCapabilitiesAreVersionedAndDeterministic(t *testing.T) {
	useCase := NewOrchestrateValidationUseCase(
		NewContractParser(NewDefaultLegacyContractAdapter()),
		[]domain.EngineClient{
			capabilityTestEngine{id: "ts.runtime"},
			capabilityTestEngine{id: "go.core"},
			capabilityTestEngine{id: "legacy.generic"},
		},
	)

	capabilities := useCase.ConfiguredEngineCapabilities()
	if capabilities.Schema != domain.EngineCapabilitiesSchemaV1 || capabilities.Digest == "" {
		t.Fatalf("unexpected capability envelope: %+v", capabilities)
	}
	if got := []string{capabilities.Engines[0].ID, capabilities.Engines[1].ID, capabilities.Engines[2].ID}; !reflect.DeepEqual(got, []string{"go.core", "legacy.generic", "ts.runtime"}) {
		t.Fatalf("unexpected capability order: %v", got)
	}
	if capabilities.Engines[1].AuthoringSupported {
		t.Fatal("legacy engine must not be advertised for new authoring")
	}
	if !reflect.DeepEqual(capabilities.Engines[2].Modes, []string{domain.ValidationModeFinal}) {
		t.Fatalf("runtime modes are not fail-closed: %v", capabilities.Engines[2].Modes)
	}
}

func TestInspectContractReportsMissingEnginesWithoutExecuting(t *testing.T) {
	useCase := NewOrchestrateValidationUseCase(
		NewContractParser(NewDefaultLegacyContractAdapter()),
		[]domain.EngineClient{capabilityTestEngine{id: "go.core"}},
	)
	contract := domain.ValidationContract{
		Version: 1,
		Kind:    "workspace_contract",
		Stages: []domain.ValidationStage{
			{ID: "compile", Engine: "go.core", Mode: domain.ValidationModeBoth},
			{ID: "run", Engine: "linux.runtime", Mode: domain.ValidationModeFinal, DependsOn: []string{"compile"}},
		},
	}
	raw, err := json.Marshal(contract)
	if err != nil {
		t.Fatalf("marshal contract: %v", err)
	}

	result, err := useCase.InspectContract(domain.ContractInspectionRequest{
		CodeStructure: raw,
		Mode:          domain.ValidationModeFinal,
	})
	if err != nil {
		t.Fatalf("inspect contract: %v", err)
	}
	if result.Runnable || !reflect.DeepEqual(result.MissingEngines, []string{"linux.runtime"}) {
		t.Fatalf("unexpected inspection result: %+v", result)
	}
	if !reflect.DeepEqual(result.ExecutionOrder, []string{"compile", "run"}) {
		t.Fatalf("unexpected execution order: %v", result.ExecutionOrder)
	}
	if result.ContractDigest == "" || result.CapabilitiesDigest == "" {
		t.Fatalf("inspection digests are required: %+v", result)
	}
}

func TestRuntimeEngineModeIsConsistentAcrossCapabilitiesInspectionAndExecution(t *testing.T) {
	useCase := NewOrchestrateValidationUseCase(
		NewContractParser(NewDefaultLegacyContractAdapter()),
		[]domain.EngineClient{capabilityTestEngine{id: "go.core"}, capabilityTestEngine{id: "go.gin.runtime"}},
	)
	contract, err := json.Marshal(domain.ValidationContract{Version: 1, Kind: "workspace_contract", Stages: []domain.ValidationStage{
		{ID: "compile", Engine: "go.core", Mode: domain.ValidationModeBoth},
		{ID: "serve", Engine: "go.gin.runtime", Mode: domain.ValidationModeBoth},
	}})
	if err != nil {
		t.Fatal(err)
	}

	live, err := useCase.InspectContract(domain.ContractInspectionRequest{CodeStructure: contract, Mode: domain.ValidationModeLive})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(live.ExecutionOrder, []string{"compile"}) {
		t.Fatalf("live order=%v", live.ExecutionOrder)
	}
	final, err := useCase.InspectContract(domain.ContractInspectionRequest{CodeStructure: contract, Mode: domain.ValidationModeFinal})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(final.ExecutionOrder, []string{"compile", "serve"}) {
		t.Fatalf("final order=%v", final.ExecutionOrder)
	}
}
