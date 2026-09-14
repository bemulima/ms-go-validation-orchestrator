package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

func TestTeacherExplanationForPassingResultIsEmptyAndDigestBound(t *testing.T) {
	result := domain.ValidationResult{ContractKind: "workspace_contract", ContractVersion: 1, Passed: true, Stages: []domain.StageReport{}}
	explanation := BuildTeacherValidationExplanation(result)
	if explanation.Schema != domain.TeacherValidationExplanationSchemaV1 || !explanation.Passed ||
		len(explanation.BlockingIssues) != 0 || explanation.Truncated || !strings.HasPrefix(explanation.SourceDigest, "sha256:") {
		t.Fatalf("unexpected passing explanation: %+v", explanation)
	}
}

func TestTeacherExplanationKeepsOnlyUniqueBlockingIssuesInExecutionOrder(t *testing.T) {
	required := domain.ValidationIssue{
		Code: "MISSING_HANDLER", Message: "Implement the handler.", Hint: "Start with ServeHTTP.",
		File: "internal/http/handler.go", Line: 14, Column: 3, StageID: "spoofed", Engine: "spoofed",
		Severity: "error", Selector: "#private", Route: "/hidden", Symbol: "secret", Property: "private",
	}
	optional := domain.ValidationIssue{Code: "OPTIONAL_STYLE", Message: "Optional style check.", Severity: "error"}
	result := domain.ValidationResult{
		ContractKind: "workspace_contract", ContractVersion: 1, Passed: false,
		Stages: []domain.StageReport{
			{StageID: "required", Engine: "go.core", Passed: false, Errors: []domain.ValidationIssue{required, required}},
			{StageID: "optional", Engine: "css.ast", Passed: false, Optional: true, Errors: []domain.ValidationIssue{optional}},
			{StageID: "warning", Engine: "html.dom", Passed: true, Warnings: []domain.ValidationIssue{{Code: "WARN", Message: "warning", Severity: "warning"}}},
		},
		Errors: []domain.ValidationIssue{required, required, optional},
	}
	explanation := BuildTeacherValidationExplanation(result)
	if explanation.Passed || len(explanation.BlockingIssues) != 1 {
		t.Fatalf("unexpected blocking issues: %+v", explanation)
	}
	issue := explanation.BlockingIssues[0]
	if issue.Code != required.Code || issue.StageID != "required" || issue.Engine != "go.core" || issue.File != required.File || issue.Line != 14 || issue.Column != 3 {
		t.Fatalf("normalized issue lost safe fields: %+v", issue)
	}
	payload, err := json.Marshal(explanation)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"OPTIONAL_STYLE", "WARN", "#private", "/hidden", "secret", "property", "selector", "route", "symbol"} {
		if strings.Contains(string(payload), forbidden) {
			t.Fatalf("safe projection leaked %q: %s", forbidden, payload)
		}
	}
}

func TestTeacherExplanationDropsOptionalExecutionFailureFromMixedResult(t *testing.T) {
	required := domain.ValidationIssue{Code: "REQUIRED_FAILURE", Message: "Required check failed.", Severity: "error", StageID: "required", Engine: "go.core"}
	optionalExecution := domain.ValidationIssue{Code: "STAGE_EXECUTION_ERROR", Message: "private transport failure", Severity: "error", StageID: "optional", Engine: "css.ast"}
	result := domain.ValidationResult{
		Passed: false,
		Stages: []domain.StageReport{
			{StageID: "required", Engine: "go.core", Passed: false, Errors: []domain.ValidationIssue{required}},
			{StageID: "optional", Engine: "css.ast", Passed: false, Optional: true},
		},
		Errors: []domain.ValidationIssue{required, optionalExecution},
	}
	explanation := BuildTeacherValidationExplanation(result)
	if len(explanation.BlockingIssues) != 1 || explanation.BlockingIssues[0].Code != "REQUIRED_FAILURE" {
		t.Fatalf("optional execution issue entered blocking projection: %+v", explanation)
	}
}

