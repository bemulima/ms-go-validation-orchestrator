package engines

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type WorkspaceFoundationClient struct {
	baseURL string
	http    jsonPoster
	engine  string
}

func (client WorkspaceFoundationClient) NeedsWorkspaceFiles(stage domain.ValidationStage) bool {
	// Provider-owned inputs, not the heuristic public capability projection.
	switch client.engine {
	// Code, PHP framework and Python DTOs consume only inline files. NextJS
	// and Browser materialize Files; Docker selects its target from Files.
	case "golang", "go.core", "go.gin", "go.echo", "java.compile", "java.runtime", "kotlin.compile", "kotlin.runtime",
		"php.laravel", "php.yii2", "php.yii3", "php.symfony", "python.core", "python.django", "nextjs.app", "browser.runtime", "docker.dockerfile", "docker.compose",
		"db.postgres.schema", "db.mysql.schema", "db.tarantool.schema":
		return true
	case "db.postgres.runtime", "db.mysql.runtime":
		// collectSetupStatements consumes Files only for authored setupFiles.
		var checks struct {
			SetupFiles []string `json:"setupFiles"`
		}
		return json.Unmarshal(stage.Checks, &checks) == nil && len(checks.SetupFiles) > 0
	case "cache.redis.config", "search.elasticsearch.mapping", "search.manticore", "search.sphinx":
		// targetWorkspaceFiles reads explicit targets from RootPath as fallback;
		// without targets it can enumerate only the inline Files projection.
		return len(stage.Targets.Files) == 0
	default:
		// Git, Linux, dedicated HTTP/framework runtime and runtime cache/search
		// retain their existing root/connection prerequisites. Unclassified
		// foundation variants are not claimed as repaired Files consumers.
		return false
	}
}

func NewWorkspaceFoundationClient(
	baseURL string,
	httpClient jsonPoster,
	engine string,
) WorkspaceFoundationClient {
	return WorkspaceFoundationClient{
		baseURL: baseURL,
		http:    httpClient,
		engine:  engine,
	}
}

func (client WorkspaceFoundationClient) EngineID() string {
	return client.engine
}

func (client WorkspaceFoundationClient) Validate(
	ctx context.Context,
	input domain.EngineValidationInput,
) (domain.StageExecutionResult, error) {
	if strings.TrimSpace(input.Workspace.RootPath) == "" {
		// These providers require repository or process state that inline
		// path/content files cannot supply. Keep Node HTTP and Linux Files
		// inputs on their existing provider paths.
		switch client.engine {
		case "git.core", "http.runtime", "python.django.runtime", "go.gin.runtime", "go.echo.runtime",
			"php.laravel.runtime", "php.symfony.runtime", "php.yii2.runtime", "php.yii3.runtime":
			return domain.StageExecutionResult{}, fmt.Errorf("%w: workspace execution prerequisite unavailable", domain.ErrUnsupportedEngine)
		}
	}
	responseBody, err := client.http.PostJSON(ctx, client.baseURL+"/api/v1/validate", map[string]any{
		"taskId":       input.TaskID,
		"locale":       input.Locale,
		"mode":         input.Mode,
		"taskMetadata": input.TaskMetadata,
		"stage": map[string]any{
			"id":             input.Stage.ID,
			"name":           input.Stage.Name,
			"engine":         input.Stage.Engine,
			"language":       input.Stage.Language,
			"framework":      input.Stage.Framework,
			"optional":       input.Stage.Optional,
			"dependsOn":      input.Stage.DependsOn,
			"timeoutSeconds": input.Stage.TimeoutSeconds,
			"targets": map[string]any{
				"files":      input.Stage.Targets.Files,
				"entrypoint": input.Stage.Targets.Entrypoint,
			},
			"rules":  rawJSONOrValue(input.Stage.Rules),
			"checks": rawJSONOrValue(input.Stage.Checks),
		},
		"workspace": map[string]any{
			"files":     workspaceFilesAsMaps(input.Workspace.Files),
			"root_path": input.Workspace.RootPath,
		},
	})
	if err != nil {
		return domain.StageExecutionResult{}, err
	}

	return parseCommonValidationResponse(responseBody, input.Stage)
}
