package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"monitoring-platform/internal/domain"
)

const (
	sendAttempts   = 3
	sendBackoff    = time.Second
	requestTimeout = 10 * time.Second
)

// ResultClient delivers probe results to the control plane ingestion API.
type ResultClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewResultClient(baseURL, token string) *ResultClient {
	return &ResultClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		token:      token,
		httpClient: &http.Client{Timeout: requestTimeout},
	}
}

func (c *ResultClient) Send(ctx context.Context, result domain.ProbeResult) error {
	payload, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal probe result: %w", err)
	}

	var lastErr error

	for attempt := 1; attempt <= sendAttempts; attempt++ {
		lastErr = c.post(ctx, payload)
		if lastErr == nil {
			return nil
		}

		if attempt < sendAttempts {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(sendBackoff * time.Duration(attempt)):
			}
		}
	}

	return lastErr
}

func (c *ResultClient) post(ctx context.Context, payload []byte) error {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/internal/v1/results",
		bytes.NewReader(payload),
	)
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+c.token)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("ingestion API returned %s: %s", response.Status, string(body))
	}

	return nil
}
