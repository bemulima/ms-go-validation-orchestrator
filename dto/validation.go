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

type InspectContractRequest struct {
	Mode          string `json:"mode,omitempty"`
	CodeStructure any    `json:"code_structure"`
}

type VerifyContractRequest struct {
	Schema               string                     `json:"schema"`
	BlueprintDigest      string                     `json:"blueprint_digest"`
	RuntimeProfileDigest string                     `json:"runtime_profile_digest"`
	CodeStructure        any                        `json:"code_structure"`
	Cases                []ContractVerificationCase `json:"cases"`
}

type ContractVerificationCase struct {
	ID            string `json:"id"`
	Kind          string `json:"kind"`
	WorkspaceRoot string `json:"workspace_root"`
}

type WorkspaceInput struct {
	Files    []WorkspaceFileInput `json:"files,omitempty"`
	RootPath string               `json:"root_path,omitempty"`
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
