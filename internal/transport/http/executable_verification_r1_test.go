package http

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/example/ms-validation-orchestrator-service/internal/domain"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type deadlineWriter struct {
	*httptest.ResponseRecorder
	deadline time.Time
	fail     bool
}

func (w *deadlineWriter) SetWriteDeadline(d time.Time) error {
	w.deadline = d
	if w.fail {
		return errors.New("unsupported")
	}
	return nil
}

type deadlineVerifier struct {
	calls    int
	deadline bool
	ctx      context.Context
}

func (v *deadlineVerifier) Verify(ctx context.Context, _ domain.ExecutableVerificationRequest) (domain.ExecutableVerificationReceipt, error) {
	v.calls++
	v.ctx = ctx
	d, ok := ctx.Deadline()
	v.deadline = ok && time.Until(d) > 299*time.Second
	return domain.ExecutableVerificationReceipt{}, nil
}
func (v *deadlineVerifier) Prepare(ctx context.Context, _ json.RawMessage) (domain.GoTestManifest, error) {
	v.calls++
	v.ctx = ctx
	_, v.deadline = ctx.Deadline()
	return domain.GoTestManifest{}, nil
}
func (v *deadlineVerifier) Validate(domain.VerificationReceiptValidation) bool {
	v.calls++
	return true
}
func TestExecutableLimitsAndScopedDeadline(t *testing.T) {
	for _, path := range []string{"/api/v2/contracts/verify", "/api/v2/contracts/prepare", "/internal/verification-receipts/validate"} {
		for _, size := range []int{(1 << 20) + 32, (2 << 20) + 1} {
			v := &deadlineVerifier{}
			body := `{}`
			if strings.HasSuffix(path, "prepare") {
				body = `{"schema":"go-test-prepare.v1","binding":{"profile_id":"go-double-numbers-cases.v1"}}`
			}
			body += strings.Repeat(" ", size-len(body))
			r := httptest.NewRequest("POST", path, strings.NewReader(body))
			r.Header.Set("X-Internal-Token", "unit")
			w := &deadlineWriter{ResponseRecorder: httptest.NewRecorder()}
			validate := strings.Contains(path, "receipts")
			executableVerificationHandler(v, "unit", validate).ServeHTTP(w, r)
			if size > 2<<20 {
				if w.Code != 400 || v.calls != 0 {
					t.Fatal("oversize accepted")
				}
			} else {
				if w.Code != 200 || v.calls != 1 {
					t.Fatalf("valid 2MiB request rejected: %d", w.Code)
				}
				if validate {
					if !w.deadline.IsZero() {
						t.Fatal("receipt deadline widened")
					}
				} else {
					if !v.deadline || w.deadline.IsZero() {
						t.Fatal("missing scoped deadline")
					}
					select {
					case <-v.ctx.Done():
					default:
						t.Fatal("handler context not canceled")
					}
				}
			}
		}
	}
}
func TestExecutableUnsupportedDeadlineAndStrictJSONFailClosed(t *testing.T) {
	for _, body := range []string{`{}`, `{} {}`, `{"unknown":1}`} {
		v := &deadlineVerifier{}
		r := httptest.NewRequest("POST", "/api/v2/contracts/verify", strings.NewReader(body))
		r.Header.Set("X-Internal-Token", "unit")
		w := &deadlineWriter{ResponseRecorder: httptest.NewRecorder(), fail: true}
		executableVerificationHandler(v, "unit", false).ServeHTTP(w, r)
		if v.calls != 0 || w.Code == 200 {
			t.Fatal("invalid request reached verifier")
		}
	}
}
