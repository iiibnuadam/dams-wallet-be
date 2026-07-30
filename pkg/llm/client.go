// Package llm is a thin, generic wrapper around DeepSeek's OpenAI-compatible
// chat completions API. A Client with no API key is "disabled" and every
// method becomes a safe no-op -- callers never need to branch on
// "is AI configured".
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const defaultBaseURL = "https://api.deepseek.com/chat/completions"
const defaultModel = "deepseek-chat" // DeepSeek's fast, cheap general-purpose model
const defaultTimeout = 15 * time.Second

type Config struct {
	APIKey  string
	Model   string
	Timeout time.Duration
}

type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	model      string
	timeout    time.Duration
	enabled    bool
}

// New builds a Client. If cfg.APIKey is empty, the returned Client is
// disabled and GenerateJSON always returns an error without making any
// network call -- callers should check Enabled() first, or simply rely on
// the error to trigger their own fallback path.
func New(cfg Config) *Client {
	if cfg.APIKey == "" {
		return &Client{enabled: false}
	}
	model := cfg.Model
	if model == "" {
		model = defaultModel
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    defaultBaseURL,
		apiKey:     cfg.APIKey,
		model:      model,
		timeout:    timeout,
		enabled:    true,
	}
}

func (c *Client) Enabled() bool { return c.enabled }
func (c *Client) Timeout() time.Duration {
	if c.timeout <= 0 {
		return defaultTimeout
	}
	return c.timeout
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string         `json:"model"`
	Messages       []chatMessage  `json:"messages"`
	ResponseFormat responseFormat `json:"response_format"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// GenerateJSON sends a single, non-streaming chat completion request in
// DeepSeek's JSON-object mode and returns the raw JSON text of the
// response. The expected output shape must be described in systemPrompt --
// DeepSeek's json_object mode guarantees syntactically valid JSON, not
// conformance to an externally supplied schema.
func (c *Client) GenerateJSON(ctx context.Context, systemPrompt, userPrompt string) (json.RawMessage, error) {
	if !c.enabled {
		return nil, fmt.Errorf("llm: client is disabled (no API key configured)")
	}

	reqBody, err := json.Marshal(chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		ResponseFormat: responseFormat{Type: "json_object"},
	})
	if err != nil {
		return nil, fmt.Errorf("llm: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("llm: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("llm: request failed: %w", err)
	}
	defer resp.Body.Close()

	var parsed chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("llm: decode response: %w", err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("llm: api error: %s", parsed.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("llm: unexpected status %d", resp.StatusCode)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("llm: response had no choices")
	}
	return json.RawMessage(parsed.Choices[0].Message.Content), nil
}
