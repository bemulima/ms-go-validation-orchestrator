package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

const practiceTestDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

type practiceSnapshotReaderStub struct {
	snapshot domain.PinnedSnapshotV2
	err      error
}

func (reader practiceSnapshotReaderStub) ReadPinnedSnapshot(context.Context, domain.PracticeSnapshotRefV2) (domain.PinnedSnapshotV2, error) {
	return reader.snapshot, reader.err
}

type practiceEngineStub struct {
	id     string
	result domain.StageExecutionResult
	err    error
	call   func(context.Context)
}

func (engine practiceEngineStub) EngineID() string {
	if engine.id == "" {
		return "practice.test"
	}
	return engine.id
}

func (engine practiceEngineStub) Validate(ctx context.Context, _ domain.EngineValidationInput) (domain.StageExecutionResult, error) {
	if engine.call != nil {
		engine.call(ctx)
	}
	return engine.result, engine.err
}

func TestPracticeValidationRequiredSemanticFailureFails(t *testing.T) {
	t.Parallel()
	request := practiceRequest(t, true, false)
	useCase := newPracticeUseCase(domain.StageExecutionResult{Passed: false, Errors: []domain.ValidationIssue{{Code: "CHECK_FAILED", Message: "Expected value was missing."}}}, nil, nil)
	result, err := useCase.ValidatePractice(context.Background(), request)
	if err != nil {
		t.Fatalf("ValidatePractice() error = %v", err)
	}
	if result.Outcome != domain.PracticeValidationFail {
		t.Fatalf("outcome = %s, want FAIL", result.Outcome)
	}
}

func TestPracticeValidationOptionalSemanticFailureDoesNotBlock(t *testing.T) {
	t.Parallel()
	request := practiceRequest(t, true, false)
	contract := `{"version":1,"kind":"workspace_contract","stages":[{"id":"required-check","engine":"practice.test","optional":false,"rules":{}},{"id":"optional-check","engine":"practice.optional","optional":true,"rules":{}}]}`
	request.ValidationSpecification = json.RawMessage(strings.Replace(string(request.ValidationSpecification), `{"version":1,"kind":"workspace_contract","stages":[{"id":"check","engine":"practice.test","optional":false,"timeout_seconds":0,"rules":{}}]}`, contract, 1))
	request.ValidationContractDigest, _ = ComputePracticeValidationContractDigest(request.Scope, request.MilestoneID, request.ValidationSpecification)
	useCase := NewPracticeValidationUseCase(NewContractParser(NewDefaultLegacyContractAdapter()), []domain.EngineClient{
		practiceEngineStub{id: "practice.test", result: domain.StageExecutionResult{Passed: true}},
		practiceEngineStub{id: "practice.optional", result: domain.StageExecutionResult{Passed: false, Errors: []domain.ValidationIssue{{Code: "OPTIONAL_FAILED", Message: "Optional check did not pass."}}}},
	}, practiceSnapshotReaderStub{snapshot: testPinnedSnapshot()})
	result, err := useCase.ValidatePractice(context.Background(), request)
	if err != nil {
		t.Fatalf("ValidatePractice() error = %v", err)
	}
	if result.Outcome != domain.PracticeValidationPass {
		t.Fatalf("outcome = %s, want PASS", result.Outcome)
	}
	if len(result.Stages) != 2 {
		t.Fatalf("optional semantic result was not reported safely: %+v", result.Stages)
	}
	var optional *domain.PracticeValidationStageSummaryV2
	for index := range result.Stages {
		if result.Stages[index].StageID == "optional-check" {
			optional = &result.Stages[index]
		}
	}
	if optional == nil || optional.Required || optional.Outcome != domain.PracticeValidationFail {
		t.Fatalf("optional semantic result was not reported safely: %+v", result.Stages)
	}
}

