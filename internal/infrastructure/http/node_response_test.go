package engines

import (
	"errors"
	"testing"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

func TestParseNodeValidationResponseRejectsMalformedProtocol(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{name: "empty object", body: `{}`},
		{name: "non-boolean outcome", body: `{"ok":1}`},
		{name: "malformed summary", body: `{"ok":false,"summary":[]}`},
		{name: "malformed summary flag", body: `{"ok":false,"summary":{"staticOk":"false"}}`},
		{name: "malformed errors", body: `{"ok":false,"errors":{}}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseNodeValidationResponse([]byte(test.body), domain.ValidationStage{ID: "test", Engine: "node.express"})
			if !errors.Is(err, domain.ErrValidatorProtocol) {
				t.Fatalf("expected validator protocol error, got %v", err)
			}
		})
	}
}
