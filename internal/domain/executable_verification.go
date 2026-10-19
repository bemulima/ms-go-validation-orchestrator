package domain

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// CanonicalDigestV2 is scoped to new C2 contracts; V1 preimages are unchanged.
func CanonicalDigestV2(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	var obj any
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	if d.Decode(&obj) != nil {
		return ""
	}
	b, err = json.Marshal(obj)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

type GoTestManifest struct {
	StarterFiles           map[string]string `json:"starter_files,omitempty"`
	ReadOnlyFiles          []string          `json:"readonly_files,omitempty"`
	EditableFiles          []string          `json:"editable_files,omitempty"`
	Entrypoints            []string          `json:"entrypoints,omitempty"`
	Schema                 string            `json:"schema"`
	ProfileID              string            `json:"profile_id"`
	BundleVersion          string            `json:"bundle_version"`
	BundleDigest           string            `json:"bundle_digest"`
	RunnerDigest           string            `json:"runner_digest"`
	ExecutionProfileDigest string            `json:"execution_profile_digest"`
	CompilerImage          string            `json:"compiler_image"`
	TestFileDigests        map[string]string `json:"test_file_digests"`
	CriterionTests         map[string]string `json:"criterion_tests"`
	Starter                string            `json:"starter"`
	StarterDigest          string            `json:"starter_digest"`
	FixtureDigests         map[string]string `json:"fixture_digests"`
}
type GoTestBinding struct {
	Schema                 string            `json:"schema"`
	ProfileID              string            `json:"profile_id"`
	BundleDigest           string            `json:"bundle_digest"`
	RunnerDigest           string            `json:"runner_digest"`
	ExecutionProfileDigest string            `json:"execution_profile_digest"`
	StarterDigest          string            `json:"starter_digest"`
	Files                  map[string]string `json:"files,omitempty"`
}
type GoTestOutcome struct {
	Status       string            `json:"status"`
	Code         string            `json:"code"`
	Criteria     map[string]string `json:"criteria"`
	SourceDigest string            `json:"source_digest"`
}
type GoTestResult struct {
	Schema   string                   `json:"schema"`
	Manifest GoTestManifest           `json:"manifest"`
	Outcomes map[string]GoTestOutcome `json:"outcomes"`
	Digest   string                   `json:"digest"`
}
type GoTestPort interface {
	Capability(context.Context) (GoTestManifest, error)
	Run(context.Context, GoTestBinding, bool) (GoTestResult, error)
}
type ExecutableVerificationRequest struct {
	Schema               string            `json:"schema"`
	IdempotencyKey       string            `json:"idempotency_key"`
	CandidateDigest      string            `json:"candidate_digest"`
	ValidationContract   json.RawMessage   `json:"validation_contract"`
	RuntimeProfileDigest string            `json:"runtime_profile_digest"`
	Binding              GoTestBinding     `json:"binding"`
	CriterionTests       map[string]string `json:"criterion_tests"`
}
type ExecutableVerificationReceipt struct {
	Schema             string                        `json:"schema"`
	Request            ExecutableVerificationRequest `json:"request"`
	ContractDigest     string                        `json:"contract_digest"`
	CapabilitiesDigest string                        `json:"capabilities_digest"`
	Verification       GoTestResult                  `json:"verification"`
	Issuer             string                        `json:"issuer"`
	KeyID              string                        `json:"key_id"`
	VerifiedAt         string                        `json:"verified_at"`
	ReceiptDigest      string                        `json:"receipt_digest"`
	Seal               string                        `json:"seal"`
}
type VerificationReceiptValidation struct {
	Receipt         ExecutableVerificationReceipt `json:"receipt"`
	ExpectedRequest ExecutableVerificationRequest `json:"expected_request"`
}

// ProfiledGoTestPort is additive; existing single-profile adapters remain valid.
type ProfiledGoTestPort interface {
	CapabilityForProfile(context.Context, string) (GoTestManifest, error)
}
