package engines

import (
	"context"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type PHPClient struct {
	baseURL string
	http    HTTPClient
}

func NewPHPClient(baseURL string, httpClient HTTPClient) PHPClient {
	return PHPClient{
		baseURL: baseURL,
		http:    httpClient,
	}
}

func (client PHPClient) EngineID() string {
	return "php.core"
}

func (client PHPClient) Validate(
	ctx context.Context,
	input domain.EngineValidationInput,
) (domain.StageExecutionResult, error) {
	body, ok := firstTargetContent(input.Stage, input.Workspace)
	if !ok {
		return domain.StageExecutionResult{
			Passed: false,
			Errors: []domain.ValidationIssue{
				{
					Code:     "PHP_TARGET_FILE_MISSING",
					Message:  "no target php file found in workspace",
					Severity: "error",
					StageID:  input.Stage.ID,
					Engine:   input.Stage.Engine,
				},
			},
		}, nil
	}

	responseBody, err := client.http.PostJSON(ctx, client.baseURL+"/api/v1/validate/rules", map[string]any{
		"code":       body,
		"phpVersion": "8.2",
		"rules":      rawJSONOrEmptyObject(input.Stage.Rules),
	})
	if err != nil {
		return domain.StageExecutionResult{}, err
	}

	return parseCommonValidationResponse(responseBody, input.Stage)
}
