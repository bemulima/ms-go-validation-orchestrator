package engines

import (
	"context"
	"encoding/json"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type HTTPRuntimeDispatchClient struct {
	genericClient domain.EngineClient
	nodeClient    domain.EngineClient
}

func NewHTTPRuntimeDispatchClient(
	genericClient domain.EngineClient,
	nodeClient domain.EngineClient,
) HTTPRuntimeDispatchClient {
	return HTTPRuntimeDispatchClient{
		genericClient: genericClient,
		nodeClient:    nodeClient,
	}
}

func (client HTTPRuntimeDispatchClient) EngineID() string {
	return "http.runtime"
}

func (client HTTPRuntimeDispatchClient) Validate(
	ctx context.Context,
	input domain.EngineValidationInput,
) (domain.StageExecutionResult, error) {
	if hasRuntimeCommand(input.Stage.Checks) && client.genericClient != nil {
		return client.genericClient.Validate(ctx, input)
	}
	if client.nodeClient != nil {
		return client.nodeClient.Validate(ctx, input)
	}
	if client.genericClient != nil {
		return client.genericClient.Validate(ctx, input)
	}

	return domain.StageExecutionResult{
		Passed: false,
		Errors: []domain.ValidationIssue{
			{
				Code:     "HTTP_RUNTIME_NOT_CONFIGURED",
				Message:  "http.runtime engine is not configured",
				Severity: "error",
				StageID:  input.Stage.ID,
				Engine:   input.Stage.Engine,
			},
		},
	}, nil
}

func hasRuntimeCommand(raw json.RawMessage) bool {
	if len(raw) == 0 || string(raw) == "null" {
		return false
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return false
	}

	_, exists := payload["command"]
	return exists
}
