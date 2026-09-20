package engines

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type HTTPClient struct {
	client *http.Client
}

const maxEngineResponseBytes int64 = 4 << 20

func NewHTTPClient(timeout time.Duration) HTTPClient {
	return HTTPClient{
		client: &http.Client{Timeout: timeout},
	}
}

func (client HTTPClient) PostJSON(
	ctx context.Context,
	url string,
	payload any,
) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := client.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxEngineResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if int64(len(responseBody)) > maxEngineResponseBytes {
		return nil, fmt.Errorf("read response: engine response is too large")
	}

	if response.StatusCode >= http.StatusBadRequest && response.StatusCode != http.StatusUnprocessableEntity {
		return responseBody, fmt.Errorf("unexpected status %d", response.StatusCode)
	}

	return responseBody, nil
}
