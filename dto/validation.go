package dto

type ValidateRequest struct {
	TaskID                string         `json:"task_id"`
	CodeStructureTypeCode string         `json:"code_structure_type_code,omitempty"`
	Mode                  string         `json:"mode,omitempty"`
	Locale                string         `json:"locale,omitempty"`
	CodeStructure         any            `json:"code_structure"`
	Workspace             WorkspaceInput `json:"workspace,omitempty"`
	TaskMetadata          TaskMetadata   `json:"task_metadata,omitempty"`
}

type WorkspaceInput struct {
	Files []WorkspaceFileInput `json:"files,omitempty"`
}

type WorkspaceFileInput struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type TaskMetadata struct {
	TaskKind               string `json:"task_kind,omitempty"`
	ExecutionMode          string `json:"execution_mode,omitempty"`
	EvaluationMode         string `json:"evaluation_mode,omitempty"`
	SubmissionMode         string `json:"submission_mode,omitempty"`
	SupportsLiveValidation bool   `json:"supports_live_validation,omitempty"`
}
