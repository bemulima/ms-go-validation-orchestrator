//go:build integration

package app

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/ms-validation-orchestrator-service/config"
)

// This is production App composition and registered HTTP, with a declared
// test-only transport-fault gate. The HTML endpoint must be the actual owner
// production DI/parser/rules/router process; it is never replaced by a fake.
func TestT12ServeFeedbackValidationProvider(t *testing.T) {
	if os.Getenv("T12_SERVE_ORCHESTRATOR") != "true" {
		t.Skip("owned provider gate absent")
	}
	token, sandboxToken := os.Getenv("T12_ORCHESTRATOR_TOKEN"), os.Getenv("T12_SANDBOX_TOKEN")
	out, stop, proofPath := os.Getenv("T12_ORCHESTRATOR_READY_FILE"), os.Getenv("T12_ORCHESTRATOR_STOP_FILE"), os.Getenv("T12_ORCHESTRATOR_RESULT_FILE")
	htmlURL, sandboxURL := os.Getenv("T12_HTML_PROVIDER_URL"), os.Getenv("T12_SANDBOX_URL")
	if token == "" || sandboxToken == "" || out == "" || stop == "" || proofPath == "" || htmlURL == "" || sandboxURL == "" {
		t.Fatal("missing owned actual-provider configuration")
	}
	if _, err := os.Stat(stop); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("stop marker must be absent")
	}
	// Owner direct positive/negative is prerequisites evidence for the exact
	// inline rules used by Runtime; authored message deliberately carries a
	// private canary which must never reach Runtime public/Teacher projections.
	rules := json.RawMessage(`{"version":2,"body":{"roots":[{"tag":{"value":"main","errorMessage":"T12_PRIVATE_VALIDATOR_CANARY private evaluation reference"}}]}}`)
	direct := map[string]any{}
	client := &http.Client{Timeout: 10 * time.Second}
	for _, kind := range []string{"positive", "negative"} {
		tag := "main"
		if kind == "negative" {
			tag = "div"
		}
		raw, _ := json.Marshal(map[string]any{"code": "<!DOCTYPE html><html lang=\"en\"><body><" + tag + ">Public exercise</" + tag + "></body></html>", "taskId": "t12-public-inline-rule", "rules": rules})
		response, err := client.Post(strings.TrimRight(htmlURL, "/")+"/validate", "application/json", bytes.NewReader(raw))
		if err != nil {
			t.Fatal("actual HTML owner unavailable")
		}
		var result struct {
			Valid  *bool `json:"isValid"`
			Errors []struct {
				Code string `json:"code"`
			} `json:"errors"`
		}
		err = json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result)
		response.Body.Close()
		if err != nil || response.StatusCode != 200 || result.Valid == nil || *result.Valid != (kind == "positive") {
			t.Fatalf("actual HTML %s prerequisite failed status=%d", kind, response.StatusCode)
		}
		codes := []string{}
		for _, issue := range result.Errors {
			codes = append(codes, issue.Code)
		}
		if kind == "negative" && (len(codes) != 1 || codes[0] != "TAG_MISMATCH") {
			t.Fatal("actual owner semantic criterion absent")
		}
		direct[kind] = map[string]any{"status": response.StatusCode, "is_valid": *result.Valid, "codes": codes}
	}
	cfg := config.Config{InternalAPIToken: token, SandboxServiceBaseURL: sandboxURL, SandboxServiceInternalToken: sandboxToken, Engines: config.EngineEndpoints{HTML: htmlURL}, VerificationWorkspacesDir: t.TempDir()}
	production := New(cfg).Server().Handler
	var unavailable atomic.Bool
	var count, faultCount atomic.Int64
	var mu sync.Mutex
	observations := []map[string]any{}
	export := func() {
		mu.Lock()
		defer mu.Unlock()
		raw, err := json.MarshalIndent(map[string]any{"actual_html_owner": direct, "validation_http_requests": count.Load(), "injected_transport_failures": faultCount.Load(), "responses": observations, "classification": "production Orchestrator App registered HTTP + actual HTML owner; explicit local HTTP reply-drop fault; synthetic public binding; no full learner E2E"}, "", "  ")
		if err != nil || os.WriteFile(proofPath, append(raw, '\n'), 0600) != nil {
			t.Error("safe provider proof export failed")
		}
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/__t12/outage/") {
			supplied := r.Header.Get("X-Internal-Token")
			if r.Method != http.MethodPost || supplied == "" || subtle.ConstantTimeCompare([]byte(supplied), []byte(token)) != 1 {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			switch r.URL.Path {
			case "/__t12/outage/on":
				unavailable.Store(true)
			case "/__t12/outage/off":
				unavailable.Store(false)
			default:
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.URL.Path != "/api/v2/practice-validations" {
			production.ServeHTTP(w, r)
			return
		}
		count.Add(1)
		if unavailable.Load() {
			faultCount.Add(1)
			export()
			hijacker, ok := w.(http.Hijacker)
			if !ok {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			connection, _, err := hijacker.Hijack()
			if err == nil {
				connection.Close()
			}
			return
		}
		// A recorder forwards the real App response byte-for-byte. Only public
		// correlation/status/code fields, never raw messages/contracts, are saved.
		recorded := httptest.NewRecorder()
		production.ServeHTTP(recorded, r)
		var body map[string]json.RawMessage
		json.Unmarshal(recorded.Body.Bytes(), &body)
		safe := map[string]any{"http_status": recorded.Code}
		for _, key := range []string{"schema", "task_instance_id", "submission_id", "validation_run_id", "revision_digest", "validation_contract_digest", "snapshot", "outcome", "error_classification"} {
			if value, ok := body[key]; ok {
				safe[key] = value
			}
		}
		var stages []struct {
			ID      string `json:"stage_id"`
			Outcome string `json:"outcome"`
			Issues  []struct {
				Code string `json:"code"`
			} `json:"issues"`
		}
		json.Unmarshal(body["stages"], &stages)
		safe["stage_count"] = len(stages)
		codes := []string{}
		for _, stage := range stages {
			for _, issue := range stage.Issues {
				codes = append(codes, issue.Code)
			}
		}
		safe["issue_codes"] = codes
		mu.Lock()
		observations = append(observations, safe)
		mu.Unlock()
		export()
		for key, values := range recorded.Header() {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(recorded.Code)
		w.Write(recorded.Body.Bytes())
	})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal("owned listener unavailable")
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	defer server.Close()
	errCh := make(chan error, 1)
	go func() { errCh <- server.Serve(listener) }()
	raw, _ := json.Marshal(map[string]any{"url": "http://" + listener.Addr().String(), "classification": "actual registered production App HTTP+actual HTML; explicit test-only local reply-drop gate"})
	if os.WriteFile(out, append(raw, '\n'), 0600) != nil {
		t.Fatal("owned ready export failed")
	}
	export()
	t.Log("T12 actual Orchestrator and HTML provider prerequisites ready")
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.NewTimer(30 * time.Minute)
	defer deadline.Stop()
	for {
		select {
		case err := <-errCh:
			if !errors.Is(err, http.ErrServerClosed) {
				t.Fatal("owned listener failed")
			}
			return
		case <-deadline.C:
			t.Fatal("owned provider deadline exceeded")
		case <-ticker.C:
			if _, err := os.Stat(stop); err == nil {
				export()
				return
			} else if !errors.Is(err, os.ErrNotExist) {
				t.Fatal("stop marker read failed")
			}
		}
	}
}
