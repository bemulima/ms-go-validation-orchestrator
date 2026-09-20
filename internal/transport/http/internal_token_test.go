package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireInternalTokenFailsClosed(t *testing.T) {
	next := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	})
	handler := requireInternalToken("secret", next)

	tests := []struct {
		name   string
		token  string
		status int
	}{
		{name: "missing", status: http.StatusUnauthorized},
		{name: "wrong", token: "wrong", status: http.StatusUnauthorized},
		{name: "correct", token: "secret", status: http.StatusNoContent},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/v1/capabilities", nil)
			if test.token != "" {
				request.Header.Set("X-Internal-Token", test.token)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("expected status %d, got %d", test.status, response.Code)
			}
		})
	}
}

func TestRequireInternalTokenRejectsEmptyServerSecret(t *testing.T) {
	handler := requireInternalToken("", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("request reached protected handler")
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/capabilities", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", response.Code)
	}
}
