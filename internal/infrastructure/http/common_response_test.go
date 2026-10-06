package engines

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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

func TestParseCommonValidationResponseRejectsMalformedProtocol(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{name: "empty object", body: `{}`},
		{name: "no outcome", body: `{"warnings":[]}`},
		{name: "non-boolean outcome", body: `{"ok":"yes"}`},
		{name: "null ok", body: `{"ok":null}`},
		{name: "null isValid", body: `{"isValid":null}`},
		{name: "null valid", body: `{"valid":null}`},
		{name: "numeric ok", body: `{"ok":1}`},
		{name: "array isValid", body: `{"isValid":[]}`},
		{name: "object valid", body: `{"valid":{}}`},
		{name: "malformed errors", body: `{"ok":false,"errors":{}}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseCommonValidationResponse([]byte(test.body), domain.ValidationStage{ID: "test", Engine: "php.core"})
			if !errors.Is(err, domain.ErrValidatorProtocol) {
				t.Fatalf("expected validator protocol error, got %v", err)
			}
		})
	}
}

// These are synthetic protocol fault envelopes, not observed live engine verdicts.
func TestParseCommonValidationResponseRejectsContradictorySyntheticOutcomes(t *testing.T) {
	t.Parallel()
	fields := []string{"ok", "isValid", "valid"}
	for present := 1; present < 8; present++ {
		for truth := 0; truth < 8; truth++ {
			if truth&^present != 0 || truth == 0 || truth == present {
				continue
			}
			t.Run(fmt.Sprintf("present=%03b/true=%03b", present, truth), func(t *testing.T) {
				t.Parallel()
				envelope := map[string]any{"errors": []any{}}
				for index, field := range fields {
					if present&(1<<index) != 0 {
						envelope[field] = truth&(1<<index) != 0
					}
				}
				body, err := json.Marshal(envelope)
				if err != nil {
					t.Fatal(err)
				}
				result, err := parseCommonValidationResponse(body, domain.ValidationStage{ID: "synthetic", Engine: "synthetic.common"})
				if !errors.Is(err, domain.ErrValidatorProtocol) || result.Passed || len(result.RawResult) != 0 {
					t.Fatalf("conflicting flags were normalized as a verdict: error=%v passed=%t", err, result.Passed)
				}
			})
		}
	}
}

func TestParseCommonValidationResponseRejectsSyntheticTrueWithErrors(t *testing.T) {
	t.Parallel()
	for _, body := range []string{
		`{"ok":true,"errors":[{"code":"SYNTHETIC_FAILURE","message":"Synthetic issue"}]}`,
		`{"isValid":true,"errors":[{"code":"SYNTHETIC_FAILURE","message":"Synthetic issue"}]}`,
		`{"valid":true,"errors":[{"code":"SYNTHETIC_FAILURE","message":"Synthetic issue"}]}`,
		`{"ok":true,"isValid":true,"valid":true,"errors":[{"severity":"warning","message":"Still in the errors bucket"}]}`,
	} {
		t.Run(body, func(t *testing.T) {
			t.Parallel()
			result, err := parseCommonValidationResponse([]byte(body), domain.ValidationStage{ID: "synthetic", Engine: "synthetic.common"})
			if !errors.Is(err, domain.ErrValidatorProtocol) || result.Passed || len(result.RawResult) != 0 {
				t.Fatalf("true with error issues was normalized as a verdict: error=%v passed=%t", err, result.Passed)
			}
		})
	}
}

func TestParseCommonValidationResponsePreservesConsistentOutcomesAndProjection(t *testing.T) {
	t.Parallel()
	fields := []string{"ok", "isValid", "valid"}
	for present := 1; present < 8; present++ {
		for _, passed := range []bool{false, true} {
			t.Run(fmt.Sprintf("present=%03b/passed=%t", present, passed), func(t *testing.T) {
				t.Parallel()
				envelope := map[string]any{
					"errors":   []any{},
					"warnings": []map[string]string{{"code": "OPTIONAL_WARNING", "message": "Optional hint", "file": "index.html"}},
					"evidence": []map[string]string{{"message": "Checked target", "file": "index.html"}},
				}
				for index, field := range fields {
					if present&(1<<index) != 0 {
						envelope[field] = passed
					}
				}
				body, err := json.Marshal(envelope)
				if err != nil {
					t.Fatal(err)
				}
				stage := domain.ValidationStage{ID: "synthetic", Engine: "synthetic.common"}
				result, err := parseCommonValidationResponse(body, stage)
				if err != nil || result.Passed != passed || len(result.Errors) != 0 || !bytes.Equal(result.RawResult, body) {
					t.Fatalf("consistent outcome changed: %v", err)
				}
				if len(result.Warnings) != 1 || result.Warnings[0].Code != "OPTIONAL_WARNING" || result.Warnings[0].StageID != stage.ID || result.Warnings[0].Engine != stage.Engine || result.Warnings[0].Severity != "warning" || len(result.Evidence) != 1 || result.Evidence[0].File != "index.html" {
					t.Fatal("existing warning/evidence projection changed")
				}
			})
		}
	}
}

func TestParseCommonValidationResponsePreservesFalseAndErrorsOnlyCompatibility(t *testing.T) {
	t.Parallel()
	for _, body := range []string{
		`{"ok":false,"isValid":false,"valid":false,"errors":[{"code":"CHECK_FAILED","message":"Expected value missing","path":"index.html","detail":"App"}]}`,
		`{"errors":[{"code":"CHECK_FAILED","message":"Expected value missing","path":"index.html","detail":"App"}]}`,
	} {
		t.Run(body, func(t *testing.T) {
			t.Parallel()
			result, err := parseCommonValidationResponse([]byte(body), domain.ValidationStage{ID: "check", Engine: "synthetic.common"})
			if err != nil || result.Passed || len(result.Errors) != 1 || result.Errors[0].Code != "CHECK_FAILED" || result.Errors[0].File != "index.html" || result.Errors[0].Symbol != "App" || result.Errors[0].Severity != "error" {
				t.Fatalf("existing semantic failure compatibility changed: %v", err)
			}
		})
	}
}
