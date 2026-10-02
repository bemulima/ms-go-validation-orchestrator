package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type PracticeValidationUseCase struct {
	parser         ContractParser
	orchestrator   OrchestrateValidationUseCase
	snapshotReader domain.PinnedSnapshotReader
}

func NewPracticeValidationUseCase(
	parser ContractParser,
	engineClients []domain.EngineClient,
	snapshotReader domain.PinnedSnapshotReader,
) PracticeValidationUseCase {
	orchestrator := NewOrchestrateValidationUseCase(parser, engineClients)
	return PracticeValidationUseCase{
		parser:         parser,
		orchestrator:   orchestrator,
		snapshotReader: snapshotReader,
	}
}

type practiceValidationSpecificationV1 struct {
	Schema         string                         `json:"schema"`
	RuntimeProfile json.RawMessage                `json:"runtime_profile"`
	Contracts      []practiceValidationContractV1 `json:"contracts"`
}

type practiceValidationContractV1 struct {
	MilestoneID        string                      `json:"milestone_id"`
	Required           *bool                       `json:"required"`
	ValidationContract json.RawMessage             `json:"validation_contract"`
	EvaluationAssets   *practiceEvaluationAssetsV1 `json:"evaluation_assets"`
}

type practiceEvaluationAssetsV1 struct {
	TestSuiteRef          string   `json:"test_suite_ref,omitempty"`
	ReferenceSolutionRef  string   `json:"reference_solution_ref,omitempty"`
	NegativeFixtureRefs   []string `json:"negative_fixture_refs,omitempty"`
	VerificationRunnerRef string   `json:"verification_runner_ref,omitempty"`
}

type parsedPracticeContract struct {
	input    practiceValidationContractV1
	contract domain.ValidationContract
}

var practiceDigestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
var practiceUUIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
var safePracticeStageIDPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)
var safePracticeIssueCodePattern = regexp.MustCompile(`^[A-Z0-9_]{1,128}$`)