func TestPracticeValidationOptionalExecutionErrorIsError(t *testing.T) {
	t.Parallel()
	request := practiceRequest(t, true, false)
	contract := `{"version":1,"kind":"workspace_contract","stages":[{"id":"required-check","engine":"practice.test","optional":false,"rules":{}},{"id":"optional-check","engine":"practice.optional","optional":true,"rules":{}}]}`
	request.ValidationSpecification = json.RawMessage(strings.Replace(string(request.ValidationSpecification), `{"version":1,"kind":"workspace_contract","stages":[{"id":"check","engine":"practice.test","optional":false,"timeout_seconds":0,"rules":{}}]}`, contract, 1))
	request.ValidationContractDigest, _ = ComputePracticeValidationContractDigest(request.Scope, request.MilestoneID, request.ValidationSpecification)
	useCase := NewPracticeValidationUseCase(NewContractParser(NewDefaultLegacyContractAdapter()), []domain.EngineClient{
		practiceEngineStub{id: "practice.test", result: domain.StageExecutionResult{Passed: true}},
		practiceEngineStub{id: "practice.optional", err: errors.New("validator unavailable")},
	}, practiceSnapshotReaderStub{snapshot: testPinnedSnapshot()})

	result, err := useCase.ValidatePractice(context.Background(), request)
	if err != nil {
		t.Fatalf("ValidatePractice() error = %v", err)
	}
	if result.Outcome != domain.PracticeValidationError || result.ErrorClassification != domain.PracticeValidationStageExecutionError {
		t.Fatalf("outcome/classification = %s/%s", result.Outcome, result.ErrorClassification)
	}
}

func TestPracticeValidationUnsupportedEngineIsError(t *testing.T) {
	t.Parallel()
	request := practiceRequest(t, true, true)
	useCase := NewPracticeValidationUseCase(NewContractParser(NewDefaultLegacyContractAdapter()), nil, practiceSnapshotReaderStub{snapshot: testPinnedSnapshot()})

	result, err := useCase.ValidatePractice(context.Background(), request)
	if err != nil {
		t.Fatalf("ValidatePractice() error = %v", err)
	}
	if result.Outcome != domain.PracticeValidationError || result.ErrorClassification != domain.PracticeValidationUnsupportedConfiguration {
		t.Fatalf("outcome/classification = %s/%s", result.Outcome, result.ErrorClassification)
	}
}

func TestPracticeValidationUnsupportedInputPrerequisiteIsErrorIncludingOptional(t *testing.T) {
	t.Parallel()
	for _, optional := range []bool{false, true} {
		t.Run(fmt.Sprintf("optional=%t", optional), func(t *testing.T) {
			t.Parallel()
			request := practiceRequest(t, true, false)
			contract := `{"version":1,"kind":"workspace_contract","stages":[{"id":"required-check","engine":"practice.test","optional":false,"rules":{}},{"id":"guarded","engine":"practice.guarded","optional":` + boolLiteral(optional) + `,"rules":{}}]}`
			request.ValidationSpecification = json.RawMessage(`{"schema":"practice-validation-specification.v1","runtime_profile":{},"contracts":[{"milestone_id":"step-1","required":true,"validation_contract":` + contract + `,"evaluation_assets":{"negative_fixture_refs":[]}}]}`)
			request.ValidationContractDigest, _ = ComputePracticeValidationContractDigest(request.Scope, request.MilestoneID, request.ValidationSpecification)
			useCase := NewPracticeValidationUseCase(NewContractParser(NewDefaultLegacyContractAdapter()), []domain.EngineClient{
				practiceEngineStub{result: domain.StageExecutionResult{Passed: true}},
				practiceEngineStub{id: "practice.guarded", err: fmt.Errorf("%w: workspace execution prerequisite unavailable", domain.ErrUnsupportedEngine)},
			}, practiceSnapshotReaderStub{snapshot: testPinnedSnapshot()})
			result, err := useCase.ValidatePractice(context.Background(), request)
			if err != nil {
				t.Fatal(err)
			}
			if result.Outcome != domain.PracticeValidationError || result.ErrorClassification != domain.PracticeValidationUnsupportedConfiguration {
				t.Fatalf("unsupported input reported as %s/%s", result.Outcome, result.ErrorClassification)
			}
			for _, stage := range result.Stages {
				if stage.StageID == "guarded" {
					if stage.Outcome != domain.PracticeValidationError || stage.Required == optional {
						t.Fatal("guarded stage lost its execution error or required/optional status")
					}
					return
				}
			}
			t.Fatal("guarded stage missing from safe result")
		})
	}
}