func TestTeacherExplanationCapsDeduplicatesAndTruncatesDeterministically(t *testing.T) {
	errorsList := make([]domain.ValidationIssue, 0, 8)
	for index := 0; index < 7; index++ {
		errorsList = append(errorsList, domain.ValidationIssue{Code: "ISSUE_" + string(rune('A'+index)), Message: "blocking issue", Severity: "error"})
	}
	errorsList = append(errorsList, errorsList[0])
	result := domain.ValidationResult{Passed: false, Stages: []domain.StageReport{{StageID: "stage", Engine: "go.core", Passed: false, Errors: errorsList}}}
	first := BuildTeacherValidationExplanation(result)
	second := BuildTeacherValidationExplanation(result)
	if len(first.BlockingIssues) != maximumTeacherBlockingIssues || !first.Truncated || first.SourceDigest != second.SourceDigest {
		t.Fatalf("unexpected bounded explanation: first=%+v second=%+v", first, second)
	}
	for index, issue := range first.BlockingIssues {
		if issue.Code != "ISSUE_"+string(rune('A'+index)) {
			t.Fatalf("unstable issue order at %d: %+v", index, first.BlockingIssues)
		}
	}
	payload, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) > maximumTeacherExplanationBytes {
		t.Fatalf("projection exceeds cap: %d", len(payload))
	}
}

func TestTeacherExplanationSanitizesUnicodeControlsPathsAndLocations(t *testing.T) {
	result := domain.ValidationResult{
		Passed: false,
		Stages: []domain.StageReport{{
			StageID: strings.Repeat("s", 140), Engine: strings.Repeat("e", 140), Passed: false,
			Errors: []domain.ValidationIssue{{
				Code: strings.Repeat("C", 140), Message: strings.Repeat("界", 1100) + "\n\x00hidden",
				Hint: strings.Repeat("🙂", 600), File: "../../private/fixture.txt", Line: -1, Column: maximumTeacherSourceLocation + 1,
				Severity: "error",
			}},
		}},
	}
	explanation := BuildTeacherValidationExplanation(result)
	if len(explanation.BlockingIssues) != 1 || !explanation.Truncated {
		t.Fatalf("unsafe issue was not safely bounded: %+v", explanation)
	}
	issue := explanation.BlockingIssues[0]
	if utf8.RuneCountInString(issue.Code) > maximumTeacherCodeRunes || utf8.RuneCountInString(issue.Message) > maximumTeacherMessageRunes ||
		utf8.RuneCountInString(issue.Hint) > maximumTeacherHintRunes || utf8.RuneCountInString(issue.StageID) > maximumTeacherIdentityRunes ||
		utf8.RuneCountInString(issue.Engine) > maximumTeacherIdentityRunes || issue.File != "" || issue.Line != 0 || issue.Column != 0 {
		t.Fatalf("unsafe issue fields survived: %+v", issue)
	}
	for _, value := range []string{issue.Code, issue.Message, issue.Hint, issue.StageID, issue.Engine} {
		for _, character := range value {
			if unicode.IsControl(character) || unicode.Is(unicode.Cf, character) {
				t.Fatalf("control character survived in %q", value)
			}
		}
	}
	payload, err := json.Marshal(explanation)
	if err != nil || len(payload) > maximumTeacherExplanationBytes {
		t.Fatalf("serialized cap failed: bytes=%d err=%v", len(payload), err)
	}
}

func TestTeacherExplanationRedactsAbsolutePathsInsideMessageAndHint(t *testing.T) {
	for _, secretPath := range []string{"/workspaces/private-fixture/solution.go", `C:\private\fixture.txt`, `\\server\share\fixture.txt`} {
		result := domain.ValidationResult{
			Passed: false,
			Stages: []domain.StageReport{{StageID: "go", Engine: "go.core", Passed: false, Errors: []domain.ValidationIssue{{
				Code: "FAIL", Message: "engine inspected " + secretPath, Hint: "copy " + secretPath, Severity: "error",
			}}}},
		}
		explanation := BuildTeacherValidationExplanation(result)
		if len(explanation.BlockingIssues) != 1 || explanation.BlockingIssues[0].Message != genericValidationIssueMessage || explanation.BlockingIssues[0].Hint != "" || !explanation.Truncated {
			t.Fatalf("path %q was not redacted: %+v", secretPath, explanation)
		}
		payload, err := json.Marshal(explanation)
		if err != nil || strings.Contains(string(payload), secretPath) {
			t.Fatalf("path %q leaked: %s err=%v", secretPath, payload, err)
		}
	}
}

