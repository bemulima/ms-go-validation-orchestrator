package mapper

import (
	"encoding/json"

	"github.com/example/ms-validation-orchestrator-service/dto"
	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

func ToDomainValidationRequest(input dto.ValidateRequest) (domain.ValidationRequest, error) {
	codeStructure, err := json.Marshal(input.CodeStructure)
	if err != nil {
		return domain.ValidationRequest{}, err
	}

	files := make([]domain.WorkspaceFile, 0, len(input.Workspace.Files))
	for _, file := range input.Workspace.Files {
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
