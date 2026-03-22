package public

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type HTTPValidationClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPValidationClient(baseURL string, client *http.Client) HTTPValidationClient {
	if client == nil {
		client = &http.Client{}
	}

	return HTTPValidationClient{
		baseURL: baseURL,
		client:  client,
	}
}

func (client HTTPValidationClient) Validate(
	ctx context.Context,
	request domain.ValidationRequest,
) (domain.ValidationResult, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return domain.ValidationResult{}, fmt.Errorf("marshal request: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		client.baseURL+"/api/v1/validate",
		bytes.NewReader(body),
	)
	if err != nil {
		return domain.ValidationResult{}, fmt.Errorf("create request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")

	response, err := client.client.Do(httpRequest)
	if err != nil {
		return domain.ValidationResult{}, fmt.Errorf("execute request: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return domain.ValidationResult{}, fmt.Errorf("read response: %w", err)
	}

	if response.StatusCode >= http.StatusBadRequest {
		return domain.ValidationResult{}, fmt.Errorf("unexpected status %d: %s", response.StatusCode, string(responseBody))
	}

	var result domain.ValidationResult
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return domain.ValidationResult{}, fmt.Errorf("decode response: %w", err)
	}

	return result, nil
}
