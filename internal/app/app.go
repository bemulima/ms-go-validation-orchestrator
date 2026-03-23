package app

import (
	"fmt"
	"net/http"
	"time"

	"github.com/example/ms-validation-orchestrator-service/config"
	"github.com/example/ms-validation-orchestrator-service/internal/adapters/engines"
	"github.com/example/ms-validation-orchestrator-service/internal/domain"
	"github.com/example/ms-validation-orchestrator-service/internal/infrastructure/logging"
	"github.com/example/ms-validation-orchestrator-service/internal/usecase"
	transporthttp "github.com/example/ms-validation-orchestrator-service/transport/http"
	api "github.com/example/ms-validation-orchestrator-service/transport/http/api/v1"
)

type App struct {
	server *http.Server
}

func New(cfg config.Config) App {
	httpClient := engines.NewHTTPClient(30 * time.Second)

	legacyAdapter := usecase.NewDefaultLegacyContractAdapter()
	parser := usecase.NewContractParser(legacyAdapter)

	orchestrator := usecase.NewOrchestrateValidationUseCase(
		parser,
		buildEngineClients(cfg, httpClient),
	)

	logger := logging.NewStdLogger()
	apiHandler := api.NewHandler(orchestrator, logger)
	router := transporthttp.NewRouter(transporthttp.Dependencies{
		APIHandler: apiHandler,
	})

	server := &http.Server{
		Addr:         cfg.HTTP.Host + ":" + itoa(cfg.HTTP.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return App{server: server}
}

func (application App) Server() *http.Server {
	return application.server
}

func buildEngineClients(
	cfg config.Config,
	httpClient engines.HTTPClient,
) []domain.EngineClient {
	engineClients := []domain.EngineClient{
		engines.NewLegacyGenericEngine(),
	}

	if cfg.Engines.HTML != "" {
		engineClients = append(engineClients, engines.NewHTMLClient(cfg.Engines.HTML, httpClient))
	}

	if cfg.Engines.CSS != "" {
		engineClients = append(
			engineClients,
			engines.NewCSSClient(cfg.Engines.CSS, httpClient),
			engines.NewSCSSClient(cfg.Engines.CSS, httpClient),
		)
	}

	if cfg.Engines.React != "" {
		engineClients = append(engineClients, engines.NewReactClient(cfg.Engines.React, httpClient))
	}

	if cfg.Engines.Node != "" {
		nodeRuntimeClient := engines.NewNodeClient(cfg.Engines.Node, httpClient, "http.runtime")
		engineClients = append(
			engineClients,
			engines.NewNodeClient(cfg.Engines.Node, httpClient, "js.ast"),
			engines.NewNodeClient(cfg.Engines.Node, httpClient, "ts.ast"),
			engines.NewNodeClient(cfg.Engines.Node, httpClient, "node.express"),
			engines.NewNodeClient(cfg.Engines.Node, httpClient, "node.fastify"),
			engines.NewNodeClient(cfg.Engines.Node, httpClient, "node.nest"),
		)
		if cfg.Engines.HTTPRuntime != "" {
			engineClients = append(
				engineClients,
				engines.NewHTTPRuntimeDispatchClient(
					engines.NewWorkspaceFoundationClient(cfg.Engines.HTTPRuntime, httpClient, "http.runtime"),
					nodeRuntimeClient,
				),
			)
		} else {
			engineClients = append(engineClients, nodeRuntimeClient)
		}
	} else if cfg.Engines.HTTPRuntime != "" {
		engineClients = append(
			engineClients,
			engines.NewWorkspaceFoundationClient(cfg.Engines.HTTPRuntime, httpClient, "http.runtime"),
		)
	}

	if cfg.Engines.HTTPRuntime != "" {
		engineClients = append(
			engineClients,
			engines.NewWorkspaceFoundationClient(cfg.Engines.HTTPRuntime, httpClient, "python.django.runtime"),
			engines.NewWorkspaceFoundationClient(cfg.Engines.HTTPRuntime, httpClient, "go.gin.runtime"),
			engines.NewWorkspaceFoundationClient(cfg.Engines.HTTPRuntime, httpClient, "go.echo.runtime"),
			engines.NewWorkspaceFoundationClient(cfg.Engines.HTTPRuntime, httpClient, "php.laravel.runtime"),
			engines.NewWorkspaceFoundationClient(cfg.Engines.HTTPRuntime, httpClient, "php.symfony.runtime"),
			engines.NewWorkspaceFoundationClient(cfg.Engines.HTTPRuntime, httpClient, "php.yii2.runtime"),
			engines.NewWorkspaceFoundationClient(cfg.Engines.HTTPRuntime, httpClient, "php.yii3.runtime"),
		)
	}

	if cfg.Engines.PHP != "" {
		engineClients = append(engineClients, engines.NewPHPClient(cfg.Engines.PHP, httpClient))
	}

	if cfg.Engines.PHPFramework != "" {
		engineClients = append(
			engineClients,
			engines.NewWorkspaceFoundationClient(cfg.Engines.PHPFramework, httpClient, "php.laravel"),
			engines.NewWorkspaceFoundationClient(cfg.Engines.PHPFramework, httpClient, "php.yii2"),
			engines.NewWorkspaceFoundationClient(cfg.Engines.PHPFramework, httpClient, "php.yii3"),
			engines.NewWorkspaceFoundationClient(cfg.Engines.PHPFramework, httpClient, "php.symfony"),
		)
	}

	if cfg.Engines.NextJS != "" {
		engineClients = append(
			engineClients,
			engines.NewWorkspaceFoundationClient(cfg.Engines.NextJS, httpClient, "nextjs.app"),
		)
	}

	if cfg.Engines.Browser != "" {
		engineClients = append(
			engineClients,
			engines.NewWorkspaceFoundationClient(cfg.Engines.Browser, httpClient, "browser.runtime"),
		)
	}

	if cfg.Engines.Git != "" {
		engineClients = append(
			engineClients,
			engines.NewWorkspaceFoundationClient(cfg.Engines.Git, httpClient, "git.core"),
		)
	}

	if cfg.Engines.Docker != "" {
		engineClients = append(
			engineClients,
			engines.NewWorkspaceFoundationClient(cfg.Engines.Docker, httpClient, "docker.dockerfile"),
			engines.NewWorkspaceFoundationClient(cfg.Engines.Docker, httpClient, "docker.compose"),
		)
	}

	if cfg.Engines.Python != "" {
		engineClients = append(
			engineClients,
			engines.NewWorkspaceFoundationClient(cfg.Engines.Python, httpClient, "python.core"),
			engines.NewWorkspaceFoundationClient(cfg.Engines.Python, httpClient, "python.django"),
		)
	}

	if cfg.Engines.Go != "" {
		engineClients = append(
			engineClients,
			engines.NewWorkspaceFoundationClient(cfg.Engines.Go, httpClient, "go.core"),
			engines.NewWorkspaceFoundationClient(cfg.Engines.Go, httpClient, "go.gin"),
			engines.NewWorkspaceFoundationClient(cfg.Engines.Go, httpClient, "go.echo"),
		)
	}

	return engineClients
}

func itoa(value int) string {
	return fmt.Sprintf("%d", value)
}