func TestPracticeValidationDigestMismatchIsAuthoringError(t *testing.T) {
	t.Parallel()
	request := practiceRequest(t, true, false)
	request.ValidationContractDigest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	useCase := newPracticeUseCase(domain.StageExecutionResult{Passed: true}, nil, nil)

	result, err := useCase.ValidatePractice(context.Background(), request)
	if err != nil {
		t.Fatalf("ValidatePractice() error = %v", err)
	}
	if result.Outcome != domain.PracticeValidationError || result.ErrorClassification != domain.PracticeValidationAuthoringDefect {
		t.Fatalf("outcome/classification = %s/%s", result.Outcome, result.ErrorClassification)
	}
}

func TestPracticeValidationDigestCanonicalizesObjectKeysAndPreservesArrays(t *testing.T) {
	t.Parallel()
	first := json.RawMessage(`{"schema":"practice-validation-specification.v1","runtime_profile":{"z":1,"a":true},"contracts":[{"milestone_id":"step-1","required":true,"validation_contract":{"kind":"workspace_contract","version":1,"stages":[{"id":"one"},{"id":"two"}]},"evaluation_assets":{"negative_fixture_refs":[]}}]}`)
	second := json.RawMessage(` { "contracts" : [ { "evaluation_assets" : { "negative_fixture_refs" : [] }, "validation_contract" : { "stages" : [ {"id":"one"},{"id":"two"} ], "version" : 1, "kind" : "workspace_contract" }, "required" : true, "milestone_id" : "step-1" } ], "runtime_profile" : {"a":true,"z":1}, "schema" : "practice-validation-specification.v1" } `)
	firstDigest, err := ComputePracticeValidationContractDigest(domain.PracticeValidationScopeMilestone, "step-1", first)
	if err != nil {
		t.Fatal(err)
	}
	secondDigest, err := ComputePracticeValidationContractDigest(domain.PracticeValidationScopeMilestone, "step-1", second)
	if err != nil {
		t.Fatal(err)
	}
	if firstDigest != secondDigest {
		t.Fatalf("object key order changed canonical digest: %s != %s", firstDigest, secondDigest)
	}
	reordered := json.RawMessage(strings.Replace(string(first), `{"id":"one"},{"id":"two"}`, `{"id":"two"},{"id":"one"}`, 1))
	reorderedDigest, err := ComputePracticeValidationContractDigest(domain.PracticeValidationScopeMilestone, "step-1", reordered)
	if err != nil {
		t.Fatal(err)
	}
	if firstDigest == reorderedDigest {
		t.Fatal("array order must remain digest-significant")
	}
}

func TestPracticeValidationSpecificationRejectsUnknownFields(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`{"schema":"practice-validation-specification.v1","runtime_profile":{},"contracts":[],"review_policy":{}}`)
	_, err := decodePracticeValidationSpecification(raw)
	if !errors.Is(err, domain.ErrInvalidRequest) {
		t.Fatalf("error = %v, want invalid request for unknown specification field", err)
	}
}

func TestPracticeValidationSpecificationTreatsAssetReferencesAsOpaque(t *testing.T) {
	t.Parallel()
	request := practiceRequest(t, true, false)
	specification, err := decodePracticeValidationSpecification(request.ValidationSpecification)
	if err != nil {
		t.Fatalf("decode specification: %v", err)
	}
	specification.Contracts[0].EvaluationAssets.TestSuiteRef = " opaque suite ref "
	specification.Contracts[0].EvaluationAssets.NegativeFixtureRefs = []string{"", "fixture ref"}
	if _, err := validatePracticeSpecification(request, specification); err != nil {
		t.Fatalf("opaque asset references should be preserved without interpretation: %v", err)
	}
}

func TestPracticeValidationRejectsFinalScopeWithPresentMilestoneField(t *testing.T) {
	t.Parallel()
	request := practiceRequest(t, true, false)
	request.Scope = domain.PracticeValidationScopeFinal
	request.MilestoneID = ""
	request.MilestoneIDPresent = true
	request.ValidationContractDigest, _ = ComputePracticeValidationContractDigest(request.Scope, "", request.ValidationSpecification)
	useCase := newPracticeUseCase(domain.StageExecutionResult{Passed: true}, nil, nil)
	_, err := useCase.ValidatePractice(context.Background(), request)
	if !errors.Is(err, domain.ErrInvalidRequest) {
		t.Fatalf("error = %v, want invalid request", err)
	}
}

