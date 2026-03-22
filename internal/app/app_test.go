package app

import (
	"testing"

	"github.com/example/ms-validation-orchestrator-service/config"
	"github.com/example/ms-validation-orchestrator-service/internal/adapters/engines"
)

func TestBuildEngineClientsRegistersPHPFrameworkHooks(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Engines: config.EngineEndpoints{
			PHPFramework: "http://ms-php-framework-validator:3018",
		},
	}

	clients := buildEngineClients(cfg, engines.NewHTTPClient(0))
	ids := make(map[string]struct{}, len(clients))
	for _, client := range clients {
		ids[client.EngineID()] = struct{}{}
	}

	for _, engineID := range []string{"php.laravel", "php.yii2", "php.yii3", "php.symfony"} {
		if _, ok := ids[engineID]; !ok {
			t.Fatalf("expected engine hook %s to be registered", engineID)
		}
	}
}
