package engines

import (
	"context"
	"errors"
	"github.com/example/ms-validation-orchestrator-service/internal/domain"
	"io"
	"net/http"
	"strings"
	"testing"
)

type failingGoBody struct{ cause error }

func (b failingGoBody) Read([]byte) (int, error) { return 0, b.cause }
func (b failingGoBody) Close() error             { return nil }

type goProviderRoundTrip func(*http.Request) (*http.Response, error)

func (f goProviderRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestGoProviderErrorsPreserveTypedCause(t *testing.T) {
	for _, tc := range []struct {
		name     string
		code     int
		body     string
		err      error
		input    any
		provider bool
		bodyErr  error
	}{
		{name: "unavailable", err: errors.New("connection refused"), provider: true},
		{name: "timeout", err: context.DeadlineExceeded, provider: true},
		{name: "body_timeout", code: 200, bodyErr: context.DeadlineExceeded, provider: true},
		{name: "truncated_body", code: 200, bodyErr: io.ErrUnexpectedEOF, provider: true},
		{name: "service", code: 503, body: "private provider detail", provider: true},
		{name: "unauthorized", code: 403, provider: true},
		{name: "protocol", code: 200, body: "not JSON private provider detail", provider: true},
		{name: "authored_invalid", code: 422, input: map[string]string{"profile_id": "go-double-numbers-cases.v1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := NewGoTestsClient("http://provider.invalid", "fixture-token")
			client.HTTP = &http.Client{Transport: goProviderRoundTrip(func(*http.Request) (*http.Response, error) {
				if tc.err != nil {
					return nil, tc.err
				}
				if tc.bodyErr != nil {
					return &http.Response{StatusCode: tc.code, Body: failingGoBody{tc.bodyErr}}, nil
				}
				return &http.Response{StatusCode: tc.code, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
			})}
			var out domain.GoTestManifest
			err := client.call(context.Background(), "/internal/go-tests/prepare", tc.input, &out)
			if err == nil {
				t.Fatal("failure accepted")
			}
			var provider domain.GoTestProviderError
			if errors.As(err, &provider) != tc.provider {
				t.Fatalf("wrong error ownership: %T", err)
			}
			if (tc.err == context.DeadlineExceeded || tc.bodyErr == context.DeadlineExceeded) && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatal("timeout cause lost")
			}
			if tc.provider && strings.Contains(err.Error(), "private") {
				t.Fatal("provider detail exposed")
			}
			if !tc.provider && !errors.Is(err, domain.ErrInvalidContract) {
				t.Fatal("authored rejection lost")
			}
		})
	}
}