// ValidatePractice executes only the immutable specification and exact pinned
// snapshot carried by a Practice Runtime V2 command. Evaluation failures are
// represented in the correlated result; only malformed HTTP commands return
// an error to the caller.
func (useCase PracticeValidationUseCase) ValidatePractice(
	ctx context.Context,
	request domain.PracticeValidationRequestV2,
) (domain.PracticeValidationResultV2, error) {
	if err := validatePracticeRequestEnvelope(request); err != nil {
		return domain.PracticeValidationResultV2{}, err
	}
	result := practiceResultBase(request)
	fail := func(classification string) (domain.PracticeValidationResultV2, error) {
		result.Outcome = domain.PracticeValidationError
		result.ErrorClassification = classification
		return result, nil
	}

	specification, err := decodePracticeValidationSpecification(request.ValidationSpecification)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidRequest) {
			return domain.PracticeValidationResultV2{}, err
		}
		return fail(domain.PracticeValidationAuthoringDefect)
	}
	if specification.Schema == "" {
		return fail(domain.PracticeValidationAuthoringDefect)
	}
	if specification.Schema != domain.PracticeValidationSpecificationSchemaV1 {
		return fail(domain.PracticeValidationUnsupportedConfiguration)
	}
	contracts, err := validatePracticeSpecification(request, specification)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidRequest) {
			return domain.PracticeValidationResultV2{}, err
		}
		return fail(domain.PracticeValidationAuthoringDefect)
	}

	computedDigest, err := ComputePracticeValidationContractDigest(request.Scope, request.MilestoneID, request.ValidationSpecification)
	if err != nil || computedDigest != request.ValidationContractDigest {
		return fail(domain.PracticeValidationAuthoringDefect)
	}

	parsedContracts := make([]parsedPracticeContract, 0, len(contracts))
	requiredCriteria := 0
	for _, contractInput := range contracts {
		parsed, legacy, parseErr := useCase.parser.Parse(domain.ValidationRequest{CodeStructure: contractInput.ValidationContract})
		if parseErr != nil || legacy {
			return fail(domain.PracticeValidationAuthoringDefect)
		}
		filteredStages := filterStagesByMode(parsed.Stages, domain.ValidationModeFinal)
		orderedStages, orderErr := orderStages(filteredStages)
		if orderErr != nil || len(orderedStages) == 0 {
			return fail(domain.PracticeValidationAuthoringDefect)
		}
		filteredLinks := filterLinksByStageIDs(parsed.Links, collectStageIDs(orderedStages))
		if *contractInput.Required {
			for _, stage := range orderedStages {
				if !stage.Optional {
					requiredCriteria++
				}
			}
			for _, link := range filteredLinks {
				if !link.Optional {
					requiredCriteria++
				}
			}
		}
		parsedContracts = append(parsedContracts, parsedPracticeContract{input: contractInput, contract: parsed})
	}
	if requiredCriteria == 0 {
		return fail(domain.PracticeValidationAuthoringDefect)
	}

	if useCase.snapshotReader == nil {
		return fail(domain.PracticeValidationSnapshotUnavailable)
	}
	snapshot, snapshotErr := useCase.snapshotReader.ReadPinnedSnapshot(ctx, request.Snapshot)
	if snapshotErr != nil {
		return fail(snapshotFailureClassification(snapshotErr))
	}
	if snapshot.Ref != request.Snapshot {
		return fail(domain.PracticeValidationSnapshotCorrupt)
	}
	workspaceDigest, digestErr := domain.WorkspaceFilesDigestV2(snapshot.Files)
	if digestErr != nil {
		return fail(domain.PracticeValidationSnapshotCorrupt)
	}
	if workspaceDigest != request.Snapshot.WorkspaceDigest {
		return fail(domain.PracticeValidationSnapshotDigestMismatch)
	}
	sort.Slice(snapshot.Files, func(left, right int) bool { return snapshot.Files[left].Path < snapshot.Files[right].Path })

	metadata, locale := practiceMetadata(specification.RuntimeProfile)
	passed := true
	for _, parsed := range parsedContracts {
		missingRequiredFiles := missingSnapshotFiles(parsed.contract.Workspace.RequiredFiles, snapshot.Files)
		for _, path := range missingRequiredFiles {
			required := *parsed.input.Required
			result.Stages = append(result.Stages, domain.PracticeValidationStageSummaryV2{
				StageID: "workspace.required_files", Required: required,
				Outcome: domain.PracticeValidationFail,
				Issues:  []domain.PracticeValidationIssueV2{{Code: "REQUIRED_FILE_MISSING", Message: safePracticeMessage(fmt.Sprintf("Required file %s is missing.", path))}},
			})
			if required {
				passed = false
			}
		}

		contractForExecution := parsed.contract
		contractForExecution.Workspace.RequiredFiles = nil
		codeStructure, marshalErr := json.Marshal(contractForExecution)
		if marshalErr != nil {
			return fail(domain.PracticeValidationAuthoringDefect)
		}
		validationRequest := domain.ValidationRequest{
			TaskID:        request.TaskInstanceID,
			Mode:          domain.ValidationModeFinal,
			Locale:        locale,
			CodeStructure: codeStructure,
			Workspace:     domain.ValidationWorkspace{Files: snapshot.Files},
			TaskMetadata:  metadata,
		}
		validation, executionErrors, executeErr := useCase.orchestrator.executePracticeContract(ctx, validationRequest)
		if executeErr != nil {
			return fail(domain.PracticeValidationAuthoringDefect)
		}
		executionErrorStages := stageExecutionErrorIDs(validation)
		for _, report := range validation.Stages {
			outcome := stageSummaryOutcome(report.Status, report.Passed)
			if executionErrorStages[report.StageID] {
				outcome = domain.PracticeValidationError
			}
			required := *parsed.input.Required && !report.Optional
			issues := safePracticeIssues(report.Errors)
			if executionErrorStages[report.StageID] {
				issues = []domain.PracticeValidationIssueV2{{Code: "STAGE_EXECUTION_ERROR", Message: "Validation could not complete."}}
			}
			if outcome == domain.PracticeValidationFail && len(issues) == 0 {
				issues = []domain.PracticeValidationIssueV2{{Code: "VALIDATION_FAILED", Message: "Validation criteria failed."}}
			}
			result.Stages = append(result.Stages, domain.PracticeValidationStageSummaryV2{
				StageID: safePracticeStageID(report.StageID), Required: required, Outcome: outcome, Issues: issues,
			})
			if required && outcome != domain.PracticeValidationPass {
				passed = false
			}
		}
		for _, report := range validation.Links {
			outcome := domain.PracticeValidationPass
			if !report.Passed {
				outcome = domain.PracticeValidationFail
			}
			required := *parsed.input.Required && !report.Optional
			issues := safePracticeIssues(report.Errors)
			if outcome == domain.PracticeValidationFail && len(issues) == 0 {
				issues = []domain.PracticeValidationIssueV2{{Code: "VALIDATION_FAILED", Message: "Validation criteria failed."}}
			}
			result.Stages = append(result.Stages, domain.PracticeValidationStageSummaryV2{
				StageID: safePracticeStageID("link:" + report.LinkID), Required: required, Outcome: outcome, Issues: issues,
			})
			if required && outcome != domain.PracticeValidationPass {
				passed = false
			}
		}

		if len(executionErrors) > 0 || containsStageExecutionError(validation) {
			result.Outcome = domain.PracticeValidationError
			result.ErrorClassification = classifyPracticeExecutionError(executionErrors)
			return result, nil
		}
		if containsUnsupportedEngine(validation) {
			result.Outcome = domain.PracticeValidationError
			result.ErrorClassification = domain.PracticeValidationUnsupportedConfiguration
			return result, nil
		}
	}

	if passed {
		result.Outcome = domain.PracticeValidationPass
	} else {
		result.Outcome = domain.PracticeValidationFail
	}
	return result, nil
}

