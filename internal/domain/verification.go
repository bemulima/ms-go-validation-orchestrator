package domain

import (
	"encoding/json"
	"time"
)

const (
	ContractVerificationRequestSchemaV1 = "validation-contract-verification-request.v1"
	ContractVerificationReceiptSchemaV1 = "validation-contract-verification-receipt.v1"

	VerificationCaseStarter   = "starter"
	VerificationCaseReference = "reference"
	VerificationCaseNegative  = "negative"
)

// ContractVerificationRequestV1 references isolated sandbox workspaces. It
// never transports private fixture contents through the orchestrator API.
type ContractVerificationRequestV1 struct {
	Schema               string                       `json:"schema"`
	BlueprintDigest      string                       `json:"blueprint_digest"`
	RuntimeProfileDigest string                       `json:"runtime_profile_digest"`
	CodeStructure        json.RawMessage              `json:"code_structure"`
	Cases                []ContractVerificationCaseV1 `json:"cases"`
}

type ContractVerificationCaseV1 struct {
	ID            string `json:"id"`
	Kind          string `json:"kind"`
	WorkspaceRoot string `json:"workspace_root"`
}

type ContractVerificationReceiptV1 struct {
	Schema               string                             `json:"schema"`
	BlueprintDigest      string                             `json:"blueprint_digest"`
	RuntimeProfileDigest string                             `json:"runtime_profile_digest"`
	ContractDigest       string                             `json:"contract_digest"`
	CapabilitiesDigest   string                             `json:"capabilities_digest"`
	Passed               bool                               `json:"passed"`
	Cases                []ContractVerificationCaseResultV1 `json:"cases"`
	VerifiedAt           time.Time                          `json:"verified_at"`
	ReceiptDigest        string                             `json:"receipt_digest"`
}

type ContractVerificationCaseResultV1 struct {
	ID             string `json:"id"`
	Kind           string `json:"kind"`
	ExpectedPassed bool   `json:"expected_passed"`
	ActualPassed   bool   `json:"actual_passed"`
	ResultDigest   string `json:"result_digest"`
}
