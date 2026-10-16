package http

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type ExecutableVerifier interface {
	Verify(context.Context, domain.ExecutableVerificationRequest) (domain.ExecutableVerificationReceipt, error)
	Validate(domain.VerificationReceiptValidation) bool
}

func executableVerificationHandler(v ExecutableVerifier, token string, validate bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if token == "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Internal-Token")), []byte(token)) != 1 {
			http.Error(w, "unauthorized", 401)
			return
		}
		if r.Method != "POST" {
			http.Error(w, "method_not_allowed", 405)
			return
		}
		d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		d.DisallowUnknownFields()
		var input domain.ExecutableVerificationRequest
		var receipt domain.VerificationReceiptValidation
		var target any = &input
		if validate {
			target = &receipt
		}
		if d.Decode(target) != nil {
			http.Error(w, "invalid_request", 400)
			return
		}
		var extra any
		if d.Decode(&extra) != io.EOF {
			http.Error(w, "invalid_request", 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if validate {
			_ = json.NewEncoder(w).Encode(map[string]bool{"valid": v.Validate(receipt)})
			return
		}
		out, err := v.Verify(r.Context(), input)
		if err != nil {
			http.Error(w, "verification_rejected", 422)
			return
		}
		_ = json.NewEncoder(w).Encode(out)
	})
}
