package engines

import (
	"testing"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

func TestParseCommonValidationResponseAcceptsOKPayload(t *testing.T) {
	t.Parallel()

	result, err := parseCommonValidationResponse([]byte(`{"ok":true,"errors":[]}`), domain.ValidationStage{
		ID:     "php",
		Engine: "php.core",
	})
	if err != nil {
		t.Fatalf("parse response: %v", err)
	}

	if !result.Passed {
		t.Fatalf("expected ok payload to pass")
	}
}

func TestParseCommonValidationResponseUsesDetailAsSymbol(t *testing.T) {
	t.Parallel()

	result, err := parseCommonValidationResponse([]byte(`{
		"ok": false,
		"errors": [
			{
				"code": "CLASS_MISSING",
				"message": "Class UserService is required.",
				"detail": "UserService::create"
			}
		]
	}`), domain.ValidationStage{
		ID:     "php",
		Engine: "php.core",
	})
	if err != nil {
		t.Fatalf("parse response: %v", err)
	}

	if len(result.Errors) != 1 {
		t.Fatalf("expected one error, got %d", len(result.Errors))
	}

	if result.Errors[0].Symbol != "UserService::create" {
		t.Fatalf("expected detail to propagate as symbol, got %q", result.Errors[0].Symbol)
	}
}

func TestParseCommonValidationResponseSupportsWarningsAndEvidence(t *testing.T) {
	t.Parallel()

	result, err := parseCommonValidationResponse([]byte(`{
		"ok": false,
		"errors": [],
		"warnings": [
			{
				"code": "OPTIONAL_WARNING",
				"message": "Optional route is not implemented",
				"severity": "warning",
				"file": "src/main.ts"
			}
		],
		"evidence": [
			{
				"file": "src/main.ts",
				"message": "checked entrypoint"
			}
		]
	}`), domain.ValidationStage{
		ID:     "node",
		Engine: "nextjs.app",
	})
	if err != nil {
		t.Fatalf("parse response: %v", err)
	}

	if len(result.Warnings) != 1 {
		t.Fatalf("expected one warning, got %d", len(result.Warnings))
	}

	if result.Warnings[0].Severity != "warning" {
		t.Fatalf("expected warning severity, got %q", result.Warnings[0].Severity)
	}

	if len(result.Evidence) != 1 {
		t.Fatalf("expected one evidence item, got %d", len(result.Evidence))
	}

	if result.Evidence[0].File != "src/main.ts" {
		t.Fatalf("expected evidence file to be propagated, got %q", result.Evidence[0].File)
	}
}
