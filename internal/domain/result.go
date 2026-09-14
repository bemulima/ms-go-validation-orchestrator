package domain

const TeacherValidationExplanationSchemaV1 = "teacher-validation-explanation.v1"

type ValidationResult struct {
	ContractKind       string                          `json:"contract_kind"`
	ContractVersion    int                             `json:"contract_version"`
	Legacy             bool                            `json:"legacy"`
	Passed             bool                            `json:"passed"`
	Stages             []StageReport                   `json:"stages"`
	Links              []LinkReport                    `json:"links,omitempty"`
	Errors             []ValidationIssue               `json:"errors,omitempty"`
	TeacherExplanation *TeacherValidationExplanationV1 `json:"teacher_explanation,omitempty"`
}

// TeacherValidationExplanationV1 is a presentation-only projection of a
// normalized owner result. It deliberately cannot represent raw engine output,
// evidence, executable commands, fixtures, or private rubric selectors.
type TeacherValidationExplanationV1 struct {
	Schema         string                     `json:"schema"`
	Passed         bool                       `json:"passed"`
	BlockingIssues []TeacherValidationIssueV1 `json:"blocking_issues"`
	Truncated      bool                       `json:"truncated"`
	SourceDigest   string                     `json:"source_digest"`
}

type TeacherValidationIssueV1 struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Hint     string `json:"hint,omitempty"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	StageID  string `json:"stage_id,omitempty"`
	Engine   string `json:"engine,omitempty"`
	Severity string `json:"severity,omitempty"`
}

type StageReport struct {
	StageID   string            `json:"stage_id"`
	Engine    string            `json:"engine"`
	Status    string            `json:"status"`
	Passed    bool              `json:"passed"`
	Optional  bool              `json:"optional"`
	Duration  int64             `json:"duration_ms"`
	Evidence  []ValidationPoint `json:"evidence,omitempty"`
	Errors    []ValidationIssue `json:"errors,omitempty"`
	Warnings  []ValidationIssue `json:"warnings,omitempty"`
	RawResult []byte            `json:"-"`
}

type LinkReport struct {
	LinkID   string            `json:"link_id"`
	Kind     string            `json:"kind"`
	Status   string            `json:"status"`
	Passed   bool              `json:"passed"`
	Optional bool              `json:"optional"`
	Errors   []ValidationIssue `json:"errors,omitempty"`
}

type ValidationIssue struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity,omitempty"`
	StageID  string `json:"stage_id,omitempty"`
	Engine   string `json:"engine,omitempty"`
	File     string `json:"file,omitempty"`
	Selector string `json:"selector,omitempty"`
	Route    string `json:"route,omitempty"`
	Symbol   string `json:"symbol,omitempty"`
	Property string `json:"property,omitempty"`
	Hint     string `json:"hint,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
}

type ValidationPoint struct {
	File     string `json:"file,omitempty"`
	Selector string `json:"selector,omitempty"`
	Route    string `json:"route,omitempty"`
	Symbol   string `json:"symbol,omitempty"`
	Property string `json:"property,omitempty"`
	Message  string `json:"message,omitempty"`
}

type StageExecutionResult struct {
	Passed    bool
	Evidence  []ValidationPoint
	Errors    []ValidationIssue
	Warnings  []ValidationIssue
	RawResult []byte
}
