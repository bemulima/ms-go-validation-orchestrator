package v2

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type boundedPracticeExecutor struct{ deadline chan time.Duration }

func (e *boundedPracticeExecutor) ValidatePractice(ctx context.Context, _ domain.PracticeValidationRequestV2) (domain.PracticeValidationResultV2, error) {
	if d, ok := ctx.Deadline(); ok {
		e.deadline <- time.Until(d)
	}
	time.Sleep(70 * time.Millisecond)
	return domain.PracticeValidationResultV2{Schema: domain.PracticeValidationResultSchemaV2, Outcome: domain.PracticeValidationPass}, nil
}
func TestPracticeValidationHasBoundedResponseBudgetBeyondSharedDeadline(t *testing.T) {
	executor := &boundedPracticeExecutor{deadline: make(chan time.Duration, 1)}
	handler := NewHandler(executor)
	server := httptest.NewUnstartedServer(http.HandlerFunc(handler.ValidatePractice))
	server.Config.WriteTimeout = 20 * time.Millisecond
	server.Start()
	defer server.Close()
	res, err := server.Client().Post(server.URL, "application/json", strings.NewReader(validPracticeRequestJSON()))
	if err != nil {
		t.Fatalf("official validation response lost after shared deadline: %v", err)
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, res.Body)
	if res.StatusCode != 200 {
		t.Fatalf("status=%d", res.StatusCode)
	}
	select {
	case deadline := <-executor.deadline:
		if deadline <= 0 || deadline > 2*time.Minute {
			t.Fatalf("unbounded execution context %v", deadline)
		}
	default:
		t.Fatal("missing bounded execution context")
	}
}
