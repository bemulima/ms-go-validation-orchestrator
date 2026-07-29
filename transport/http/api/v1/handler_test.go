package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type stubValidationExecutor struct {
	engineIDs []string
}

func (stubValidationExecutor) Execute(
	context.Context,
	domain.ValidationRequest,
) (domain.ValidationResult, error) {
	return domain.ValidationResult{}, nil
}

func (executor stubValidationExecutor) ConfiguredEngineIDs() []string {
	return executor.engineIDs
}

type stubLogger struct{}

func (stubLogger) Info(string, map[string]string)  {}
func (stubLogger) Error(string, map[string]string) {}

func TestListEngines(t *testing.T) {
	t.Parallel()

	handler := NewHandler(stubValidationExecutor{
		engineIDs: []string{"go.core", "java.compile", "linux.runtime"},
	}, stubLogger{})
	router := http.NewServeMux()
	RegisterRoutes(router, handler)

	request := httptest.NewRequest(http.MethodGet, "/engines", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	var payload struct {
		Engines []string `json:"engines"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	want := []string{"go.core", "java.compile", "linux.runtime"}
	if !reflect.DeepEqual(payload.Engines, want) {
		t.Fatalf("expected engines %v, got %v", want, payload.Engines)
	}
}

func TestListEnginesRejectsOtherMethods(t *testing.T) {
	t.Parallel()

	handler := NewHandler(stubValidationExecutor{}, stubLogger{})
	router := http.NewServeMux()
	RegisterRoutes(router, handler)

	request := httptest.NewRequest(http.MethodPost, "/engines", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", response.Code)
	}
}
