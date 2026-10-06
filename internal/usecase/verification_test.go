package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type fixtureOutcomeEngine struct {
	id string
}

func (engine fixtureOutcomeEngine) EngineID() string                         { return engine.id }
func (fixtureOutcomeEngine) NeedsWorkspaceFiles(domain.ValidationStage) bool { return true }

func (engine fixtureOutcomeEngine) Validate(
	_ context.Context,
	input domain.EngineValidationInput,
) (domain.StageExecutionResult, error) {
	return domain.StageExecutionResult{
		Passed: len(input.Workspace.Files) == 1 && input.Workspace.Files[0].Content == "<main>valid fixture</main>",
	}, nil
}

func TestVerifyContractProducesExecutableReceipt(t *testing.T) {
	useCase := NewOrchestrateValidationUseCase(
		NewContractParser(NewDefaultLegacyContractAdapter()),
		[]domain.EngineClient{fixtureOutcomeEngine{id: "go.core"}},
	).WithWorkspaceFileReader(&verificationFixtureReader{})
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

type verificationFilesEngine struct {
	t      *testing.T
	inputs []domain.EngineValidationInput
}

func (*verificationFilesEngine) EngineID() string                                { return "html" }
func (*verificationFilesEngine) NeedsWorkspaceFiles(domain.ValidationStage) bool { return true }

func (engine *verificationFilesEngine) Validate(_ context.Context, input domain.EngineValidationInput) (domain.StageExecutionResult, error) {
	engine.inputs = append(engine.inputs, input)
	if len(input.Workspace.Files) != 1 || input.Workspace.Files[0].Path != "index.html" {
		engine.t.Errorf("engine must receive hydrated path/content files, got %d files", len(input.Workspace.Files))
		return domain.StageExecutionResult{Passed: false}, nil
	}
	return domain.StageExecutionResult{Passed: input.Workspace.Files[0].Content == "<main>valid fixture</main>"}, nil
}

// Root-name-driven fake outcomes cannot prove that file-only providers receive
// fixture contents. This regression exercises the existing Files contract.
func TestVerifyContractHydratesFilesBeforeEngineExecution(t *testing.T) {
	engine := &verificationFilesEngine{t: t}
	useCase := NewOrchestrateValidationUseCase(NewContractParser(NewDefaultLegacyContractAdapter()), []domain.EngineClient{engine})
	reader := &verificationFixtureReader{}
	useCase = useCase.WithWorkspaceFileReader(reader)
	request := domain.ContractVerificationRequestV1{
		Schema: domain.ContractVerificationRequestSchemaV1, BlueprintDigest: testSHA256Digest("1"), RuntimeProfileDigest: testSHA256Digest("2"),
		CodeStructure: json.RawMessage(`{"version":1,"kind":"workspace_contract","stages":[{"id":"html","engine":"html"}]}`),
		Cases: []domain.ContractVerificationCaseV1{
			{ID: "starter", Kind: domain.VerificationCaseStarter, WorkspaceRoot: "/workspaces/starter"},
			{ID: "reference", Kind: domain.VerificationCaseReference, WorkspaceRoot: "/workspaces/reference"},
			{ID: "negative", Kind: domain.VerificationCaseNegative, WorkspaceRoot: "/workspaces/negative"},
		},
	}
	receipt, err := useCase.VerifyContract(context.Background(), request)
	if err != nil || !receipt.Passed {
		t.Fatalf("file-backed fixture verification must match expected outcomes: passed=%t err=%v", receipt.Passed, err)
	}
	if len(engine.inputs) != len(request.Cases) {
		t.Fatalf("engine calls=%d, want %d", len(engine.inputs), len(request.Cases))
	}
	for i, input := range engine.inputs {
		if input.Workspace.RootPath != request.Cases[i].WorkspaceRoot {
			t.Fatal("hydration must preserve the exact portable case root")
		}
	}
	if len(reader.roots) != len(request.Cases) {
		t.Fatalf("reader calls=%d, want one per case", len(reader.roots))
	}
}

type verificationErrorEngine struct{}

func (verificationErrorEngine) EngineID() string { return "go.core" }
func (verificationErrorEngine) Validate(context.Context, domain.EngineValidationInput) (domain.StageExecutionResult, error) {
	return domain.StageExecutionResult{}, errors.New("private fixture content at /host/private/reference")
}

func TestVerifyContractRejectsTechnicalFailureInsteadOfFixtureOutcome(t *testing.T) {
	useCase := NewOrchestrateValidationUseCase(NewContractParser(NewDefaultLegacyContractAdapter()), []domain.EngineClient{verificationErrorEngine{}})
	useCase = useCase.WithWorkspaceFileReader(&verificationFixtureReader{})
	_, err := useCase.VerifyContract(context.Background(), domain.ContractVerificationRequestV1{
		Schema: domain.ContractVerificationRequestSchemaV1, BlueprintDigest: testSHA256Digest("1"), RuntimeProfileDigest: testSHA256Digest("2"),
		CodeStructure: json.RawMessage(`{"version":1,"kind":"workspace_contract","stages":[{"id":"go","engine":"go.core","optional":true}]}`),
		Cases: []domain.ContractVerificationCaseV1{
			{ID: "starter", Kind: domain.VerificationCaseStarter, WorkspaceRoot: "/workspaces/starter"},
			{ID: "reference", Kind: domain.VerificationCaseReference, WorkspaceRoot: "/workspaces/reference"},
			{ID: "negative", Kind: domain.VerificationCaseNegative, WorkspaceRoot: "/workspaces/negative"},
		},
	})
	if !errors.Is(err, domain.ErrStageExecutionFailed) || strings.Contains(err.Error(), "private") {
		t.Fatalf("technical verification error must be redacted, got %v", err)
	}
}

type verificationFixtureReader struct {
	roots []string
	err   error
}

func (reader *verificationFixtureReader) ReadFiles(_ context.Context, root string) ([]domain.WorkspaceFile, error) {
	reader.roots = append(reader.roots, root)
	content := "invalid fixture"
	if root == "/workspaces/reference" {
		content = "<main>valid fixture</main>"
	}
	return []domain.WorkspaceFile{{Path: "index.html", Content: content}}, reader.err
}

func TestVerifyContractMissingReaderOrReadFailureIsTechnicalAndRedacted(t *testing.T) {
	for _, reader := range []*verificationFixtureReader{nil, {err: errors.New("private contents /host/secret")}} {
		engine := &verificationFilesEngine{t: t}
		useCase := NewOrchestrateValidationUseCase(NewContractParser(NewDefaultLegacyContractAdapter()), []domain.EngineClient{engine})
		if reader != nil {
			useCase = useCase.WithWorkspaceFileReader(reader)
		}
		receipt, err := useCase.VerifyContract(context.Background(), domain.ContractVerificationRequestV1{
			Schema: domain.ContractVerificationRequestSchemaV1, BlueprintDigest: testSHA256Digest("1"), RuntimeProfileDigest: testSHA256Digest("2"),
			CodeStructure: json.RawMessage(`{"version":1,"kind":"workspace_contract","stages":[{"id":"html","engine":"html"}]}`),
			Cases: []domain.ContractVerificationCaseV1{
				{ID: "starter", Kind: domain.VerificationCaseStarter, WorkspaceRoot: "/workspaces/starter"},
				{ID: "reference", Kind: domain.VerificationCaseReference, WorkspaceRoot: "/workspaces/reference"},
				{ID: "negative", Kind: domain.VerificationCaseNegative, WorkspaceRoot: "/workspaces/negative"},
			},
		})
		if !errors.Is(err, domain.ErrVerificationWorkspace) || receipt.Passed || len(engine.inputs) != 0 || strings.Contains(err.Error(), "private") || strings.Contains(err.Error(), "/host") {
			t.Fatalf("unsafe verification result: receipt=%+v err=%v calls=%d", receipt, err, len(engine.inputs))
		}
	}
}
