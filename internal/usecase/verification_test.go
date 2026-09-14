package usecase

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type fixtureOutcomeEngine struct {
	id string
}

func (engine fixtureOutcomeEngine) EngineID() string { return engine.id }

func (engine fixtureOutcomeEngine) Validate(
	_ context.Context,
	input domain.EngineValidationInput,
) (domain.StageExecutionResult, error) {
	return domain.StageExecutionResult{
		Passed: strings.Contains(input.Workspace.RootPath, "reference"),
	}, nil
}

func TestVerifyContractProducesExecutableReceipt(t *testing.T) {
	useCase := NewOrchestrateValidationUseCase(
		NewContractParser(NewDefaultLegacyContractAdapter()),
		[]domain.EngineClient{fixtureOutcomeEngine{id: "go.core"}},
	)
	contract, err := json.Marshal(domain.ValidationContract{
		Version: 1,
		Kind:    "workspace_contract",
		Stages:  []domain.ValidationStage{{ID: "go", Engine: "go.core", Mode: domain.ValidationModeFinal}},
	})
	if err != nil {
		t.Fatalf("marshal contract: %v", err)
	}

	receipt, err := useCase.VerifyContract(context.Background(), domain.ContractVerificationRequestV1{
		Schema:               domain.ContractVerificationRequestSchemaV1,
		BlueprintDigest:      testSHA256Digest("1"),
		RuntimeProfileDigest: testSHA256Digest("2"),
		CodeStructure:        contract,
		Cases: []domain.ContractVerificationCaseV1{
			{ID: "starter", Kind: domain.VerificationCaseStarter, WorkspaceRoot: "/workspaces/starter"},
			{ID: "reference", Kind: domain.VerificationCaseReference, WorkspaceRoot: "/workspaces/reference"},
			{ID: "negative-invalid-move", Kind: domain.VerificationCaseNegative, WorkspaceRoot: "/workspaces/negative"},
		},
	})
	if err != nil {
		t.Fatalf("verify contract: %v", err)
	}
	if !receipt.Passed || receipt.Schema != domain.ContractVerificationReceiptSchemaV1 {
		t.Fatalf("unexpected receipt: %+v", receipt)
	}
	if receipt.ReceiptDigest == "" || receipt.ContractDigest == "" || receipt.CapabilitiesDigest == "" {
		t.Fatalf("receipt digests are required: %+v", receipt)
	}
	if len(receipt.Cases) != 3 || receipt.Cases[0].ExpectedPassed || !receipt.Cases[1].ExpectedPassed {
		t.Fatalf("unexpected case results: %+v", receipt.Cases)
	}
}

func TestVerifyContractRequiresStarterReferenceAndNegativeCases(t *testing.T) {
	useCase := NewOrchestrateValidationUseCase(
		NewContractParser(NewDefaultLegacyContractAdapter()),
		[]domain.EngineClient{fixtureOutcomeEngine{id: "go.core"}},
	)
	contract := json.RawMessage(`{"version":1,"kind":"workspace_contract","stages":[{"id":"go","engine":"go.core"}]}`)

	_, err := useCase.VerifyContract(context.Background(), domain.ContractVerificationRequestV1{
		Schema:               domain.ContractVerificationRequestSchemaV1,
		BlueprintDigest:      testSHA256Digest("1"),
		RuntimeProfileDigest: testSHA256Digest("2"),
		CodeStructure:        contract,
		Cases: []domain.ContractVerificationCaseV1{
			{ID: "starter", Kind: domain.VerificationCaseStarter, WorkspaceRoot: "/workspaces/starter"},
			{ID: "reference", Kind: domain.VerificationCaseReference, WorkspaceRoot: "/workspaces/reference"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "negative") {
		t.Fatalf("expected missing negative case error, got %v", err)
	}
}

func testSHA256Digest(character string) string {
	return "sha256:" + strings.Repeat(character, 64)
}
