package engines

import (
	"context"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type ReactClient struct {
	baseURL string
	http    jsonPoster
}

func NewReactClient(baseURL string, httpClient jsonPoster) ReactClient {
	return ReactClient{
		baseURL: baseURL,
		http:    httpClient,
	}
}

func (client ReactClient) EngineID() string {
	return "react.ast"
}

func (client ReactClient) Validate(
	ctx context.Context,
	input domain.EngineValidationInput,
) (domain.StageExecutionResult, error) {
	body, ok := firstTargetContent(input.Stage, input.Workspace)
	if !ok {
		return domain.StageExecutionResult{
			Passed: false,
			Errors: []domain.ValidationIssue{
				{
					Code:     "REACT_TARGET_FILE_MISSING",
					Message:  "no target react file found in workspace",
					Severity: "error",
					StageID:  input.Stage.ID,
					Engine:   input.Stage.Engine,
				},
			},
		}, nil
	}

	responseBody, err := client.http.PostJSON(ctx, client.baseURL+"/api/v1/validate", map[string]any{
		"taskId":    input.TaskID,
		"code":      body,
		"language":  defaultString(input.Stage.Language, "tsx"),
		"framework": defaultString(input.Stage.Framework, "react"),
		"rules":     rawJSONOrEmptyArray(input.Stage.Rules),
		"meta": map[string]any{
			"path": firstTargetPath(input.Stage, input.Workspace),
		},
	})
	if err != nil {
		return domain.StageExecutionResult{}, err
	}

	return parseCommonValidationResponse(responseBody, input.Stage)
}
