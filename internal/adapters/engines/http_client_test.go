package engines

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPClientReturnsValidationBodyForUnprocessableEntity(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = writer.Write([]byte(`{"ok":false,"isValid":false,"errors":[{"code":"VALIDATION_FAILED"}]}`))
	}))
	defer server.Close()

	body, err := NewHTTPClient(time.Second).PostJSON(context.Background(), server.URL, map[string]string{"code": "TODO"})
	if err != nil {
		t.Fatalf("post validation response: %v", err)
	}
	if !strings.Contains(string(body), "VALIDATION_FAILED") {
		t.Fatalf("expected validation response body, got %s", body)
	}
}

func TestHTTPClientRejectsBadRequest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(`{"error":"invalid request"}`))
	}))
	defer server.Close()

	_, err := NewHTTPClient(time.Second).PostJSON(context.Background(), server.URL, map[string]string{"code": "TODO"})
	if err == nil || !strings.Contains(err.Error(), "unexpected status 400") {
		t.Fatalf("expected HTTP 400 transport error, got %v", err)
	}
}