func validatePracticeRequestEnvelope(request domain.PracticeValidationRequestV2) error {
	if request.Schema != domain.PracticeValidationRequestSchemaV2 {
		return fmt.Errorf("%w: schema must be %q", domain.ErrInvalidRequest, domain.PracticeValidationRequestSchemaV2)
	}
	for fieldName, value := range map[string]string{
		"task_instance_id":    request.TaskInstanceID,
		"submission_id":       request.SubmissionID,
		"validation_run_id":   request.ValidationRunID,
		"snapshot.sandbox_id": request.Snapshot.SandboxID,
	} {
		if !practiceUUIDPattern.MatchString(value) {
			return fmt.Errorf("%w: %s must be a canonical lowercase UUID", domain.ErrInvalidRequest, fieldName)
		}
	}
	if !practiceDigestPattern.MatchString(request.RevisionDigest) || !practiceDigestPattern.MatchString(request.ValidationContractDigest) {
		return fmt.Errorf("%w: digest fields must be lowercase sha256 digests", domain.ErrInvalidRequest)
	}
	if err := domain.ValidatePracticeSnapshotRefV2(request.Snapshot); err != nil {
		return err
	}
	switch request.Scope {
	case domain.PracticeValidationScopeMilestone:
		if !contractIDPattern.MatchString(request.MilestoneID) {
			return fmt.Errorf("%w: milestone_id is required for MILESTONE scope", domain.ErrInvalidRequest)
		}
	case domain.PracticeValidationScopeFinal:
		if request.MilestoneIDPresent || request.MilestoneID != "" {
			return fmt.Errorf("%w: milestone_id must be omitted for FINAL scope", domain.ErrInvalidRequest)
		}
	default:
		return fmt.Errorf("%w: scope must be MILESTONE or FINAL", domain.ErrInvalidRequest)
	}
	if len(request.ValidationSpecification) == 0 {
		return fmt.Errorf("%w: validation_specification is required", domain.ErrInvalidRequest)
	}
	return nil
}

func decodePracticeValidationSpecification(raw json.RawMessage) (practiceValidationSpecificationV1, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var specification practiceValidationSpecificationV1
	if err := decoder.Decode(&specification); err != nil {
		return practiceValidationSpecificationV1{}, fmt.Errorf("%w: invalid validation_specification fields", domain.ErrInvalidRequest)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return practiceValidationSpecificationV1{}, fmt.Errorf("%w: validation_specification must be one JSON object", domain.ErrInvalidRequest)
	}
	if len(specification.RuntimeProfile) == 0 || len(specification.Contracts) > 100 {
		return practiceValidationSpecificationV1{}, fmt.Errorf("%w: validation_specification requires runtime_profile and a bounded contracts array", domain.ErrInvalidRequest)
	}
	profile, err := decodeCanonicalJSON(specification.RuntimeProfile)
	if err != nil {
		return practiceValidationSpecificationV1{}, fmt.Errorf("%w: runtime_profile must be a JSON object", domain.ErrInvalidRequest)
	}
	if _, ok := profile.(map[string]any); !ok {
		return practiceValidationSpecificationV1{}, fmt.Errorf("%w: runtime_profile must be a JSON object", domain.ErrInvalidRequest)
	}
	return specification, nil
}

