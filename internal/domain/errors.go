package domain

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidRequest         = errors.New("invalid validation request")
	ErrInvalidContract        = errors.New("invalid validation contract")
	ErrUnsupportedEngine      = errors.New("unsupported validation engine")
	ErrStageExecutionFailed   = errors.New("stage execution failed")
	ErrDependencyCycle        = errors.New("validation stage dependency cycle")
	ErrInlineRulesUnsupported = errors.New("inline rules are unsupported by engine")
	ErrValidatorProtocol      = errors.New("validator protocol error")
	ErrVerificationWorkspace  = errors.New("verification workspace unavailable or unsupported")
)

type ValidatorHTTPStatusError struct {
	StatusCode int
}

func (failure ValidatorHTTPStatusError) Error() string {
	return fmt.Sprintf("unexpected status %d", failure.StatusCode)
}
