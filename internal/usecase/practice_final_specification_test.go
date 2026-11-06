package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

const practiceFullFinalContract = `{"version":1,"kind":"workspace_contract","workspace":{"required_files":["main.txt"]},"stages":[{"id":"a-final","engine":"browser.runtime","mode":"final","depends_on":["z-shell"],"rules":{"selector":"<main>&","epsilon":1e-7,"large_integer":9007199254740993}},{"id":"z-shell","engine":"browser.runtime","mode":"both","targets":{"files":["main.txt"]},"rules":{"selector":"private-selector"}},{"id":"live-check","engine":"browser.runtime","mode":"live","rules":{}}],"links":[{"id":"content","kind":"workspace.file_contains","depends_on":["a-final"],"config":{"file":"main.txt","needle":"student work"}}]}`

type practiceFinalEngine struct {
	inputs []domain.EngineValidationInput
	result domain.StageExecutionResult
	err    error
}

func (*practiceFinalEngine) EngineID() string { return "browser.runtime" }
func (engine *practiceFinalEngine) Validate(_ context.Context, input domain.EngineValidationInput) (domain.StageExecutionResult, error) {
	engine.inputs = append(engine.inputs, input)
	return engine.result, engine.err
}

type practiceFinalSnapshotReader struct {
	refs []domain.PracticeSnapshotRefV2
}

func (reader *practiceFinalSnapshotReader) ReadPinnedSnapshot(_ context.Context, ref domain.PracticeSnapshotRefV2) (domain.PinnedSnapshotV2, error) {
	reader.refs = append(reader.refs, ref)
	return testPinnedSnapshot(), nil
}

func practiceFinalRequest(t *testing.T, specification string) domain.PracticeValidationRequestV2 {
	t.Helper()
	request := practiceRequest(t, true, false)
	request.Scope = domain.PracticeValidationScopeFinal
	request.MilestoneID = ""
	request.ValidationSpecification = json.RawMessage(specification)
	digest, err := ComputePracticeValidationContractDigest(request.Scope, "", request.ValidationSpecification)
	if err != nil {
		t.Fatal(err)
	}
	request.ValidationContractDigest = digest
	return request
}

func practiceFinalSpecification(contract string) string {
	return `{"schema":"practice-validation-specification.v2","runtime_profile":{"locale":"en","task_kind":"code"},"final_contract":{"required":true,"validation_contract":` + contract + `,"evaluation_assets":{"test_suite_ref":"private-suite","reference_solution_ref":"private-reference","negative_fixture_refs":["private-negative"],"verification_runner_ref":"private-runner"}}}`
}