func validatePracticeSpecification(
	request domain.PracticeValidationRequestV2,
	specification practiceValidationSpecificationV1,
) ([]practiceValidationContractV1, error) {
	if len(specification.Contracts) == 0 {
		return nil, fmt.Errorf("%w: no executable contracts", domain.ErrInvalidContract)
	}
	seenMilestones := make(map[string]struct{}, len(specification.Contracts))
	for _, contract := range specification.Contracts {
		if contract.MilestoneID == "" || !contractIDPattern.MatchString(contract.MilestoneID) || contract.Required == nil || contract.EvaluationAssets == nil {
			return nil, fmt.Errorf("%w: malformed immutable executable contract", domain.ErrInvalidContract)
		}
		if len(contract.ValidationContract) == 0 || !json.Valid(contract.ValidationContract) {
			return nil, fmt.Errorf("%w: validation_contract must be a JSON object", domain.ErrInvalidContract)
		}
		var object map[string]json.RawMessage
		if err := json.Unmarshal(contract.ValidationContract, &object); err != nil || object == nil {
			return nil, fmt.Errorf("%w: validation_contract must be a JSON object", domain.ErrInvalidContract)
		}
		if _, duplicate := seenMilestones[contract.MilestoneID]; duplicate {
			return nil, fmt.Errorf("%w: duplicate contract milestone %q", domain.ErrInvalidContract, contract.MilestoneID)
		}
		seenMilestones[contract.MilestoneID] = struct{}{}
	}
	if request.Scope == domain.PracticeValidationScopeMilestone {
		if len(specification.Contracts) != 1 || specification.Contracts[0].MilestoneID != request.MilestoneID {
			return nil, fmt.Errorf("%w: MILESTONE scope must contain exactly the requested milestone contract", domain.ErrInvalidContract)
		}
	}
	return specification.Contracts, nil
}

func practiceResultBase(request domain.PracticeValidationRequestV2) domain.PracticeValidationResultV2 {
	return domain.PracticeValidationResultV2{
		Schema:                   domain.PracticeValidationResultSchemaV2,
		TaskInstanceID:           request.TaskInstanceID,
		SubmissionID:             request.SubmissionID,
		ValidationRunID:          request.ValidationRunID,
		RevisionDigest:           request.RevisionDigest,
		ValidationContractDigest: request.ValidationContractDigest,
		Snapshot:                 request.Snapshot,
		Outcome:                  domain.PracticeValidationError,
		Stages:                   []domain.PracticeValidationStageSummaryV2{},
	}
}

func practiceMetadata(raw json.RawMessage) (domain.TaskMetadata, string) {
	var profile map[string]json.RawMessage
	if err := json.Unmarshal(raw, &profile); err != nil {
		return domain.TaskMetadata{}, ""
	}
	stringValue := func(key string) string {
		var value string
		_ = json.Unmarshal(profile[key], &value)
		return value
	}
	var supportsLive bool
	_ = json.Unmarshal(profile["supports_live_validation"], &supportsLive)
	return domain.TaskMetadata{
		TaskKind: stringValue("task_kind"), ExecutionMode: stringValue("execution_mode"),
		EvaluationMode: stringValue("evaluation_mode"), SubmissionMode: stringValue("submission_mode"),
		SupportsLiveValidation: supportsLive,
	}, stringValue("locale")
}

func missingSnapshotFiles(required []string, files []domain.WorkspaceFile) []string {
	available := make(map[string]struct{}, len(files))
	for _, file := range files {
		available[file.Path] = struct{}{}
	}
	missing := make([]string, 0)
	for _, path := range required {
		if _, exists := available[path]; !exists {
			missing = append(missing, path)
		}
	}
	return missing
}

func stageSummaryOutcome(status string, passed bool) domain.PracticeValidationOutcomeV2 {
	if status == "skipped" {
		return domain.PracticeValidationSkipped
	}
	if passed {
		return domain.PracticeValidationPass
	}
	return domain.PracticeValidationFail
}

func safePracticeStageID(value string) string {
	if safePracticeStageIDPattern.MatchString(value) {
		return value
	}
	return "validation"
}