func TestPracticeValidationRedactsUnsafeStageIssueText(t *testing.T) {
	t.Parallel()
	request := practiceRequest(t, true, false)
	useCase := newPracticeUseCase(domain.StageExecutionResult{Passed: false, Errors: []domain.ValidationIssue{{Code: "bad code", Message: "failed in /workspace/private/secret.go"}}}, nil, nil)
	result, err := useCase.ValidatePractice(context.Background(), request)
	if err != nil {
		t.Fatalf("ValidatePractice() error = %v", err)
	}
	if len(result.Stages) != 1 || len(result.Stages[0].Issues) != 1 {
		t.Fatalf("unexpected safe issues: %+v", result.Stages)
	}
	issue := result.Stages[0].Issues[0]
	if issue.Code != "VALIDATION_FAILED" || strings.Contains(issue.Message, "/workspace/") {
		t.Fatalf("unsafe issue was not redacted: %+v", issue)
	}
}

func TestPracticeValidationExecutionFailureClassifications(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		err  error
		want string
	}{
		{name: "timeout", err: context.DeadlineExceeded, want: domain.PracticeValidationValidatorTimeout},
		{name: "transport", err: &url.Error{Op: "Post", URL: "http://validator/validate", Err: errors.New("connection refused")}, want: domain.PracticeValidationValidatorUnavailable},
		{name: "server error", err: domain.ValidatorHTTPStatusError{StatusCode: 503}, want: domain.PracticeValidationValidatorUnavailable},
		{name: "typed protocol", err: domain.ErrValidatorProtocol, want: domain.PracticeValidationValidatorProtocol},
		{name: "protocol", err: &json.SyntaxError{Offset: 1}, want: domain.PracticeValidationValidatorProtocol},
		{name: "unclassified stage error", err: errors.New("validator failed"), want: domain.PracticeValidationStageExecutionError},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := classifyPracticeExecutionError([]error{test.err}); got != test.want {
				t.Fatalf("classification = %s, want %s", got, test.want)
			}
		})
	}
}

func TestPracticeValidationZeroRequiredCriteriaIsAuthoringError(t *testing.T) {
	t.Parallel()
	request := practiceRequest(t, false, false)
	useCase := newPracticeUseCase(domain.StageExecutionResult{Passed: true}, nil, nil)

	result, err := useCase.ValidatePractice(context.Background(), request)
	if err != nil {
		t.Fatalf("ValidatePractice() error = %v", err)
	}
	if result.Outcome != domain.PracticeValidationError || result.ErrorClassification != domain.PracticeValidationAuthoringDefect {
		t.Fatalf("outcome/classification = %s/%s", result.Outcome, result.ErrorClassification)
	}
}

func TestPracticeValidationEmptyContractSetIsAuthoringError(t *testing.T) {
	t.Parallel()
	request := practiceRequest(t, true, false)
	request.ValidationSpecification = json.RawMessage(`{"schema":"practice-validation-specification.v1","runtime_profile":{},"contracts":[]}`)
	request.ValidationContractDigest, _ = ComputePracticeValidationContractDigest(request.Scope, request.MilestoneID, request.ValidationSpecification)
	useCase := newPracticeUseCase(domain.StageExecutionResult{Passed: true}, nil, nil)
	result, err := useCase.ValidatePractice(context.Background(), request)
	if err != nil {
		t.Fatalf("ValidatePractice() error = %v", err)
	}
	if result.Outcome != domain.PracticeValidationError || result.ErrorClassification != domain.PracticeValidationAuthoringDefect {
		t.Fatalf("outcome/classification = %s/%s", result.Outcome, result.ErrorClassification)
	}
}

func TestPracticeValidationUsesExactSnapshotAndMapsReaderFailures(t *testing.T) {
	t.Parallel()
	request := practiceRequest(t, true, false)
	useCase := newPracticeUseCase(domain.StageExecutionResult{Passed: true}, nil, errors.New("snapshot not available"))

	result, err := useCase.ValidatePractice(context.Background(), request)
	if err != nil {
		t.Fatalf("ValidatePractice() error = %v", err)
	}
	if result.Outcome != domain.PracticeValidationError || result.ErrorClassification != domain.PracticeValidationSnapshotUnavailable {
		t.Fatalf("outcome/classification = %s/%s", result.Outcome, result.ErrorClassification)
	}
}