func TestPracticeFinalSpecificationExecutesWholeContractOnce(t *testing.T) {
	t.Parallel()
	original := practiceFinalSpecification(practiceFullFinalContract)
	request := practiceFinalRequest(t, original)
	engine := &practiceFinalEngine{result: domain.StageExecutionResult{Passed: true}}
	reader := &practiceFinalSnapshotReader{}
	useCase := NewPracticeValidationUseCase(NewContractParser(NewDefaultLegacyContractAdapter()), []domain.EngineClient{engine}, reader)
	result, err := useCase.ValidatePractice(context.Background(), request)
	if err != nil || result.Outcome != domain.PracticeValidationPass {
		t.Fatalf("got %s/%s, error %v", result.Outcome, result.ErrorClassification, err)
	}
	if len(reader.refs) != 1 || reader.refs[0] != request.Snapshot {
		t.Fatalf("exact snapshot reads = %+v", reader.refs)
	}
	if len(engine.inputs) != 2 || engine.inputs[0].Stage.ID != "z-shell" || engine.inputs[1].Stage.ID != "a-final" {
		t.Fatalf("whole final contract was not executed once in dependency order: %+v", engine.inputs)
	}
	for _, input := range engine.inputs {
		if input.TaskID != request.TaskInstanceID || input.Mode != domain.ValidationModeFinal || input.Locale != "en" || input.TaskMetadata.TaskKind != "code" ||
			input.Workspace.RootPath != "" || !reflect.DeepEqual(input.Workspace.Files, testPinnedSnapshot().Files) {
			t.Fatalf("execution lost immutable input or runtime metadata: %+v", input)
		}
	}
	rules, err := decodeCanonicalJSON(engine.inputs[1].Stage.Rules)
	if err != nil {
		t.Fatal(err)
	}
	wantRules, _ := decodeCanonicalJSON([]byte(`{"selector":"<main>&","epsilon":1e-7,"large_integer":9007199254740993}`))
	if !reflect.DeepEqual(rules, wantRules) || string(request.ValidationSpecification) != original {
		t.Fatal("full contract rules or immutable specification changed")
	}
	if len(result.Stages) != 3 || result.Stages[0].StageID != "z-shell" || result.Stages[1].StageID != "a-final" || result.Stages[2].StageID != "link:content" {
		t.Fatalf("original stage/link identities not retained: %+v", result.Stages)
	}
	for _, stage := range result.Stages {
		if !stage.Required || stage.Outcome != domain.PracticeValidationPass {
			t.Fatalf("required final criterion lost: %+v", stage)
		}
	}
	if result.Schema != domain.PracticeValidationResultSchemaV2 || result.TaskInstanceID != request.TaskInstanceID ||
		result.SubmissionID != request.SubmissionID || result.ValidationRunID != request.ValidationRunID ||
		result.RevisionDigest != request.RevisionDigest || result.ValidationContractDigest != request.ValidationContractDigest || result.Snapshot != request.Snapshot {
		t.Fatal("final response lost correlation")
	}
	safe, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"milestone_id", "private-", `"validation_contract":`, "evaluation_assets", "selector", "rules", "large_integer"} {
		if strings.Contains(string(safe), forbidden) {
			t.Fatalf("private final input leaked into response: %s", forbidden)
		}
	}
}

