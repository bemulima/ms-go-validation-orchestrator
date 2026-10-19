package engines

import (
	"context"
	"encoding/json"
	"github.com/example/ms-validation-orchestrator-service/internal/domain"
	"os"
	"testing"
)

// This opt-in test requires separately authorized, isolated real owner binaries/OCI.
// Its private config (including trusted owner reference source) must never be committed.
func TestGoRunnerLivePinnedBindings(t *testing.T) {
	path := os.Getenv("LC06_RUNNER_LIVE_CONFIG")
	if path == "" {
		t.Skip("isolated real Code Validator/OCI config not authorized or supplied")
	}
	var cfg struct {
		DefaultURL, DefaultToken string
		Routes                   []GoRunnerRoute
		Cases                    []struct {
			Name                             string
			Manifest                         domain.GoTestManifest
			Reference, Starter, CompileError map[string]string
		}
	}
	raw, e := os.ReadFile(path)
	if e != nil || json.Unmarshal(raw, &cfg) != nil {
		t.Fatal("invalid private live config")
	}
	routes, _ := json.Marshal(cfg.Routes)
	for _, fixture := range cfg.Cases {
		t.Run(fixture.Name, func(t *testing.T) {
			t.Parallel()
			client, e := NewConfiguredGoTestsClient(cfg.DefaultURL, cfg.DefaultToken, string(routes))
			if e != nil {
				t.Fatal(e)
			}
			m := fixture.Manifest
			b := routeBinding(m)
			b.Schema = "go-test-execution.v1"
			checks, _ := json.Marshal(map[string]any{"profile_id": m.ProfileID, "private_binding": b})
			for _, test := range []struct {
				name  string
				files map[string]string
				want  string
			}{{"reference", fixture.Reference, "PASS"}, {"starter", fixture.Starter, "FAIL"}, {"compile_error", fixture.CompileError, "ERROR"}} {
				input := domain.EngineValidationInput{Mode: "final", Stage: domain.ValidationStage{Checks: checks}}
				for p, content := range test.files {
					input.Workspace.Files = append(input.Workspace.Files, domain.WorkspaceFile{Path: p, Content: content})
				}
				result, e := client.Validate(context.Background(), input)
				got := "FAIL"
				if e != nil {
					got = "ERROR"
				} else if result.Passed {
					got = "PASS"
				}
				if got != test.want {
					t.Fatalf("%s got %s want %s", test.name, got, test.want)
				}
				t.Logf("real pinned runner %s: %s => %s", fixture.Name, test.name, got)
			}
		})
	}
}
func TestGoRunnerLiveUnavailablePinnedTarget(t *testing.T) {
	if os.Getenv("LC06_RUNNER_EXPECT_UNAVAILABLE") != "1" {
		t.Skip("requires separately stopped owned legacy target")
	}
	raw, e := os.ReadFile(os.Getenv("LC06_RUNNER_LIVE_CONFIG"))
	if e != nil {
		t.Fatal(e)
	}
	var cfg struct {
		DefaultURL, DefaultToken string
		Routes                   []GoRunnerRoute
		Cases                    []struct {
			Name      string
			Manifest  domain.GoTestManifest
			Reference map[string]string
		}
	}
	if json.Unmarshal(raw, &cfg) != nil {
		t.Fatal("private config")
	}
	routes, _ := json.Marshal(cfg.Routes)
	client, e := NewConfiguredGoTestsClient(cfg.DefaultURL, cfg.DefaultToken, string(routes))
	if e != nil {
		t.Fatal(e)
	}
	for _, fixture := range cfg.Cases {
		if fixture.Name != "legacy" {
			continue
		}
		b := routeBinding(fixture.Manifest)
		b.Schema = "go-test-execution.v1"
		checks, _ := json.Marshal(map[string]any{"profile_id": b.ProfileID, "private_binding": b})
		input := domain.EngineValidationInput{Mode: "final", Stage: domain.ValidationStage{Checks: checks}}
		for path, content := range fixture.Reference {
			input.Workspace.Files = append(input.Workspace.Files, domain.WorkspaceFile{Path: path, Content: content})
		}
		if _, e = client.Validate(context.Background(), input); e == nil {
			t.Fatal("unavailable pinned target did not produce technical ERROR")
		}
		t.Log("real legacy target unavailable => technical ERROR, no substituted runner")
		return
	}
	t.Fatal("missing historical case")
}
