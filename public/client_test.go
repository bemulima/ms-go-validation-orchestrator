package public

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

func TestNewHTTPValidationClientProvidesSafeDefaultTimeout(t *testing.T) {
	client := NewHTTPValidationClient("http://validation", nil)
	if client.client.Timeout != 30*time.Second {
		t.Fatalf("expected 30s default timeout, got %s", client.client.Timeout)
	}
}

func TestAuthenticatedHTTPValidationClientSendsInternalToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("X-Internal-Token") != "secret" {
			t.Fatalf("unexpected internal token %q", request.Header.Get("X-Internal-Token"))
		}
		_ = json.NewEncoder(writer).Encode(domain.ValidationResult{Passed: true})
	}))
	defer server.Close()

	client := NewAuthenticatedHTTPValidationClient(server.URL, "secret", server.Client())
	result, err := client.Validate(context.Background(), domain.ValidationRequest{
		CodeStructure: json.RawMessage(`{"rules":{}}`),
	})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !result.Passed {
		t.Fatal("expected passing result")
	}
}