func TestPracticeFinalSpecificationRejectsInvalidInputsBeforeIO(t *testing.T) {
	t.Parallel()
	valid := practiceFinalSpecification(practiceFullFinalContract)
	for _, test := range []struct {
		name, old, replacement string
		requestError           bool
	}{
		{"v1 cannot accept final_contract", "practice-validation-specification.v2", "practice-validation-specification.v1", true},
		{"schema alias", `"schema":`, `"Schema":`, true},
		{"duplicated schema", `"schema":`, `"schema":"practice-validation-specification.v2","schema":`, true},
		{"schema case collision", `"schema":`, `"Schema":"practice-validation-specification.v1","schema":`, true},
		{"final contract alias", `"final_contract":`, `"final_Contract":`, true},
		{"duplicated final contract", `"final_contract":`, `"final_contract":null,"final_contract":`, true},
		{"duplicated runtime profile", `"runtime_profile":`, `"runtime_profile":{},"runtime_profile":`, true},
		{"required alias", `"required":true`, `"Required":true`, true},
		{"required case collision", `"required":true`, `"required":false,"Required":true`, true},
		{"duplicated required", `"required":true`, `"required":false,"required":true`, true},
		{"assets alias", `"evaluation_assets":`, `"Evaluation_Assets":`, true},
		{"duplicated assets", `"evaluation_assets":`, `"evaluation_assets":{},"evaluation_assets":`, true},
		{"asset reference alias", `"test_suite_ref":`, `"Test_Suite_Ref":`, true},
		{"duplicated asset reference", `"test_suite_ref":`, `"test_suite_ref":"other-private-suite","test_suite_ref":`, true},
		{"mixed contracts", `"final_contract":`, `"contracts":[],"final_contract":`, true},
		{"null contracts still forbidden", `"final_contract":`, `"contracts":null,"final_contract":`, true},
		{"spec milestone forbidden", `"final_contract":`, `"milestone_id":"invented","final_contract":`, true},
		{"contract milestone forbidden", `"required":true`, `"milestone_id":"invented","required":true`, true},
		{"review policy forbidden", `"required":true`, `"review_policy":{},"required":true`, true},
		{"unknown asset", `"test_suite_ref":`, `"secret":true,"test_suite_ref":`, true},
		{"profile missing", `"runtime_profile":{"locale":"en","task_kind":"code"},`, "", true},
		{"profile null", `{"locale":"en","task_kind":"code"}`, "null", true},
		{"profile array", `{"locale":"en","task_kind":"code"}`, "[]", true},
		{"required missing", `"required":true,`, "", false},
		{"required null", `"required":true`, `"required":null`, false},
		{"required false", `"required":true`, `"required":false`, false},
		{"contract null", practiceFullFinalContract, "null", false},
		{"contract array", practiceFullFinalContract, "[]", false},
		{"contract scalar", practiceFullFinalContract, "42", false},
		{"legacy contract", practiceFullFinalContract, "{}", false},
		{"assets null", `{"test_suite_ref":"private-suite","reference_solution_ref":"private-reference","negative_fixture_refs":["private-negative"],"verification_runner_ref":"private-runner"}`, "null", false},
		{"assets missing", `,"evaluation_assets":{"test_suite_ref":"private-suite","reference_solution_ref":"private-reference","negative_fixture_refs":["private-negative"],"verification_runner_ref":"private-runner"}`, "", false},
		{"final contract missing", valid, `{"schema":"practice-validation-specification.v2","runtime_profile":{}}`, false},
		{"final contract null", valid, `{"schema":"practice-validation-specification.v2","runtime_profile":{},"final_contract":null}`, false},
		{"invalid full contract field", `"kind":"workspace_contract"`, `"kind":"workspace_contract","hidden_rubric":{}`, false},
		{"no final stages", practiceFullFinalContract, `{"version":1,"kind":"workspace_contract","stages":[{"id":"live","engine":"browser.runtime","mode":"live"}]}`, false},
		{"zero required criteria", practiceFullFinalContract, `{"version":1,"kind":"workspace_contract","stages":[{"id":"optional","engine":"browser.runtime","optional":true}]}`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if !strings.Contains(valid, test.old) {
				t.Fatal("invalid test mutation")
			}
			request := practiceFinalRequest(t, strings.Replace(valid, test.old, test.replacement, 1))
			engine := &practiceFinalEngine{result: domain.StageExecutionResult{Passed: true}}
			reader := &practiceFinalSnapshotReader{}
			useCase := NewPracticeValidationUseCase(NewContractParser(NewDefaultLegacyContractAdapter()), []domain.EngineClient{engine}, reader)
			result, err := useCase.ValidatePractice(context.Background(), request)
			if test.requestError {
				if !errors.Is(err, domain.ErrInvalidRequest) {
					t.Fatalf("error = %v, want invalid request", err)
				}
			} else if err != nil || result.Outcome != domain.PracticeValidationError || result.ErrorClassification != domain.PracticeValidationAuthoringDefect {
				t.Fatalf("got %s/%s, error %v, want authoring ERROR", result.Outcome, result.ErrorClassification, err)
			}
			if len(reader.refs) != 0 || len(engine.inputs) != 0 {
				t.Fatal("invalid specification performed I/O")
			}
		})
	}
}

func TestPracticeFinalSpecificationLeavesEngineRuleKeysOpaque(t *testing.T) {
	t.Parallel()
	contract := strings.Replace(practiceFullFinalContract, `"epsilon":1e-7`, `"epsilon":1e-7,"Schema":"engine-owned","Required":false,"schema":"different-engine-key"`, 1)
	engine := &practiceFinalEngine{result: domain.StageExecutionResult{Passed: true}}
	useCase := NewPracticeValidationUseCase(NewContractParser(NewDefaultLegacyContractAdapter()), []domain.EngineClient{engine}, practiceSnapshotReaderStub{snapshot: testPinnedSnapshot()})
	result, err := useCase.ValidatePractice(context.Background(), practiceFinalRequest(t, practiceFinalSpecification(contract)))
	if err != nil || result.Outcome != domain.PracticeValidationPass || len(engine.inputs) != 2 {
		t.Fatalf("opaque engine rules rejected: %s/%s error %v", result.Outcome, result.ErrorClassification, err)
	}
}

