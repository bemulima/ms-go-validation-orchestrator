package domain

type ValidationResult struct {
	ContractKind    string            `json:"contract_kind"`
	ContractVersion int               `json:"contract_version"`
	Legacy          bool              `json:"legacy"`
	Passed          bool              `json:"passed"`
	Stages          []StageReport     `json:"stages"`
	Links           []LinkReport      `json:"links,omitempty"`
	Errors          []ValidationIssue `json:"errors,omitempty"`
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
