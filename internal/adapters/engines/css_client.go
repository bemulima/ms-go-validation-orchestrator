package engines

import (
	"context"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type CSSClient struct {
	baseURL string
	http    HTTPClient
}

func NewCSSClient(baseURL string, httpClient HTTPClient) CSSClient {
	return CSSClient{
		baseURL: baseURL,
		http:    httpClient,
	}
}

func (client CSSClient) EngineID() string {
	return "css.ast"
}

func (client CSSClient) Validate(
	ctx context.Context,
	input domain.EngineValidationInput,
) (domain.StageExecutionResult, error) {
	body, ok := firstTargetContent(input.Stage, input.Workspace)
	if !ok {
		return domain.StageExecutionResult{
			Passed: false,
			Errors: []domain.ValidationIssue{
				{
					Code:     "CSS_TARGET_FILE_MISSING",
					Message:  "no target css file found in workspace",
					Severity: "error",
					StageID:  input.Stage.ID,
					Engine:   input.Stage.Engine,
				},
			},
		}, nil
	}

	responseBody, err := client.http.PostJSON(ctx, client.baseURL+"/api/v1/validate", map[string]any{
		"language": defaultString(input.Stage.Language, "css"),
		"code":     body,
		"rules":    rawJSONOrEmptyObject(input.Stage.Rules),
		"locale":   input.Locale,
		"meta": map[string]any{
			"path": firstTargetPath(input.Stage, input.Workspace),
		},
	})
	if err != nil {
		return domain.StageExecutionResult{}, err
	}

	return parseCommonValidationResponse(responseBody, input.Stage)
}

type SCSSClient struct {
	CSSClient
}

func NewSCSSClient(baseURL string, httpClient HTTPClient) SCSSClient {
	return SCSSClient{
		CSSClient: NewCSSClient(baseURL, httpClient),
	}
}

func (client SCSSClient) EngineID() string {
	return "scss.ast"
}

func firstTargetPath(stage domain.ValidationStage, workspace domain.ValidationWorkspace) string {
	for _, target := range stage.Targets.Files {
		for _, file := range workspace.Files {
			if file.Path == target {
				return file.Path
			}
		}
	}

	if len(workspace.Files) == 0 {
		return ""
	}

	return workspace.Files[0].Path
}
