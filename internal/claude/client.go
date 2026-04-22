package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DefaultModel is the model identifier used when no override is given.
const DefaultModel = "claude-sonnet-4-5-20250929"

// DefaultAPIURL is the Anthropic Messages endpoint.
const DefaultAPIURL = "https://api.anthropic.com/v1/messages"

// Client sends requests to the Anthropic Messages API.
type Client struct {
	apiKey     string
	apiURL     string
	model      string
	httpClient *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithModel overrides the default model.
func WithModel(model string) Option { return func(c *Client) { c.model = model } }

// WithHTTPClient injects a custom HTTP client (useful for tests with mocked transports).
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.httpClient = h } }

// WithAPIURL overrides the API endpoint (useful for tests).
func WithAPIURL(u string) Option { return func(c *Client) { c.apiURL = u } }

// NewClient constructs a Client. apiKey is required.
func NewClient(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey: apiKey,
		apiURL: DefaultAPIURL,
		model:  DefaultModel,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Model returns the model identifier this client uses.
func (c *Client) Model() string { return c.model }

// CreateMessage sends a single request to /v1/messages and returns the raw response.
// It does not run the agentic loop — that logic lives in the caller.
func (c *Client) CreateMessage(ctx context.Context, req MessagesRequest) (*MessagesResponse, error) {
	if req.Model == "" {
		req.Model = c.model
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = 4096
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		var wrapper struct {
			Error APIError `json:"error"`
		}
		if err := json.Unmarshal(respBody, &wrapper); err == nil {
			apiErr.Type = wrapper.Error.Type
			apiErr.Message = wrapper.Error.Message
		} else {
			apiErr.Type = "unknown"
			apiErr.Message = string(respBody)
		}
		return nil, apiErr
	}

	var out MessagesResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &out, nil
}
