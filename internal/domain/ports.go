package domain

import "context"

type EngineValidationInput struct {
	TaskID       string
	Stage        ValidationStage
	Workspace    ValidationWorkspace
	TaskMetadata TaskMetadata
	Locale       string
	Mode         string
}

type EngineClient interface {
	EngineID() string
	Validate(ctx context.Context, input EngineValidationInput) (StageExecutionResult, error)
}

type Logger interface {
	Info(message string, fields map[string]string)
	Error(message string, fields map[string]string)
}
