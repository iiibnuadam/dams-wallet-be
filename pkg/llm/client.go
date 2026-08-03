// Package llm is a thin, generic wrapper around OpenAI-compatible
// chat completions APIs. It supports two providers out of the box:
//   - "deepseek"  -> https://api.deepseek.com/chat/completions
//   - "huggingface" -> configurable Hugging Face Inference Endpoint
//
// A Client with no API key is "disabled" only when the chosen provider
// requires a key and none is supplied. Every method is safe to call on a
// disabled client -- callers never need to branch on "is AI configured".
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	defaultProvider = "deepseek"

	deepseekBaseURL = "https://api.deepseek.com/chat/completions"
	deepseekModel   = "deepseek-chat"

	// Default Hugging Face endpoint supplied by the user.
	huggingfaceBaseURL = "https://q5dh1rfszfym23hj.us-east-2.aws.endpoints.huggingface.cloud/v1/chat/completions"
	huggingfaceModel   = "deepseek-ai/DeepSeek-V4-Flash-0731"
)

// A single "Analisis dengan AI" call asks the model to narrate every
// signal plus talking points in one non-streaming response, which can
// take well over 15s depending on how many signals there are and how busy
// the provider is -- 60s gives it realistic room without hanging forever.
const defaultTimeout = 60 * time.Second

type Config struct {
	Provider        string        // "deepseek" or "huggingface"
	BaseURL         string        // optional provider endpoint override
	APIKey          string        // Bearer token; required for deepseek, optional for huggingface
	Model           string        // model name; provider-specific defaults apply if empty
	Timeout         time.Duration // request/ overall client timeout
	Temperature     float64       // optional sampling temperature (huggingface)
	TopP            float64       // optional nucleus sampling (huggingface)
	ReasoningEffort string        // optional reasoning effort (huggingface)
}

type Client struct {
	httpClient      *http.Client
	provider        string
	baseURL         string
	apiKey          string
	model           string
	timeout         time.Duration
	enabled         bool
	temperature     float64
	topP            float64
	reasoningEffort string
}

// New builds a Client. If cfg.Provider is empty it defaults to "deepseek".
// The DeepSeek provider requires an API key; if missing the client is
// disabled. The Hugging Face provider is enabled even without an API key
// because the supplied endpoint may be public/unauthenticated.
func New(cfg Config) *Client {
	provider := strings.ToLower(cfg.Provider)
	if provider == "" {
		provider = defaultProvider
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	switch provider {
	case "huggingface":
		baseURL := cfg.BaseURL
		if baseURL == "" {
			baseURL = huggingfaceBaseURL
		}
		model := cfg.Model
		if model == "" {
			model = huggingfaceModel
		}
		return &Client{
			httpClient:      &http.Client{Timeout: timeout},
			provider:        provider,
			baseURL:         baseURL,
			apiKey:          cfg.APIKey,
			model:           model,
			timeout:         timeout,
			enabled:         true,
			temperature:     cfg.Temperature,
			topP:            cfg.TopP,
			reasoningEffort: cfg.ReasoningEffort,
		}

	case "deepseek":
		fallthrough
	default:
		if cfg.APIKey == "" {
			return &Client{provider: provider, enabled: false}
		}
		baseURL := cfg.BaseURL
		if baseURL == "" {
			baseURL = deepseekBaseURL
		}
		model := cfg.Model
		if model == "" {
			model = deepseekModel
		}
		return &Client{
			httpClient: &http.Client{Timeout: timeout},
			provider:   provider,
			baseURL:    baseURL,
			apiKey:     cfg.APIKey,
			model:      model,
			timeout:    timeout,
			enabled:    true,
		}
	}
}

func (c *Client) Enabled() bool { return c.enabled }
func (c *Client) Provider() string {
	if c.provider == "" {
		return defaultProvider
	}
	return c.provider
}
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

type responseFormat struct {
	Type string `json:"type"`
}

type chatRequest struct {
	Model           string          `json:"model"`
	Messages        []chatMessage   `json:"messages"`
	ResponseFormat  *responseFormat `json:"response_format,omitempty"`
	Temperature     float64         `json:"temperature,omitempty"`
	TopP            float64         `json:"top_p,omitempty"`
	ReasoningEffort string          `json:"reasoning_effort,omitempty"`
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

// GenerateJSON sends a single, non-streaming chat completion request and
// returns the raw JSON text of the response. The expected output shape must
// be described in systemPrompt -- neither provider guarantees conformance to
// an externally supplied schema.
func (c *Client) GenerateJSON(ctx context.Context, systemPrompt, userPrompt string) (json.RawMessage, error) {
	if !c.enabled {
		return nil, fmt.Errorf("llm: client is disabled (provider %q is not configured)", c.Provider())
	}

	reqBody := chatRequest{
		Model:    c.model,
		Messages: []chatMessage{{Role: "system", Content: systemPrompt}, {Role: "user", Content: userPrompt}},
	}

	// Provider-specific request shape.
	switch c.provider {
	case "huggingface":
		reqBody.Temperature = c.temperature
		reqBody.TopP = c.topP
		reqBody.ReasoningEffort = c.reasoningEffort
	default:
		// DeepSeek supports json_object mode which makes it far more likely
		// the model returns syntactically valid JSON.
		reqBody.ResponseFormat = &responseFormat{Type: "json_object"}
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("llm: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("llm: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

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

	return json.RawMessage(cleanJSON(parsed.Choices[0].Message.Content)), nil
}

// cleanJSON strips surrounding whitespace and optional markdown code fences
// so that a model returning ```json {...} ``` can still be parsed.
func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}
	return s
}