func safePracticeIssues(issues []domain.ValidationIssue) []domain.PracticeValidationIssueV2 {
	result := make([]domain.PracticeValidationIssueV2, 0, len(issues))
	for _, issue := range issues {
		code := issue.Code
		if !safePracticeIssueCodePattern.MatchString(code) {
			code = "VALIDATION_FAILED"
		}
		message := safePracticeMessage(issue.Message)
		if message == "" {
			message = "Validation criteria failed."
		}
		result = append(result, domain.PracticeValidationIssueV2{Code: code, Message: message})
	}
	return result
}

func safePracticeMessage(value string) string {
	if obviousAbsolutePathPattern.MatchString(value) {
		return "Validation criteria failed."
	}
	var builder strings.Builder
	for _, character := range value {
		if character < 0x20 || character == 0x7f || character == 0x2028 || character == 0x2029 {
			continue
		}
		builder.WriteRune(character)
		if builder.Len() >= 1000 {
			break
		}
	}
	return strings.TrimSpace(builder.String())
}

func containsStageExecutionError(result domain.ValidationResult) bool {
	for _, issue := range result.Errors {
		if issue.Code == "STAGE_EXECUTION_ERROR" {
			return true
		}
	}
	for _, stage := range result.Stages {
		for _, issue := range stage.Errors {
			if issue.Code == "STAGE_EXECUTION_ERROR" {
				return true
			}
		}
	}
	return false
}

func stageExecutionErrorIDs(result domain.ValidationResult) map[string]bool {
	ids := make(map[string]bool)
	for _, issue := range result.Errors {
		if issue.Code == "STAGE_EXECUTION_ERROR" && issue.StageID != "" {
			ids[issue.StageID] = true
		}
	}
	for _, stage := range result.Stages {
		for _, issue := range stage.Errors {
			if issue.Code == "STAGE_EXECUTION_ERROR" {
				ids[stage.StageID] = true
			}
		}
	}
	return ids
}

func containsUnsupportedEngine(result domain.ValidationResult) bool {
	for _, issue := range result.Errors {
		if issue.Code == "UNSUPPORTED_ENGINE" {
			return true
		}
	}
	for _, stage := range result.Stages {
		for _, issue := range stage.Errors {
			if issue.Code == "UNSUPPORTED_ENGINE" {
				return true
			}
		}
	}
	return false
}

func classifyPracticeExecutionError(executionErrors []error) string {
	for _, err := range executionErrors {
		if errors.Is(err, domain.ErrUnsupportedEngine) {
			return domain.PracticeValidationUnsupportedConfiguration
		}
		if errors.Is(err, domain.ErrValidatorProtocol) {
			return domain.PracticeValidationValidatorProtocol
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return domain.PracticeValidationValidatorTimeout
		}
		var timeoutError interface{ Timeout() bool }
		if errors.As(err, &timeoutError) && timeoutError.Timeout() {
			return domain.PracticeValidationValidatorTimeout
		}
		var syntaxError *json.SyntaxError
		var typeError *json.UnmarshalTypeError
		if errors.As(err, &syntaxError) || errors.As(err, &typeError) {
			return domain.PracticeValidationValidatorProtocol
		}
		var urlError *url.Error
		if errors.As(err, &urlError) {
			return domain.PracticeValidationValidatorUnavailable
		}
		var networkError net.Error
		if errors.As(err, &networkError) {
			if networkError.Timeout() {
				return domain.PracticeValidationValidatorTimeout
			}
			return domain.PracticeValidationValidatorUnavailable
		}
		var statusError domain.ValidatorHTTPStatusError
		if errors.As(err, &statusError) {
			if statusError.StatusCode == 429 || statusError.StatusCode >= 500 {
				return domain.PracticeValidationValidatorUnavailable
			}
			return domain.PracticeValidationValidatorProtocol
		}
	}
	return domain.PracticeValidationStageExecutionError
}

func snapshotFailureClassification(err error) string {
	switch {
	case errors.Is(err, domain.ErrPinnedSnapshotDigestMismatch):
		return domain.PracticeValidationSnapshotDigestMismatch
	case errors.Is(err, domain.ErrPinnedSnapshotCorrupt):
		return domain.PracticeValidationSnapshotCorrupt
	case errors.Is(err, domain.ErrPinnedSnapshotUnavailable):
		return domain.PracticeValidationSnapshotUnavailable
	default:
		return domain.PracticeValidationSnapshotUnavailable
	}
}