func TestPracticeValidationEnforcesPerStageTimeout(t *testing.T) {
	t.Parallel()
	request := practiceRequest(t, true, false)
	request.ValidationSpecification = json.RawMessage(strings.ReplaceAll(string(request.ValidationSpecification), `"timeout_seconds":0`, `"timeout_seconds":1`))
	request.ValidationContractDigest, _ = ComputePracticeValidationContractDigest(request.Scope, request.MilestoneID, request.ValidationSpecification)
	engine := practiceEngineStub{call: func(ctx context.Context) {
		<-ctx.Done()
	}}
	useCase := NewPracticeValidationUseCase(NewContractParser(NewDefaultLegacyContractAdapter()), []domain.EngineClient{engine}, practiceSnapshotReaderStub{snapshot: testPinnedSnapshot()})

	started := time.Now()
	result, err := useCase.ValidatePractice(context.Background(), request)
	if err != nil {
		t.Fatalf("ValidatePractice() error = %v", err)
	}
	if result.Outcome != domain.PracticeValidationError || result.ErrorClassification != domain.PracticeValidationValidatorTimeout {
		t.Fatalf("outcome/classification = %s/%s", result.Outcome, result.ErrorClassification)
	}
	if time.Since(started) > 3*time.Second {
		t.Fatalf("stage timeout was not enforced, elapsed %s", time.Since(started))
	}
}

func newPracticeUseCase(result domain.StageExecutionResult, engineErr error, snapshotErr error) PracticeValidationUseCase {
	engine := practiceEngineStub{result: result, err: engineErr}
	return NewPracticeValidationUseCase(NewContractParser(NewDefaultLegacyContractAdapter()), []domain.EngineClient{engine}, practiceSnapshotReaderStub{snapshot: testPinnedSnapshot(), err: snapshotErr})
}

func practiceRequest(t *testing.T, required, unsupported bool) domain.PracticeValidationRequestV2 {
	t.Helper()
	engine := "practice.test"
	if unsupported {
		engine = "missing.engine"
	}
	contract := json.RawMessage(`{"version":1,"kind":"workspace_contract","stages":[{"id":"check","engine":"` + engine + `","optional":false,"timeout_seconds":0,"rules":{}}]}`)
	spec := json.RawMessage(`{"schema":"practice-validation-specification.v1","runtime_profile":{},"contracts":[{"milestone_id":"step-1","required":` + boolLiteral(required) + `,"validation_contract":` + string(contract) + `,"evaluation_assets":{"negative_fixture_refs":[]}}]}`)
	digest, err := ComputePracticeValidationContractDigest(domain.PracticeValidationScopeMilestone, "step-1", spec)
	if err != nil {
		t.Fatalf("ComputePracticeValidationContractDigest(): %v", err)
	}
	return domain.PracticeValidationRequestV2{
		Schema:                   domain.PracticeValidationRequestSchemaV2,
		TaskInstanceID:           "11111111-1111-4111-8111-111111111111",
		SubmissionID:             "22222222-2222-4222-8222-222222222222",
		ValidationRunID:          "33333333-3333-4333-8333-333333333333",
		RevisionDigest:           practiceTestDigest,
		ValidationContractDigest: digest,
		Snapshot:                 testPinnedSnapshot().Ref,
		Scope:                    domain.PracticeValidationScopeMilestone,
		MilestoneID:              "step-1",
		ValidationSpecification:  spec,
	}
}

func testPinnedSnapshot() domain.PinnedSnapshotV2 {
	files := []domain.WorkspaceFile{{Path: "main.txt", Content: "student work"}}
	return domain.PinnedSnapshotV2{Ref: domain.PracticeSnapshotRefV2{SandboxID: "44444444-4444-4444-8444-444444444444", SnapshotID: "snapshot-1", GenerationID: "gen-1", WorkspaceDigest: workspaceDigestForTest(files)}, Files: files}
}

func workspaceDigestForTest(files []domain.WorkspaceFile) string {
	digest, err := domain.WorkspaceFilesDigestV2(files)
	if err != nil {
		panic(err)
	}
	return digest
}

func boolLiteral(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
