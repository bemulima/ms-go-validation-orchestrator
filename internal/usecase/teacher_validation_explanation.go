package usecase

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

const (
	maximumTeacherBlockingIssues   = 5
	maximumTeacherExplanationBytes = 16 << 10
	maximumTeacherCodeRunes        = 128
	maximumTeacherIdentityRunes    = 128
	maximumTeacherMessageRunes     = 1000
	maximumTeacherHintRunes        = 500
	maximumTeacherFileRunes        = 512
	maximumTeacherSourceLocation   = 1_000_000
	genericStageExecutionMessage   = "A validation engine could not complete this check."
	genericStageExecutionHint      = "Retry validation. If it still fails, ask for help."
	genericValidationIssueMessage  = "A required validation check was not satisfied."
)

var obviousAbsolutePathPattern = regexp.MustCompile(`(?i)(^|[[:space:]"'(:=])(/[[:graph:]]+|[a-z]:[\\/][[:graph:]]+|\\\\[[:graph:]]+|//[[:graph:]]+)`)

type teacherExplanationBuilder struct {
	explanation domain.TeacherValidationExplanationV1
	seen        map[domain.TeacherValidationIssueV1]struct{}
}

// BuildTeacherValidationExplanation creates a bounded presentation projection
// from the normalized result only. Stage RawResult is neither hashed nor
// copied. Normalized evidence is digest-bound but cannot enter issue output.
func BuildTeacherValidationExplanation(result domain.ValidationResult) domain.TeacherValidationExplanationV1 {
	source := result
	source.TeacherExplanation = nil
	encoded, _ := json.Marshal(source)
	digest := sha256.Sum256(encoded)
	builder := teacherExplanationBuilder{
		explanation: domain.TeacherValidationExplanationV1{
			Schema:         domain.TeacherValidationExplanationSchemaV1,
			Passed:         result.Passed,
			BlockingIssues: []domain.TeacherValidationIssueV1{},
			SourceDigest:   "sha256:" + hex.EncodeToString(digest[:]),
		},
		seen: make(map[domain.TeacherValidationIssueV1]struct{}),
	}
	if result.Passed {
		return builder.explanation
	}

	stageByID := make(map[string]domain.StageReport, len(result.Stages))
	executionFailures := make(map[string][]domain.ValidationIssue)
	for _, issue := range result.Errors {
		if issue.Code == "STAGE_EXECUTION_ERROR" {
			executionFailures[issue.StageID] = append(executionFailures[issue.StageID], issue)
		}
	}
	for _, stage := range result.Stages {
		stageByID[stage.StageID] = stage
		if stage.Optional {
			continue
		}
		for range executionFailures[stage.StageID] {
			builder.add(domain.ValidationIssue{
				Code: "STAGE_EXECUTION_ERROR", Message: genericStageExecutionMessage,
				Hint: genericStageExecutionHint, Severity: "error",
			}, stage.StageID, stage.Engine)
		}
		if stage.Passed {
			continue
		}
		for _, issue := range stage.Errors {
			builder.add(issue, stage.StageID, stage.Engine)
		}
	}

	aggregatedIssues := make(map[domain.ValidationIssue]struct{})
	for _, stage := range result.Stages {
		for _, issue := range stage.Errors {
			aggregatedIssues[issue] = struct{}{}
		}
	}
	for _, report := range result.Links {
		for _, issue := range report.Errors {
			aggregatedIssues[issue] = struct{}{}
		}
		if report.Optional {
			continue
		}
		if report.Passed {
			continue
		}
		for _, issue := range report.Errors {
			builder.add(issue, "", "")
		}
	}
	for _, issue := range result.Errors {
		if issue.Code == "STAGE_EXECUTION_ERROR" {
			if _, knownStage := stageByID[issue.StageID]; !knownStage {
				builder.add(domain.ValidationIssue{
					Code: "STAGE_EXECUTION_ERROR", Message: genericStageExecutionMessage,
					Hint: genericStageExecutionHint, Severity: "error",
				}, issue.StageID, issue.Engine)
			}
			continue
		}
		if _, knownStage := stageByID[issue.StageID]; issue.StageID != "" && knownStage {
			continue
		}
		if _, alreadyAggregated := aggregatedIssues[issue]; alreadyAggregated {
			continue
		}
		builder.add(issue, issue.StageID, issue.Engine)
	}
	return builder.explanation
}

