package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
	apiV2 "github.com/example/ms-validation-orchestrator-service/internal/transport/http/api/v2"
)

type practiceValidationRouterStub struct{}

func (practiceValidationRouterStub) ValidatePractice(context.Context, domain.PracticeValidationRequestV2) (domain.PracticeValidationResultV2, error) {
	return domain.PracticeValidationResultV2{}, nil
}

func TestPracticeValidationV2RouteRequiresInternalToken(t *testing.T) {
	t.Parallel()
	router := NewRouter(Dependencies{
		PracticeValidationHandler: apiV2.NewHandler(practiceValidationRouterStub{}),
		InternalToken:             "validation-token",
	})
	for _, test := range []struct {
		name       string
		token      string
		wantStatus int
	}{
		{name: "missing", wantStatus: http.StatusUnauthorized},
		{name: "valid token reaches handler", token: "validation-token", wantStatus: http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := httptest.NewRequest(http.MethodPost, "/api/v2/practice-validations", strings.NewReader(`{}`))
			if test.token != "" {
				request.Header.Set("X-Internal-Token", test.token)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status=%d body=%s, want %d", response.Code, response.Body.String(), test.wantStatus)
			}
		})
	}
}