func TestPracticeFinalSpecificationPreservesV1ExecutionAndFeedback(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name           string
		result         domain.StageExecutionResult
		err            error
		want           domain.PracticeValidationOutcomeV2
		classification string
	}{
		{name: "pass", result: domain.StageExecutionResult{Passed: true}, want: domain.PracticeValidationPass},
		{name: "semantic failure", result: domain.StageExecutionResult{Errors: []domain.ValidationIssue{{Code: "CHECK_FAILED", Message: "Expected criterion missing."}}}, want: domain.PracticeValidationFail},
		{name: "private execution error", err: errors.New("private fixture /workspaces/secret"), want: domain.PracticeValidationError, classification: domain.PracticeValidationStageExecutionError},
		{name: "timeout", err: context.DeadlineExceeded, want: domain.PracticeValidationError, classification: domain.PracticeValidationValidatorTimeout},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			v1 := `{"schema":"practice-validation-specification.v1","runtime_profile":{"locale":"en","task_kind":"code"},"contracts":[{"milestone_id":"existing-authored-step","required":true,"validation_contract":` + practiceFullFinalContract + `,"evaluation_assets":{}}]}`
			v2 := strings.Replace(practiceFinalSpecification(practiceFullFinalContract), `{"test_suite_ref":"private-suite","reference_solution_ref":"private-reference","negative_fixture_refs":["private-negative"],"verification_runner_ref":"private-runner"}`, "{}", 1)
			var previous domain.PracticeValidationResultV2
			var previousInputs []domain.EngineValidationInput
			for index, specification := range []string{v1, v2} {
				request := practiceFinalRequest(t, specification)
				engine := &practiceFinalEngine{result: test.result, err: test.err}
				useCase := NewPracticeValidationUseCase(NewContractParser(NewDefaultLegacyContractAdapter()), []domain.EngineClient{engine}, practiceSnapshotReaderStub{snapshot: testPinnedSnapshot()})
				result, err := useCase.ValidatePractice(context.Background(), request)
				if err != nil || result.Outcome != test.want || result.ErrorClassification != test.classification {
					t.Fatalf("version %d got %s/%s error %v", index+1, result.Outcome, result.ErrorClassification, err)
				}
				if index == 1 && (!reflect.DeepEqual(previous.Stages, result.Stages) || !reflect.DeepEqual(previousInputs, engine.inputs)) {
					t.Fatal("V1 and V2 changed execution or safe feedback for identical full contract")
				}
				if test.err != nil {
					encoded, _ := json.Marshal(result)
					if strings.Contains(string(encoded), "private") || strings.Contains(string(encoded), "/workspaces/") {
						t.Fatal("provider failure leaked private details")
					}
				}
				previous, previousInputs = result, engine.inputs
			}
		})
	}
}

func TestPracticeFinalSpecificationKeepsOptionalAndGoAssetRules(t *testing.T) {
	t.Parallel()
	contract := `{"version":1,"kind":"workspace_contract","stages":[{"id":"required","engine":"practice.test"},{"id":"optional","engine":"practice.optional","optional":true}]}`
	for _, executionError := range []bool{false, true} {
		t.Run(boolLiteral(executionError), func(t *testing.T) {
			t.Parallel()
			var engineErr error
			want := domain.PracticeValidationPass
			if executionError {
				engineErr = domain.ErrValidatorProtocol
				want = domain.PracticeValidationError
			}
			request := practiceFinalRequest(t, practiceFinalSpecification(contract))
			useCase := NewPracticeValidationUseCase(NewContractParser(NewDefaultLegacyContractAdapter()), []domain.EngineClient{
				practiceEngineStub{result: domain.StageExecutionResult{Passed: true}},
				practiceEngineStub{id: "practice.optional", result: domain.StageExecutionResult{Passed: false}, err: engineErr},
			}, practiceSnapshotReaderStub{snapshot: testPinnedSnapshot()})
			result, err := useCase.ValidatePractice(context.Background(), request)
			if err != nil || result.Outcome != want {
				t.Fatalf("got %s/%s error %v", result.Outcome, result.ErrorClassification, err)
			}
			if executionError && result.ErrorClassification != domain.PracticeValidationValidatorProtocol {
				t.Fatal("optional execution failure lost typed classification")
			}
		})
	}
	t.Run("missing immutable Go assets", func(t *testing.T) {
		t.Parallel()
		contract := `{"version":1,"kind":"workspace_contract","stages":[{"id":"go","engine":"go.tests.v1","mode":"final","checks":{"profile_id":"go-double-numbers.v1"}}]}`
		spec := `{"schema":"practice-validation-specification.v2","runtime_profile":{},"final_contract":{"required":true,"validation_contract":` + contract + `,"evaluation_assets":{}}}`
		_, fake, _ := executableFixture(t)
		called := false
		engine := goBindingProviderStub{manifest: fake.result.Manifest, practiceEngineStub: practiceEngineStub{call: func(context.Context) { called = true }}}
		useCase := NewPracticeValidationUseCase(NewContractParser(NewDefaultLegacyContractAdapter()), []domain.EngineClient{engine}, practiceSnapshotReaderStub{snapshot: testPinnedSnapshot()})
		result, err := useCase.ValidatePractice(context.Background(), practiceFinalRequest(t, spec))
		if err != nil || result.Outcome != domain.PracticeValidationError || result.ErrorClassification != domain.PracticeValidationAuthoringDefect || called {
			t.Fatalf("missing frozen Go assets bypassed owner binding: %s/%s error %v", result.Outcome, result.ErrorClassification, err)
		}
	})
}

