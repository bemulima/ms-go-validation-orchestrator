package app

import (
	"github.com/example/ms-validation-orchestrator-service/config"
	"strings"
	"testing"
	"time"
)

func TestSharedWriteTimeoutRemainsThirtySeconds(t *testing.T) {
	cfg := config.Config{}
	cfg.PlannerVerificationToken = "planner"
	cfg.CatalogVerificationToken = "catalog"
	cfg.GoTestInternalToken = "go"
	cfg.Engines.Go = "http://unit.invalid"
	cfg.VerificationSigningKey = strings.Repeat("s", 32)
	cfg.VerificationKeyID = "unit"
	if New(cfg).Server().WriteTimeout != 30*time.Second {
		t.Fatal("shared timeout changed")
	}
	cfg.PlannerVerificationToken = ""
	cfg.CatalogVerificationToken = ""
	if New(cfg).Server().WriteTimeout != 30*time.Second {
		t.Fatal("default timeout changed")
	}
}
