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

// WorkspaceFileReader projects an isolated portable verification root into the
// existing text path/content representation. It never supplies official pins.
type WorkspaceFileReader interface {
	ReadFiles(ctx context.Context, portableRoot string) ([]WorkspaceFile, error)
}

// VerificationFileInput is an optional adapter-owned transport requirement.
// Undeclared engines retain their prior root dispatch; no input support is inferred.
type VerificationFileInput interface {
	NeedsWorkspaceFiles(stage ValidationStage) bool
}

type Logger interface {
	Info(message string, fields map[string]string)
	Error(message string, fields map[string]string)
}
