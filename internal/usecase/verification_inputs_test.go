package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type verificationRootEngine struct{ id string }

func (engine verificationRootEngine) EngineID() string                         { return engine.id }
func (verificationRootEngine) NeedsWorkspaceFiles(domain.ValidationStage) bool { return false }
func (engine verificationRootEngine) Validate(_ context.Context, input domain.EngineValidationInput) (domain.StageExecutionResult, error) {
	return domain.StageExecutionResult{Passed: strings.HasSuffix(input.Workspace.RootPath, "/reference")}, nil
}

type verificationTextEngine struct{ verificationRootEngine }

func (verificationTextEngine) NeedsWorkspaceFiles(domain.ValidationStage) bool { return true }

func verificationInputRequest(stages []domain.ValidationStage, links []domain.ValidationLink) domain.ContractVerificationRequestV1 {
	contract, _ := json.Marshal(domain.ValidationContract{Version: 1, Kind: "workspace_contract", Stages: stages, Links: links})
	return domain.ContractVerificationRequestV1{Schema: domain.ContractVerificationRequestSchemaV1, BlueprintDigest: testSHA256Digest("1"), RuntimeProfileDigest: testSHA256Digest("2"), CodeStructure: contract, Cases: []domain.ContractVerificationCaseV1{
		{ID: "starter", Kind: domain.VerificationCaseStarter, WorkspaceRoot: "/workspaces/starter"},
		{ID: "reference", Kind: domain.VerificationCaseReference, WorkspaceRoot: "/workspaces/reference"},
		{ID: "negative", Kind: domain.VerificationCaseNegative, WorkspaceRoot: "/workspaces/negative"},
	}}
}

func TestVerifyRootOnlyDoesNotRequireLocalMount(t *testing.T) {
	for _, id := range []string{"git.core", "http.runtime", "linux.fs"} {
		for _, reader := range []*verificationFixtureReader{nil, {err: errors.New("unavailable local mount")}} {
			useCase := NewOrchestrateValidationUseCase(NewContractParser(NewDefaultLegacyContractAdapter()), []domain.EngineClient{verificationRootEngine{id: id}})
			if reader != nil {
				useCase = useCase.WithWorkspaceFileReader(reader)
			}
			receipt, err := useCase.VerifyContract(context.Background(), verificationInputRequest([]domain.ValidationStage{{ID: "root", Engine: id}}, nil))
			if err != nil || !receipt.Passed || (reader != nil && len(reader.roots) != 0) {
				t.Fatalf("root-only %s gained mount dependency: passed=%t err=%v", id, receipt.Passed, err)
			}
		}
	}
}

func TestVerifyFilteredFileConsumersDoNotRequireHydration(t *testing.T) {
	useCase := NewOrchestrateValidationUseCase(NewContractParser(NewDefaultLegacyContractAdapter()), []domain.EngineClient{verificationRootEngine{id: "git.core"}, verificationTextEngine{verificationRootEngine{id: "html.dom"}}})
	request := verificationInputRequest([]domain.ValidationStage{{ID: "root", Engine: "git.core"}, {ID: "html", Engine: "html.dom", Mode: domain.ValidationModeLive}}, []domain.ValidationLink{{ID: "filtered", Kind: "workspace.file_contains", DependsOn: []string{"html"}, Config: json.RawMessage(`{"file":"index.html","needle":"valid"}`)}})
	receipt, err := useCase.VerifyContract(context.Background(), request)
	if err != nil || !receipt.Passed {
		t.Fatalf("filtered consumers created mount dependency: passed=%t err=%v", receipt.Passed, err)
	}
}

func TestVerifyRetainedTextLinkAndMixedProjectionRequireFiles(t *testing.T) {
	for _, name := range []string{"retained link", "mixed binary projection"} {
		stages := []domain.ValidationStage{{ID: "root", Engine: "git.core"}}
		var links []domain.ValidationLink
		if name == "retained link" {
			links = []domain.ValidationLink{{ID: "text", Kind: "workspace.file_contains", Config: json.RawMessage(`{"file":"index.html","needle":"valid"}`)}}
		} else {
			stages = append(stages, domain.ValidationStage{ID: "text", Engine: "html.dom"})
		}
		reader := &verificationFixtureReader{err: errors.New("unsupported binary private contents")}
		useCase := NewOrchestrateValidationUseCase(NewContractParser(NewDefaultLegacyContractAdapter()), []domain.EngineClient{verificationRootEngine{id: "git.core"}, verificationTextEngine{verificationRootEngine{id: "html.dom"}}}).WithWorkspaceFileReader(reader)
		receipt, err := useCase.VerifyContract(context.Background(), verificationInputRequest(stages, links))
		if !errors.Is(err, domain.ErrVerificationWorkspace) || receipt.Passed || len(reader.roots) != 1 || strings.Contains(err.Error(), "private") {
			t.Fatalf("unsupported text projection must fail technical: %+v %v calls=%d", receipt, err, len(reader.roots))
		}
	}
}
