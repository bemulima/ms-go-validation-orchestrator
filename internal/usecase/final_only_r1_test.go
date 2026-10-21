package usecase

import (
	"context"
	"encoding/json"
	"github.com/example/ms-validation-orchestrator-service/internal/domain"
	"testing"
)

func TestGoFinalOnlyStageFiltering(t *testing.T) {
	for _, mode := range []string{"", "both", "final"} {
		s := []domain.ValidationStage{{ID: "go", Engine: "go.tests.v1", Mode: mode}}
		if len(filterStagesByMode(s, "live")) != 0 || len(filterStagesByMode(s, "final")) != 1 {
			t.Fatal(mode)
		}
	}
	if !isFinalOnlyEngine("http.runtime") || isFinalOnlyEngine("go.core") {
		t.Fatal("predicate regression")
	}
}

type finalOnlyFake struct{ calls int }

func (f *finalOnlyFake) EngineID() string { return "go.tests.v1" }
func (f *finalOnlyFake) Validate(context.Context, domain.EngineValidationInput) (domain.StageExecutionResult, error) {
	f.calls++
	return domain.StageExecutionResult{Passed: true}, nil
}
func TestGoCapabilitiesInspectionAndExecutionShareFinalOnly(t *testing.T) {
	f := &finalOnlyFake{}
	u := NewOrchestrateValidationUseCase(NewContractParser(NewDefaultLegacyContractAdapter()), []domain.EngineClient{f})
	caps := u.ConfiguredEngineCapabilities()
	if len(caps.Engines) != 1 || len(caps.Engines[0].Modes) != 1 || caps.Engines[0].Modes[0] != "final" {
		t.Fatal("capabilities")
	}
	for _, mode := range []string{"live", "final"} {
		raw := json.RawMessage(`{"version":1,"kind":"workspace_contract","stages":[{"id":"go","engine":"go.tests.v1","mode":"both"}]}`)
		plan, e := u.InspectContract(domain.ContractInspectionRequest{CodeStructure: raw, Mode: mode})
		if e != nil {
			t.Fatal(e)
		}
		before := f.calls
		_, e = u.Execute(context.Background(), domain.ValidationRequest{CodeStructure: raw, Mode: mode})
		if e != nil {
			t.Fatal(e)
		}
		want := 0
		if mode == "final" {
			want = 1
		}
		if f.calls-before != want || len(plan.ExecutionOrder) != want {
			t.Fatal("inspection/execution disagree")
		}
	}
}
