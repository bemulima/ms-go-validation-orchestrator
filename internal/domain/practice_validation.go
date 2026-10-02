package domain

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
)

const (
	PracticeValidationRequestSchemaV2       = "practice-validation-request.v2"
	PracticeValidationSpecificationSchemaV1 = "practice-validation-specification.v1"
	PracticeValidationResultSchemaV2        = "practice-validation-result.v2"
	PracticePinnedSnapshotContentSchemaV2   = "practice-pinned-snapshot-content.v2"

	PracticeValidationScopeMilestone = "MILESTONE"
	PracticeValidationScopeFinal     = "FINAL"

	PracticeValidationValidatorTimeout         = "VALIDATOR_TIMEOUT"
	PracticeValidationValidatorUnavailable     = "VALIDATOR_UNAVAILABLE"
	PracticeValidationValidatorProtocol        = "VALIDATOR_PROTOCOL"
	PracticeValidationStageExecutionError      = "STAGE_EXECUTION_ERROR"
	PracticeValidationSnapshotUnavailable      = "SNAPSHOT_UNAVAILABLE"
	PracticeValidationSnapshotCorrupt          = "SNAPSHOT_CORRUPT"
	PracticeValidationSnapshotDigestMismatch   = "SNAPSHOT_DIGEST_MISMATCH"
	PracticeValidationUnsupportedConfiguration = "UNSUPPORTED_CONFIGURATION"
	PracticeValidationAuthoringDefect          = "AUTHORING_DEFECT"
)

type PracticeValidationOutcomeV2 string

const (
	PracticeValidationPass    PracticeValidationOutcomeV2 = "PASS"
	PracticeValidationFail    PracticeValidationOutcomeV2 = "FAIL"
	PracticeValidationError   PracticeValidationOutcomeV2 = "ERROR"
	PracticeValidationSkipped PracticeValidationOutcomeV2 = "SKIPPED"
)

var (
	ErrPinnedSnapshotUnavailable    = errors.New("pinned snapshot unavailable")
	ErrPinnedSnapshotCorrupt        = errors.New("pinned snapshot corrupt")
	ErrPinnedSnapshotDigestMismatch = errors.New("pinned snapshot digest mismatch")
)

var practiceSnapshotIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
var practiceSandboxUUIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
var practiceSnapshotDigestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// PracticeSnapshotRefV2 binds validation to one immutable Sandbox generation.
type PracticeSnapshotRefV2 struct {
	SandboxID       string `json:"sandbox_id"`
	SnapshotID      string `json:"snapshot_id"`
	GenerationID    string `json:"generation_id"`
	WorkspaceDigest string `json:"workspace_digest"`
}

func ValidatePracticeSnapshotRefV2(ref PracticeSnapshotRefV2) error {
	if !practiceSandboxUUIDPattern.MatchString(ref.SandboxID) ||
		!practiceSnapshotIDPattern.MatchString(ref.SnapshotID) || ref.SnapshotID == "." || ref.SnapshotID == ".." ||
		!practiceSnapshotIDPattern.MatchString(ref.GenerationID) || ref.GenerationID == "." || ref.GenerationID == ".." ||
		!practiceSnapshotDigestPattern.MatchString(ref.WorkspaceDigest) {
		return fmt.Errorf("%w: invalid exact snapshot reference", ErrInvalidRequest)
	}
	return nil
}

// PinnedSnapshotV2 contains bytes read from an exact retained Sandbox pin.
type PinnedSnapshotV2 struct {
	Ref      PracticeSnapshotRefV2
	Revision int64
	Files    []WorkspaceFile
}

// PracticeValidationRequestV2 is the strict inbound Practice Runtime command.
// ValidationSpecification remains raw here so the digest is computed over the
// complete versioned JSON value, including optional fields that were supplied.
type PracticeValidationRequestV2 struct {
	Schema                   string                `json:"schema"`
	TaskInstanceID           string                `json:"task_instance_id"`
	SubmissionID             string                `json:"submission_id"`
	ValidationRunID          string                `json:"validation_run_id"`
	RevisionDigest           string                `json:"revision_digest"`
	ValidationContractDigest string                `json:"validation_contract_digest"`
	Snapshot                 PracticeSnapshotRefV2 `json:"snapshot"`
	Scope                    string                `json:"scope"`
	MilestoneID              string                `json:"milestone_id,omitempty"`
	MilestoneIDPresent       bool                  `json:"-"`
	ValidationSpecification  json.RawMessage       `json:"validation_specification"`
}

func (request *PracticeValidationRequestV2) UnmarshalJSON(data []byte) error {
	type requestAlias PracticeValidationRequestV2
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var decoded requestAlias
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("request must contain exactly one JSON object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*request = PracticeValidationRequestV2(decoded)
	_, request.MilestoneIDPresent = fields["milestone_id"]
	return nil
}

type PracticeValidationStageSummaryV2 struct {
	StageID  string                      `json:"stage_id"`
	Required bool                        `json:"required"`
	Outcome  PracticeValidationOutcomeV2 `json:"outcome"`
	Issues   []PracticeValidationIssueV2 `json:"issues"`
}

type PracticeValidationIssueV2 struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type PracticeValidationResultV2 struct {
	Schema                   string                             `json:"schema"`
	TaskInstanceID           string                             `json:"task_instance_id"`
	SubmissionID             string                             `json:"submission_id"`
	ValidationRunID          string                             `json:"validation_run_id"`
	RevisionDigest           string                             `json:"revision_digest"`
	ValidationContractDigest string                             `json:"validation_contract_digest"`
	Snapshot                 PracticeSnapshotRefV2              `json:"snapshot"`
	Outcome                  PracticeValidationOutcomeV2        `json:"outcome"`
	ErrorClassification      string                             `json:"error_classification,omitempty"`
	Stages                   []PracticeValidationStageSummaryV2 `json:"stages"`
}

// PinnedSnapshotReader must read by the full immutable reference; it must not
// fall back to the current workspace head.
type PinnedSnapshotReader interface {
	ReadPinnedSnapshot(ctx context.Context, ref PracticeSnapshotRefV2) (PinnedSnapshotV2, error)
}

// WorkspaceFilesDigestV2 matches Sandbox's atomic generation digest: SHA-256
// over JSON-encoded path/content pairs sorted by path.
func WorkspaceFilesDigestV2(files []WorkspaceFile) (string, error) {
	canonical := make([]WorkspaceFile, len(files))
	copy(canonical, files)
	sort.Slice(canonical, func(left, right int) bool { return canonical[left].Path < canonical[right].Path })
	for index, file := range canonical {
		if err := ValidateWorkspaceFilePath(file.Path); err != nil {
			return "", fmt.Errorf("invalid snapshot file path: %w", err)
		}
		if index > 0 && canonical[index-1].Path == file.Path {
			return "", fmt.Errorf("duplicate snapshot file path %q", file.Path)
		}
	}
	payload, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("marshal snapshot files: %w", err)
	}
	digest := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}
