package engines

import (
	"context"
	"encoding/json"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type NodeClient struct {
	baseURL string
	http    jsonPoster
	engine  string
}

type jsonPoster interface {
	PostJSON(ctx context.Context, url string, payload any) ([]byte, error)
}

func NewNodeClient(baseURL string, httpClient jsonPoster, engine string) NodeClient {
	return NodeClient{
		baseURL: baseURL,
		http:    httpClient,
		engine:  engine,
	}
}

func (client NodeClient) EngineID() string {
	return client.engine
}

func (client NodeClient) Validate(
	ctx context.Context,
	input domain.EngineValidationInput,
) (domain.StageExecutionResult, error) {
	staticMode, structureMode, runtimeMode := nodeStageModes(client.engine, input.Stage)

	responseBody, err := client.http.PostJSON(ctx, client.baseURL+"/api/v1/validate-node", map[string]any{
		"language":  defaultString(input.Stage.Language, inferLanguageFromEngine(client.engine)),
		"framework": defaultString(input.Stage.Framework, inferFrameworkFromEngine(client.engine)),
		"mode": map[string]bool{
			"static":    staticMode,
			"structure": structureMode,
			"runtime":   runtimeMode,
		},
		"taskMeta": map[string]any{
			"taskId": input.TaskID,
		},
		"code": map[string]any{
			"entrypoint": input.Stage.Targets.Entrypoint,
			"files":      workspaceFilesAsMaps(input.Workspace.Files),
		},
		"rules": map[string]any{
			"static":    rawJSONOrEmptyObject(input.Stage.Rules),
			"structure": rawJSONOrEmptyObject(input.Stage.Rules),
			"runtime":   rawJSONOrEmptyObject(input.Stage.Checks),
		},
	})
	if err != nil {
		return domain.StageExecutionResult{}, err
	}

	return parseNodeValidationResponse(responseBody, input.Stage)
}

func inferLanguageFromEngine(engine string) string {
	switch engine {
	case "ts.ast", "node.nest":
		return "ts"
	default:
		return "js"
	}
}

func inferFrameworkFromEngine(engine string) string {
	switch engine {
	case "node.express":
		return "express"
	case "node.fastify":
		return "fastify"
	case "node.nest":
		return "nestjs"
	default:
		return "none"
	}
}

func nodeStageModes(engine string, stage domain.ValidationStage) (bool, bool, bool) {
	switch engine {
	case "js.ast", "ts.ast":
		return true, false, false
	case "http.runtime":
		return false, false, true
	default:
		return true, hasPayload(stage.Rules), hasPayload(stage.Checks)
	}
}

func parseNodeValidationResponse(
	body []byte,
	stage domain.ValidationStage,
) (domain.StageExecutionResult, error) {
	type validationError struct {
		Code     string `json:"code"`
		Level    string `json:"level"`
		Message  string `json:"message"`
		Location struct {
			File   string `json:"file"`
			Line   int    `json:"line"`
			Column int    `json:"column"`
		} `json:"location"`
		Meta map[string]any `json:"meta"`
	}

	type validationSummary struct {
		StaticOk    bool `json:"staticOk"`
		StructureOk bool `json:"structureOk"`
		RuntimeOk   bool `json:"runtimeOk"`
	}

	type response struct {
		OK      bool              `json:"ok"`
		Summary validationSummary `json:"summary"`
		Errors  []validationError `json:"errors"`
	}

	var payload response
	if err := json.Unmarshal(body, &payload); err != nil {
		return domain.StageExecutionResult{}, err
	}

	passed := payload.OK
	if !payload.OK && len(payload.Errors) == 0 {
		passed = payload.Summary.StaticOk && payload.Summary.StructureOk && payload.Summary.RuntimeOk
	}

	errors := make([]domain.ValidationIssue, 0, len(payload.Errors))
	for _, item := range payload.Errors {
		errors = append(errors, domain.ValidationIssue{
			Code:     defaultString(item.Code, "VALIDATION_FAILED"),
			Message:  item.Message,
			Severity: defaultString(item.Level, "error"),
			StageID:  stage.ID,
			Engine:   stage.Engine,
			File:     item.Location.File,
			Line:     item.Location.Line,
			Column:   item.Location.Column,
			Selector: metaString(item.Meta, "selector"),
			Route:    metaString(item.Meta, "route"),
			Symbol:   metaString(item.Meta, "symbol"),
			Property: metaString(item.Meta, "property"),
			Hint:     metaString(item.Meta, "hint"),
		})
	}

	return domain.StageExecutionResult{
		Passed:    passed,
		Errors:    errors,
		RawResult: body,
	}, nil
}

func metaString(meta map[string]any, key string) string {
	if meta == nil {
		return ""
	}

	value, ok := meta[key]
	if !ok {
		return ""
	}

	stringValue, ok := value.(string)
	if !ok {
		return ""
	}

	return stringValue
}
