package usecase

import "github.com/example/ms-validation-orchestrator-service/internal/domain"

type LegacyContractAdapter interface {
	Adapt(request domain.ValidationRequest) (domain.ValidationContract, error)
}

type DefaultLegacyContractAdapter struct{}

func NewDefaultLegacyContractAdapter() DefaultLegacyContractAdapter {
	return DefaultLegacyContractAdapter{}
}

func (adapter DefaultLegacyContractAdapter) Adapt(request domain.ValidationRequest) (domain.ValidationContract, error) {
	stage := domain.ValidationStage{
		ID:       "legacy-generic",
		Name:     "Legacy Generic Validation",
		Engine:   "legacy.generic",
		Mode:     domain.ValidationModeBoth,
		Optional: false,
		Targets: domain.StageTargets{
			Files: collectWorkspaceFiles(request.Workspace),
		},
		Rules: request.CodeStructure,
	}

	return domain.ValidationContract{
		Version:   1,
		Kind:      "legacy_contract",
		Profile:   request.CodeStructureTypeCode,
		Stages:    []domain.ValidationStage{stage},
		Links:     []domain.ValidationLink{},
		Workspace: domain.ContractWorkspace{},
	}, nil
}

func collectWorkspaceFiles(workspace domain.ValidationWorkspace) []string {
	files := make([]string, 0, len(workspace.Files))
	for _, file := range workspace.Files {
		files = append(files, file.Path)
	}

	return files
}
