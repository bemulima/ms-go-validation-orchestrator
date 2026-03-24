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

func TestBuildEngineClientsRegistersGitAndDockerHooks(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Engines: config.EngineEndpoints{
			Git:    "http://ms-go-git-validator:8080",
			Docker: "http://ms-go-docker-validator:8080",
		},
	}

	clients := buildEngineClients(cfg, engines.NewHTTPClient(0))
	ids := make(map[string]struct{}, len(clients))
	for _, client := range clients {
		ids[client.EngineID()] = struct{}{}
	}

	for _, engineID := range []string{"git.core", "docker.dockerfile", "docker.compose"} {
		if _, ok := ids[engineID]; !ok {
			t.Fatalf("expected engine hook %s to be registered", engineID)
		}
	}
}

func TestBuildEngineClientsRegistersPythonAndGoHooks(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Engines: config.EngineEndpoints{
			Python: "http://ms-py-validator:8080",
			Go:     "http://ms-go-code-validator:8080",
		},
	}

	clients := buildEngineClients(cfg, engines.NewHTTPClient(0))
	ids := make(map[string]struct{}, len(clients))
	for _, client := range clients {
		ids[client.EngineID()] = struct{}{}
	}

	for _, engineID := range []string{"python.core", "python.django", "golang", "go.core", "go.gin", "go.echo"} {
		if _, ok := ids[engineID]; !ok {
			t.Fatalf("expected engine hook %s to be registered", engineID)
		}
	}
}

func TestBuildEngineClientsRegistersGenericHTTPRuntimeHook(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Engines: config.EngineEndpoints{
			HTTPRuntime: "http://ms-go-http-runtime-validator:8080",
		},
	}

	clients := buildEngineClients(cfg, engines.NewHTTPClient(0))
	ids := make(map[string]struct{}, len(clients))
	for _, client := range clients {
		ids[client.EngineID()] = struct{}{}
	}

	if _, ok := ids["http.runtime"]; !ok {
		t.Fatalf("expected engine hook http.runtime to be registered")
	}

	for _, engineID := range []string{
		"python.django.runtime",
		"go.gin.runtime",
		"go.echo.runtime",
		"php.laravel.runtime",
		"php.symfony.runtime",
		"php.yii2.runtime",
		"php.yii3.runtime",
	} {
		if _, ok := ids[engineID]; !ok {
			t.Fatalf("expected engine hook %s to be registered", engineID)
		}
	}
}

func TestBuildEngineClientsRegistersDBHooks(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Engines: config.EngineEndpoints{
			DB: "http://ms-go-db-validator:8080",
		},
	}

	clients := buildEngineClients(cfg, engines.NewHTTPClient(0))
	ids := make(map[string]struct{}, len(clients))
	for _, client := range clients {
		ids[client.EngineID()] = struct{}{}
	}

	for _, engineID := range []string{
		"db.postgres.schema",
		"db.postgres.runtime",
		"db.mysql.schema",
		"db.mysql.runtime",
		"db.tarantool.schema",
		"db.tarantool.runtime",
	} {
		if _, ok := ids[engineID]; !ok {
			t.Fatalf("expected engine hook %s to be registered", engineID)
		}
	}
}

func TestBuildEngineClientsRegistersLinuxHooks(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Engines: config.EngineEndpoints{
			Linux: "http://ms-go-linux-validator:8090",
		},
	}

	clients := buildEngineClients(cfg, engines.NewHTTPClient(0))
	ids := make(map[string]struct{}, len(clients))
	for _, client := range clients {
		ids[client.EngineID()] = struct{}{}
	}

	for _, engineID := range []string{"linux.fs", "linux.cli", "linux.runtime"} {
		if _, ok := ids[engineID]; !ok {
			t.Fatalf("expected engine hook %s to be registered", engineID)
		}
	}
}

func TestBuildEngineClientsRegistersCacheSearchHooks(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Engines: config.EngineEndpoints{
			CacheSearch: "http://ms-go-cache-search-validator:8091",
		},
	}

	clients := buildEngineClients(cfg, engines.NewHTTPClient(0))
	ids := make(map[string]struct{}, len(clients))
	for _, client := range clients {
		ids[client.EngineID()] = struct{}{}
	}

	for _, engineID := range []string{
		"cache.redis.config",
		"cache.redis.runtime",
		"search.elasticsearch.mapping",
		"search.elasticsearch.runtime",
		"search.manticore",
		"search.sphinx",
	} {
		if _, ok := ids[engineID]; !ok {
			t.Fatalf("expected engine hook %s to be registered", engineID)
		}
	}
}
