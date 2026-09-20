package engines

import (
	"context"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type WorkspaceFoundationClient struct {
	baseURL string
	http    jsonPoster
	engine  string
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
