package domain

import "encoding/json"

const (
	EngineCapabilitiesSchemaV1 = "validation-engine-capabilities.v1"
	ContractInspectionSchemaV1 = "validation-contract-inspection.v1"
)

type EngineCapabilitiesV1 struct {
	Schema  string               `json:"schema"`
	Digest  string               `json:"digest"`
	Engines []EngineCapabilityV1 `json:"engines"`
}

type EngineCapabilityV1 struct {
	ID                 string   `json:"id"`
	ContractVersions   []int    `json:"contract_versions"`
	Modes              []string `json:"modes"`
	Execution          string   `json:"execution"`
	WorkspaceInputs    []string `json:"workspace_inputs"`
	AuthoringSupported bool     `json:"authoring_supported"`
}

type ContractInspectionRequest struct {
	CodeStructure json.RawMessage `json:"code_structure"`
	Mode          string          `json:"mode,omitempty"`
}

type ContractInspectionResultV1 struct {
	Schema             string   `json:"schema"`
	ContractKind       string   `json:"contract_kind"`
	ContractVersion    int      `json:"contract_version"`
	ContractDigest     string   `json:"contract_digest"`
	CapabilitiesDigest string   `json:"capabilities_digest"`
	RequestedMode      string   `json:"requested_mode,omitempty"`
	ExecutionOrder     []string `json:"execution_order"`
	RequiredEngines    []string `json:"required_engines"`
	MissingEngines     []string `json:"missing_engines"`
	Runnable           bool     `json:"runnable"`
}
