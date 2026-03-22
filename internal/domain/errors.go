package domain

import "errors"

var (
	ErrInvalidRequest         = errors.New("invalid validation request")
	ErrInvalidContract        = errors.New("invalid validation contract")
	ErrUnsupportedEngine      = errors.New("unsupported validation engine")
	ErrStageExecutionFailed   = errors.New("stage execution failed")
	ErrDependencyCycle        = errors.New("validation stage dependency cycle")
	ErrInlineRulesUnsupported = errors.New("inline rules are unsupported by engine")
)