func TestTeacherExplanationEnforcesSerializedByteCap(t *testing.T) {
	issues := make([]domain.ValidationIssue, 0, maximumTeacherBlockingIssues)
	for index := 0; index < maximumTeacherBlockingIssues; index++ {
		issues = append(issues, domain.ValidationIssue{
			Code: "FAIL_" + string(rune('A'+index)), Message: strings.Repeat("🙂", maximumTeacherMessageRunes),
			Hint: strings.Repeat("界", maximumTeacherHintRunes), Severity: "error",
		})
	}
	explanation := BuildTeacherValidationExplanation(domain.ValidationResult{
		Passed: false, Stages: []domain.StageReport{{StageID: "stage", Engine: "go.core", Passed: false, Errors: issues}},
	})
	payload, err := json.Marshal(explanation)
	if err != nil || len(payload) > maximumTeacherExplanationBytes || len(explanation.BlockingIssues) >= maximumTeacherBlockingIssues || !explanation.Truncated {
		t.Fatalf("serialized cap failed: issues=%d bytes=%d truncated=%t err=%v", len(explanation.BlockingIssues), len(payload), explanation.Truncated, err)
	}
}

func TestTeacherExplanationDigestExcludesRawOutputAndProjectionItself(t *testing.T) {
	result := domain.ValidationResult{
		Passed: false,
		Stages: []domain.StageReport{{
			StageID: "go", Engine: "go.core", Passed: false,
			Errors:    []domain.ValidationIssue{{Code: "FAIL", Message: "safe", Severity: "error"}},
			RawResult: []byte("RAW_PROVIDER_SECRET"),
		}},
	}
	first := BuildTeacherValidationExplanation(result)
	result.Stages[0].RawResult = []byte("ANOTHER_RAW_SECRET")
	result.TeacherExplanation = &domain.TeacherValidationExplanationV1{Schema: "untrusted"}
	second := BuildTeacherValidationExplanation(result)
	if first.SourceDigest != second.SourceDigest {
		t.Fatalf("non-serialized source changed digest: %s != %s", first.SourceDigest, second.SourceDigest)
	}
	result.Stages[0].Errors[0].Message = "different normalized issue"
	third := BuildTeacherValidationExplanation(result)
	if third.SourceDigest == first.SourceDigest {
		t.Fatal("normalized source change did not alter digest")
	}
	payload, err := json.Marshal(first)
	if err != nil || strings.Contains(string(payload), "SECRET") {
		t.Fatalf("raw result leaked: %s err=%v", payload, err)
	}
}

type explanationExecutionFailureEngine struct{}

func (explanationExecutionFailureEngine) EngineID() string { return "go.core" }
func (explanationExecutionFailureEngine) Validate(context.Context, domain.EngineValidationInput) (domain.StageExecutionResult, error) {
	return domain.StageExecutionResult{RawResult: []byte("RAW_ENGINE_SECRET")}, errors.New("transport token=ENGINE_SECRET")
}

type explanationPassingEngine struct{}

func (explanationPassingEngine) EngineID() string { return "go.core" }
func (explanationPassingEngine) Validate(context.Context, domain.EngineValidationInput) (domain.StageExecutionResult, error) {
	return domain.StageExecutionResult{Passed: true}, nil
}

func TestExecuteAttachesPassingTeacherExplanationWithoutChangingResult(t *testing.T) {
	contract := json.RawMessage(`{"version":1,"kind":"workspace_contract","stages":[{"id":"go","engine":"go.core"}]}`)
	useCase := NewOrchestrateValidationUseCase(NewContractParser(NewDefaultLegacyContractAdapter()), []domain.EngineClient{explanationPassingEngine{}})
	result, err := useCase.Execute(context.Background(), domain.ValidationRequest{CodeStructure: contract})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed || result.TeacherExplanation == nil || !result.TeacherExplanation.Passed || len(result.TeacherExplanation.BlockingIssues) != 0 {
		t.Fatalf("passing result changed by projection: %+v", result)
	}
}

func TestExecuteRedactsInfrastructureFailureFromResultAndTeacherProjection(t *testing.T) {
	contract := json.RawMessage(`{"version":1,"kind":"workspace_contract","stages":[{"id":"go","engine":"go.core"}]}`)
	useCase := NewOrchestrateValidationUseCase(NewContractParser(NewDefaultLegacyContractAdapter()), []domain.EngineClient{explanationExecutionFailureEngine{}})
	result, err := useCase.Execute(context.Background(), domain.ValidationRequest{CodeStructure: contract})
	if err != nil {
		t.Fatal(err)
	}
	if result.Passed || len(result.Errors) != 1 || result.Errors[0].Message != genericStageExecutionMessage || result.TeacherExplanation == nil || len(result.TeacherExplanation.BlockingIssues) != 1 {
		t.Fatalf("execution failure was not safely projected: %+v", result)
	}
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"ENGINE_SECRET", "RAW_ENGINE_SECRET", "transport token="} {
		if strings.Contains(string(payload), secret) {
			t.Fatalf("execution secret leaked in normalized response: %s", payload)
		}
	}
}
