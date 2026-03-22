package engines

import (
	"context"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type LegacyGenericEngine struct{}

func NewLegacyGenericEngine() LegacyGenericEngine {
	return LegacyGenericEngine{}
}

func (engine LegacyGenericEngine) EngineID() string {
	return "legacy.generic"
}

func (engine LegacyGenericEngine) Validate(
	_ context.Context,
	input domain.EngineValidationInput,
) (domain.StageExecutionResult, error) {
	return domain.StageExecutionResult{
		Passed: false,
		Errors: []domain.ValidationIssue{
			{
				Code:     "LEGACY_CONTRACT_NOT_MIGRATED",
				Message:  "legacy code_structure was adapted, but legacy execution is not wired into orchestrator yet",
				Severity: "error",
				StageID:  input.Stage.ID,
				Engine:   input.Stage.Engine,
				Hint:     "use legacy task-answer path until orchestrator integration is enabled for this task",
			},
		},
	}, nil
}
