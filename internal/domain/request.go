package domain

import "encoding/json"

type ValidationRequest struct {
	TaskID                string              `json:"task_id"`
	CodeStructureTypeCode string              `json:"code_structure_type_code,omitempty"`
	Mode                  string              `json:"mode,omitempty"`
	Locale                string              `json:"locale,omitempty"`
	CodeStructure         json.RawMessage     `json:"code_structure"`
	Workspace             ValidationWorkspace `json:"workspace,omitempty"`
	TaskMetadata          TaskMetadata        `json:"task_metadata,omitempty"`
}

type ValidationWorkspace struct {
	Files []WorkspaceFile `json:"files,omitempty"`
}

type WorkspaceFile struct {
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
