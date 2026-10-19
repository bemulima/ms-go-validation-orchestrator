package engines

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/example/ms-validation-orchestrator-service/internal/domain"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func routeManifest(t *testing.T, name string) domain.GoTestManifest {
	t.Helper()
	b, e := os.ReadFile("testdata/go_runner_" + name + ".json")
	if e != nil {
		t.Fatal(e)
	}
	var m domain.GoTestManifest
	if json.Unmarshal(b, &m) != nil {
		t.Fatal("manifest")
	}
	return m
}
func routeBinding(m domain.GoTestManifest) domain.GoTestBinding {
	return domain.GoTestBinding{ProfileID: m.ProfileID, RunnerDigest: m.RunnerDigest, BundleDigest: m.BundleDigest, ExecutionProfileDigest: m.ExecutionProfileDigest, StarterDigest: m.StarterDigest}
}
func routeResult(m domain.GoTestManifest, b domain.GoTestBinding, verify bool) domain.GoTestResult {
	out := domain.GoTestResult{Schema: "go-test-result.v1", Manifest: m, Outcomes: map[string]domain.GoTestOutcome{}}
	names := []string{"solution"}
	if verify {
		names = []string{"reference", "starter", "negative"}
	}
	for _, name := range names {
		criteria := map[string]string{}
		for key := range m.CriterionTests {
			criteria[key] = "PASS"
		}
		status := "PASS"
		source := m.FixtureDigests[name]
		if !verify {
			source = domain.CanonicalDigestV2(b.Files)
			if b.ProfileID == "go-double-numbers.v1" {
				sum := sha256.Sum256([]byte(b.Files["solution.go"]))
				source = "sha256:" + hex.EncodeToString(sum[:])
			}
		} else if name != "reference" {
			status = "FAIL"
			key := "zeros"
			if m.ProfileID == "go-http-handler.v1" {
				key = "method"
			}
			criteria[key] = "FAIL"
		}
		out.Outcomes[name] = domain.GoTestOutcome{Status: status, Criteria: criteria, SourceDigest: source}
	}
	return out
}
func TestGoRunnerBindingRoutesNoFallback(t *testing.T) {
	old, newM, multi := routeManifest(t, "legacy"), routeManifest(t, "code"), routeManifest(t, "http")
	legacyCalls, newCalls := 0, 0
	mode := "valid"
	handler := func(legacy bool) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			token := "new-token"
			m := newM
			if legacy {
				token = "old-token"
				m = old
			}
			if r.Header.Get("Authorization") != "Bearer "+token {
				t.Error("service auth omitted")
				w.WriteHeader(401)
				return
			}
			if r.URL.Query().Get("profile_id") == multi.ProfileID && !legacy {
				m = multi
			}
			if mode == "unavailable" && legacy {
				w.WriteHeader(503)
				return
			}
			if strings.HasSuffix(r.URL.Path, "capability") {
				if mode == "wrong_build" && legacy {
					m = newM
				}
				json.NewEncoder(w).Encode(m)
				return
			}
			var b domain.GoTestBinding
			json.NewDecoder(r.Body).Decode(&b)
			if legacy {
				legacyCalls++
			} else {
				newCalls++
			}
			if b.ProfileID == multi.ProfileID && !legacy {
				m = multi
			}
			out := routeResult(m, b, strings.HasSuffix(r.URL.Path, "verify"))
			switch mode {
			case "forged":
				out.Manifest.RunnerDigest = "changed"
			case "source":
				o := out.Outcomes["solution"]
				o.SourceDigest = "changed"
				out.Outcomes["solution"] = o
			case "error":
				o := out.Outcomes["solution"]
				o.Status = "ERROR"
				out.Outcomes["solution"] = o
			}
			json.NewEncoder(w).Encode(out)
		}
	}
	legacy := httptest.NewServer(handler(true))
	defer legacy.Close()
	current := httptest.NewServer(handler(false))
	defer current.Close()
	routes := []GoRunnerRoute{{Binding: routeBinding(old), URL: legacy.URL, Token: "old-token"}, {Binding: routeBinding(newM), URL: current.URL, Token: "new-token"}, {Binding: routeBinding(multi), URL: current.URL, Token: "new-token"}}
	raw, _ := json.Marshal(routes)
	client, e := NewConfiguredGoTestsClient(current.URL, "new-token", string(raw))
	if e != nil {
		t.Fatal(e)
	}
	for _, m := range []domain.GoTestManifest{old, newM, multi} {
		b := routeBinding(m)
		b.Schema = "go-test-execution.v1"
		b.Files = map[string]string{"solution.go": m.Starter}
		if m.ProfileID == multi.ProfileID {
			b.Files = m.StarterFiles
		}
		cap, e := client.CapabilityForBinding(context.Background(), b)
		if e != nil || cap.RunnerDigest != m.RunnerDigest {
			t.Fatal("bound capability", e)
		}
		if _, e = client.Run(context.Background(), b, false); e != nil {
			t.Fatal("route execution", e)
		}
	}
	if legacyCalls != 1 || newCalls != 2 {
		t.Fatal("incorrect digest route", legacyCalls, newCalls)
	}
	base := routeBinding(old)
	base.Schema = "go-test-execution.v1"
	base.Files = map[string]string{"solution.go": old.Starter}
	for _, field := range []string{"runner", "profile", "bundle", "execution", "starter"} {
		b := base
		switch field {
		case "runner":
			b.RunnerDigest = "sha256:" + strings.Repeat("0", 64)
		case "profile":
			b.ProfileID = multi.ProfileID
		case "bundle":
			b.BundleDigest = multi.BundleDigest
		case "execution":
			b.ExecutionProfileDigest = multi.ExecutionProfileDigest
		case "starter":
			b.StarterDigest = multi.StarterDigest
		}
		if _, e = client.Run(context.Background(), b, false); e == nil {
			t.Fatal("mismatch accepted", field)
		}
	}
	if legacyCalls != 1 || newCalls != 2 {
		t.Fatal("unknown tuple reached execution")
	}
	for _, next := range []string{"unavailable", "wrong_build", "forged", "source", "error"} {
		mode = next
		before := newCalls
		if _, e = client.Run(context.Background(), base, false); e == nil {
			t.Fatal("invalid/unavailable target accepted", mode)
		}
		if newCalls != before {
			t.Fatal("fallback to new build", mode)
		}
	}
	// Missing legacy target is a hard error even though the default new endpoint is healthy.
	raw, _ = json.Marshal(routes[1:])
	missing, e := NewConfiguredGoTestsClient(current.URL, "new-token", string(raw))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = missing.Run(context.Background(), base, false); e == nil {
		t.Fatal("missing route fallback")
	}
}
func TestGoRunnerRoutesRejectInvalidDeploymentConfig(t *testing.T) {
	m := routeManifest(t, "legacy")
	good := GoRunnerRoute{Binding: routeBinding(m), URL: "http://127.0.0.1:1234", Token: "test"}
	for _, kind := range []string{"duplicate", "url", "token", "digest", "files", "unknown", "empty"} {
		route := good
		switch kind {
		case "url":
			route.URL = "file:///tmp/client-selected"
		case "token":
			route.Token = ""
		case "digest":
			route.Binding.BundleDigest = "bad"
		case "files":
			route.Binding.Files = map[string]string{"url": "bad"}
		}
		routes := []GoRunnerRoute{route}
		if kind == "duplicate" {
			routes = append(routes, route)
		}
		raw, _ := json.Marshal(routes)
		if kind == "unknown" {
			raw = []byte(`[{"arbitrary_url":"http://evil"}]`)
		}
		if kind == "empty" {
			raw = []byte(`[]`)
		}
		if _, e := NewConfiguredGoTestsClient(good.URL, good.Token, string(raw)); e == nil {
			t.Fatal("bad config accepted", kind)
		}
	}
}
func TestGoRunnerDeploymentExampleIsValidConfig(t *testing.T) {
	raw, e := os.ReadFile("../../../config/go-runner-routes.example.json")
	if e != nil {
		t.Fatal(e)
	}
	client, e := NewConfiguredGoTestsClient("http://code-validator-current:8080", "placeholder", string(raw))
	if e != nil || len(client.routes) != 3 {
		t.Fatal("invalid deployment example", e)
	}
}