func (builder *teacherExplanationBuilder) add(source domain.ValidationIssue, stageID, engine string) {
	issue, valid, changed := normalizeTeacherIssue(source, stageID, engine)
	if changed {
		builder.explanation.Truncated = true
	}
	if !valid {
		builder.explanation.Truncated = true
		return
	}
	if _, duplicate := builder.seen[issue]; duplicate {
		return
	}
	builder.seen[issue] = struct{}{}
	if len(builder.explanation.BlockingIssues) >= maximumTeacherBlockingIssues {
		builder.explanation.Truncated = true
		return
	}
	builder.explanation.BlockingIssues = append(builder.explanation.BlockingIssues, issue)
	if encoded, _ := json.Marshal(builder.explanation); len(encoded) > maximumTeacherExplanationBytes {
		builder.explanation.BlockingIssues = builder.explanation.BlockingIssues[:len(builder.explanation.BlockingIssues)-1]
		builder.explanation.Truncated = true
	}
}

func normalizeTeacherIssue(source domain.ValidationIssue, authoritativeStageID, authoritativeEngine string) (domain.TeacherValidationIssueV1, bool, bool) {
	severity := strings.ToLower(strings.TrimSpace(source.Severity))
	if severity == "warning" || severity == "info" || severity == "debug" {
		return domain.TeacherValidationIssueV1{}, false, false
	}
	code, codeChanged, codeValid := boundedTeacherText(source.Code, maximumTeacherCodeRunes)
	message, messageChanged, messageValid := boundedTeacherText(source.Message, maximumTeacherMessageRunes)
	if !codeValid || !messageValid {
		return domain.TeacherValidationIssueV1{}, false, codeChanged || messageChanged
	}
	if containsObviousAbsolutePath(message) {
		message = genericValidationIssueMessage
		messageChanged = true
	}
	issue := domain.TeacherValidationIssueV1{Code: code, Message: message}
	changed := codeChanged || messageChanged
	if source.Hint != "" {
		value, modified, valid := boundedTeacherText(source.Hint, maximumTeacherHintRunes)
		changed = changed || modified
		if valid && !containsObviousAbsolutePath(value) {
			issue.Hint = value
		} else {
			changed = true
		}
	}
	if authoritativeStageID == "" {
		authoritativeStageID = source.StageID
	}
	if authoritativeEngine == "" {
		authoritativeEngine = source.Engine
	}
	if authoritativeStageID != "" {
		value, modified, valid := boundedTeacherText(authoritativeStageID, maximumTeacherIdentityRunes)
		changed = changed || modified
		if valid {
			issue.StageID = value
		} else {
			changed = true
		}
	}
	if authoritativeEngine != "" {
		value, modified, valid := boundedTeacherText(authoritativeEngine, maximumTeacherIdentityRunes)
		changed = changed || modified
		if valid {
			issue.Engine = value
		} else {
			changed = true
		}
	}
	if severity == "error" || severity == "fatal" || severity == "critical" {
		issue.Severity = severity
	} else if severity != "" {
		changed = true
	}
	if source.File != "" {
		if validTeacherFile(source.File) {
			issue.File = source.File
		} else {
			changed = true
		}
	}
	if source.Line > 0 && source.Line <= maximumTeacherSourceLocation {
		issue.Line = source.Line
	} else if source.Line != 0 {
		changed = true
	}
	if source.Column > 0 && source.Column <= maximumTeacherSourceLocation {
		issue.Column = source.Column
	} else if source.Column != 0 {
		changed = true
	}
	return issue, true, changed
}

func boundedTeacherText(value string, maximumRunes int) (string, bool, bool) {
	if !utf8.ValidString(value) {
		return "", true, false
	}
	original := value
	value = strings.TrimSpace(value)
	var builder strings.Builder
	count := 0
	previousSpace := false
	for _, character := range value {
		if unicode.IsControl(character) || unicode.Is(unicode.Cf, character) {
			if count < maximumRunes && !previousSpace && builder.Len() > 0 {
				builder.WriteByte(' ')
				previousSpace = true
				count++
			}
			continue
		}
		if count >= maximumRunes {
			break
		}
		builder.WriteRune(character)
		previousSpace = unicode.IsSpace(character)
		count++
	}
	result := strings.TrimSpace(builder.String())
	changed := result != original || utf8.RuneCountInString(value) > maximumRunes
	return result, changed, result != ""
}

func validTeacherFile(value string) bool {
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > maximumTeacherFileRunes || hasUnsafeTeacherRune(value) ||
		strings.Contains(value, ":") || strings.HasPrefix(value, "~") {
		return false
	}
	return domain.ValidateWorkspaceFilePath(value) == nil
}

func containsObviousAbsolutePath(value string) bool {
	return obviousAbsolutePathPattern.MatchString(value)
}

func hasUnsafeTeacherRune(value string) bool {
	for _, character := range value {
		if unicode.IsControl(character) || unicode.Is(unicode.Cf, character) {
			return true
		}
	}
	return false
}
