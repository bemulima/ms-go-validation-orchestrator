package mapper

import (
	"encoding/json"
	"fmt"

	"github.com/example/ms-validation-orchestrator-service/dto"
	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

func ToDomainValidationRequest(input dto.ValidateRequest) (domain.ValidationRequest, error) {
	codeStructure, err := json.Marshal(input.CodeStructure)
	if err != nil {
		return domain.ValidationRequest{}, err
	}

	if err := domain.ValidateSandboxWorkspaceRoot(input.Workspace.RootPath); err != nil {
		return domain.ValidationRequest{}, err
	}
	if len(input.Workspace.Files) > 10000 {
		return domain.ValidationRequest{}, fmt.Errorf("%w: too many workspace files", domain.ErrInvalidRequest)
	}

	files := make([]domain.WorkspaceFile, 0, len(input.Workspace.Files))
	seenPaths := make(map[string]struct{}, len(input.Workspace.Files))
	for _, file := range input.Workspace.Files {
		if err := domain.ValidateWorkspaceFilePath(file.Path); err != nil {
			return domain.ValidationRequest{}, err
		}
		if _, duplicate := seenPaths[file.Path]; duplicate {
			return domain.ValidationRequest{}, fmt.Errorf("%w: duplicate workspace file %q", domain.ErrInvalidRequest, file.Path)
		}
		seenPaths[file.Path] = struct{}{}
		files = append(files, domain.WorkspaceFile{
			Path:    file.Path,
			Content: file.Content,
		})
	}

	return domain.ValidationRequest{
		TaskID:                input.TaskID,
		CodeStructureTypeCode: input.CodeStructureTypeCode,
		Mode:                  input.Mode,
		Locale:                input.Locale,
		CodeStructure:         codeStructure,
		Workspace: domain.ValidationWorkspace{
			Files:    files,
			RootPath: input.Workspace.RootPath,
		},
		TaskMetadata: domain.TaskMetadata{
			TaskKind:               input.TaskMetadata.TaskKind,
			ExecutionMode:          input.TaskMetadata.ExecutionMode,
			EvaluationMode:         input.TaskMetadata.EvaluationMode,
			SubmissionMode:         input.TaskMetadata.SubmissionMode,
			SupportsLiveValidation: input.TaskMetadata.SupportsLiveValidation,
		},
	}, nil
}

func ToDomainContractInspectionRequest(input dto.InspectContractRequest) (domain.ContractInspectionRequest, error) {
	codeStructure, err := json.Marshal(input.CodeStructure)
	if err != nil {
		return domain.ContractInspectionRequest{}, err
	}
	return domain.ContractInspectionRequest{
		CodeStructure: codeStructure,
		Mode:          input.Mode,
	}, nil
}

func ToDomainContractVerificationRequest(input dto.VerifyContractRequest) (domain.ContractVerificationRequestV1, error) {
	codeStructure, err := json.Marshal(input.CodeStructure)
	if err != nil {
		return domain.ContractVerificationRequestV1{}, err
	}
	cases := make([]domain.ContractVerificationCaseV1, 0, len(input.Cases))
	for _, verificationCase := range input.Cases {
		cases = append(cases, domain.ContractVerificationCaseV1{
			ID:            verificationCase.ID,
			Kind:          verificationCase.Kind,
			WorkspaceRoot: verificationCase.WorkspaceRoot,
		})
	}
	return domain.ContractVerificationRequestV1{
		Schema:               input.Schema,
		BlueprintDigest:      input.BlueprintDigest,
		RuntimeProfileDigest: input.RuntimeProfileDigest,
		CodeStructure:        codeStructure,
		Cases:                cases,
	}, nil
}
