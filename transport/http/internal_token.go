package http

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"net/http"
)

func requireInternalToken(expected string, next http.Handler) http.Handler {
	expectedDigest := sha256.Sum256([]byte(expected))
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		providedDigest := sha256.Sum256([]byte(request.Header.Get("X-Internal-Token")))
		if expected == "" || subtle.ConstantTimeCompare(providedDigest[:], expectedDigest[:]) != 1 {
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(responseWriter).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(responseWriter, request)
	})
}
