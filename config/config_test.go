package config

import "testing"

func TestLoadVerificationWorkspaceMountConfiguration(t *testing.T) {
	t.Setenv("VERIFICATION_WORKSPACES_DIR", "")
	cfg, err := Load()
	if err != nil || cfg.VerificationWorkspacesDir != "/workspaces" {
		t.Fatalf("default verification mount: %q err=%v", cfg.VerificationWorkspacesDir, err)
	}
	t.Setenv("VERIFICATION_WORKSPACES_DIR", "/private/owned/verification")
	cfg, err = Load()
	if err != nil || cfg.VerificationWorkspacesDir != "/private/owned/verification" {
		t.Fatalf("configured verification mount: %q err=%v", cfg.VerificationWorkspacesDir, err)
	}
}

func TestLoadCodeValidatorURLFallsBackToLegacyGoVariable(t *testing.T) {
	t.Setenv("CODE_VALIDATOR_URL", "")
	t.Setenv("GO_CODE_VALIDATOR_URL", "http://legacy-code-validator:8080/")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Engines.Code != "http://legacy-code-validator:8080" {
		t.Fatalf("unexpected code validator URL %q", cfg.Engines.Code)
	}
	if cfg.Engines.Go != "http://legacy-code-validator:8080" {
		t.Fatalf("unexpected legacy Go validator URL %q", cfg.Engines.Go)
	}
}

func TestLoadPrefersGenericCodeValidatorURL(t *testing.T) {
	t.Setenv("CODE_VALIDATOR_URL", "http://code-validator:8080/")
	t.Setenv("GO_CODE_VALIDATOR_URL", "http://legacy-go-validator:8080/")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Engines.Code != "http://code-validator:8080" {
		t.Fatalf("unexpected code validator URL %q", cfg.Engines.Code)
	}
	if cfg.Engines.Go != "http://legacy-go-validator:8080" {
		t.Fatalf("unexpected legacy Go validator URL %q", cfg.Engines.Go)
	}
}

func TestLoadPracticeSnapshotReaderConfiguration(t *testing.T) {
	t.Setenv("SANDBOX_SERVICE_BASE_URL", " http://sandbox:8080/ ")
	t.Setenv("SANDBOX_SERVICE_INTERNAL_TOKEN", "sandbox-service-token")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.SandboxServiceBaseURL != "http://sandbox:8080" || cfg.SandboxServiceInternalToken != "sandbox-service-token" {
		t.Fatalf("unexpected Sandbox configuration: %+v", cfg)
	}
}
