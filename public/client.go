package public

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/example/ms-validation-orchestrator-service/internal/domain"
)

type HTTPValidationClient struct {
	baseURL       string
	internalToken string
	client        *http.Client
}

const maxOrchestratorResponseBytes int64 = 4 << 20

func NewHTTPValidationClient(baseURL string, client *http.Client) HTTPValidationClient {
	return NewAuthenticatedHTTPValidationClient(baseURL, "", client)
}

func NewAuthenticatedHTTPValidationClient(
	baseURL string,
	internalToken string,
	client *http.Client,
) HTTPValidationClient {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	return HTTPValidationClient{
		baseURL:       baseURL,
		internalToken: internalToken,
		client:        client,
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
	if client.internalToken != "" {
		httpRequest.Header.Set("X-Internal-Token", client.internalToken)
	}

	response, err := client.client.Do(httpRequest)
	if err != nil {
		return domain.ValidationResult{}, fmt.Errorf("execute request: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxOrchestratorResponseBytes+1))
	if err != nil {
		return domain.ValidationResult{}, fmt.Errorf("read response: %w", err)
	}
	if int64(len(responseBody)) > maxOrchestratorResponseBytes {
		return domain.ValidationResult{}, fmt.Errorf("read response: validation response is too large")
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
