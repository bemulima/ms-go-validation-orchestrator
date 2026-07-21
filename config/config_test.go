package config

import "testing"

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