func TestPracticeFinalSpecificationCannotValidateMilestone(t *testing.T) {
	t.Parallel()
	request := practiceFinalRequest(t, practiceFinalSpecification(practiceFullFinalContract))
	request.Scope, request.MilestoneID = domain.PracticeValidationScopeMilestone, "authored-step"
	request.ValidationContractDigest, _ = ComputePracticeValidationContractDigest(request.Scope, request.MilestoneID, request.ValidationSpecification)
	reader := &practiceFinalSnapshotReader{}
	useCase := NewPracticeValidationUseCase(NewContractParser(NewDefaultLegacyContractAdapter()), nil, reader)
	_, err := useCase.ValidatePractice(context.Background(), request)
	if !errors.Is(err, domain.ErrInvalidRequest) || len(reader.refs) != 0 {
		t.Fatalf("v2 MILESTONE accepted or performed I/O: error %v", err)
	}
}

func TestPracticeFinalSpecificationDigestMismatchDoesNotReadSnapshot(t *testing.T) {
	t.Parallel()
	request := practiceFinalRequest(t, practiceFinalSpecification(practiceFullFinalContract))
	request.ValidationSpecification = json.RawMessage(strings.Replace(string(request.ValidationSpecification), "1e-7", "1e-6", 1))
	reader := &practiceFinalSnapshotReader{}
	useCase := NewPracticeValidationUseCase(NewContractParser(NewDefaultLegacyContractAdapter()), nil, reader)
	result, err := useCase.ValidatePractice(context.Background(), request)
	if err != nil || result.Outcome != domain.PracticeValidationError || result.ErrorClassification != domain.PracticeValidationAuthoringDefect || len(reader.refs) != 0 {
		t.Fatalf("digest drift accepted or performed I/O: %+v, error %v", result, err)
	}
}

func TestPracticeFinalSpecificationKeepsWholeWorkspaceAndLinkCriteria(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, old, replacement, failedStage string
	}{
		{"workspace", `"required_files":["main.txt"]`, `"required_files":["missing.txt"]`, "workspace.required_files"},
		{"link", `"needle":"student work"`, `"needle":"required content absent"`, "link:content"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			contract := strings.Replace(practiceFullFinalContract, test.old, test.replacement, 1)
			engine := &practiceFinalEngine{result: domain.StageExecutionResult{Passed: true}}
			useCase := NewPracticeValidationUseCase(NewContractParser(NewDefaultLegacyContractAdapter()), []domain.EngineClient{engine}, practiceSnapshotReaderStub{snapshot: testPinnedSnapshot()})
			result, err := useCase.ValidatePractice(context.Background(), practiceFinalRequest(t, practiceFinalSpecification(contract)))
			if err != nil || result.Outcome != domain.PracticeValidationFail || len(engine.inputs) != 2 {
				t.Fatalf("whole contract criterion dropped: %s/%s error %v", result.Outcome, result.ErrorClassification, err)
			}
			for _, stage := range result.Stages {
				if stage.StageID == test.failedStage && stage.Required && stage.Outcome == domain.PracticeValidationFail {
					return
				}
			}
			t.Fatalf("missing required criterion result %s: %+v", test.failedStage, result.Stages)
		})
	}
}
