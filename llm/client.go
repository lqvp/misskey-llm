package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"misskey-llm/logger"
)

type Client struct {
	endpoint       string
	apiKey         string
	model          string
	maxTokens      int
	temperature    float64
	topP           float64
	enableSearch   bool
	enableThinking bool
	tools          []Tool
	httpClient     *http.Client
}

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Function struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function,omitempty"`

	MaxKeyword  int  `json:"max_keyword,omitempty"`
	ForceSearch bool `json:"force_search,omitempty"`
	Limit       int  `json:"limit,omitempty"`
}

type ThinkingConfig struct {
	Type string `json:"type"`
}

type ChatRequest struct {
	Model               string          `json:"model"`
	Messages            []Message       `json:"messages"`
	MaxCompletionTokens int             `json:"max_completion_tokens,omitempty"`
	Temperature         float64         `json:"temperature,omitempty"`
	TopP                float64         `json:"top_p,omitempty"`
	Stream              bool            `json:"stream"`
	Stop                interface{}     `json:"stop,omitempty"`
	FrequencyPenalty    float64         `json:"frequency_penalty,omitempty"`
	PresencePenalty     float64         `json:"presence_penalty,omitempty"`
	Tools               []Tool          `json:"tools,omitempty"`
	Thinking            *ThinkingConfig `json:"thinking,omitempty"`
}

type ChatResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Choice struct {
	FinishReason string  `json:"finish_reason"`
	Index        int     `json:"index"`
	Message      Message `json:"message"`
}

type Usage struct {
	CompletionTokens int `json:"completion_tokens"`
	PromptTokens     int `json:"prompt_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type StreamResponse struct {
	ID      string         `json:"id"`
	Choices []StreamChoice `json:"choices"`
}

type StreamChoice struct {
	Index        int          `json:"index"`
	Delta        StreamDelta  `json:"delta"`
	FinishReason string       `json:"finish_reason,omitempty"`
}

type StreamDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

type SearchResult struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Summary     string `json:"summary"`
	SiteName    string `json:"site_name"`
	PublishTime string `json:"publish_time"`
}

type Annotation struct {
	Type        string `json:"type"`
	URL         string `json:"url,omitempty"`
	Title       string `json:"title,omitempty"`
	Summary     string `json:"summary,omitempty"`
	SiteName    string `json:"site_name,omitempty"`
	PublishTime string `json:"publish_time,omitempty"`
}

type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

type ToolHandler func(args string) (string, error)

type ClientOption func(*Client)

func WithTemperature(t float64) ClientOption {
	return func(c *Client) {
		c.temperature = t
	}
}

func WithTopP(p float64) ClientOption {
	return func(c *Client) {
		c.topP = p
	}
}

func WithSearch(enabled bool) ClientOption {
	return func(c *Client) {
		c.enableSearch = enabled
	}
}

func WithThinking(enabled bool) ClientOption {
	return func(c *Client) {
		c.enableThinking = enabled
	}
}

func WithJinaSearch() ClientOption {
	return func(c *Client) {
		c.tools = append(c.tools, Tool{
			Type: "function",
			Function: Function{
				Name:        "jina_search",
				Description: "Search the web for current information. Use this when you need up-to-date facts, news, or information that may not be in your training data.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "The search query",
						},
					},
					"required": []string{"query"},
				},
			},
		})
	}
}

func WithFetchURL() ClientOption {
	return func(c *Client) {
		c.tools = append(c.tools, Tool{
			Type: "function",
			Function: Function{
				Name:        "fetch_url",
				Description: "Fetch and read the content of a URL. Use this when a user shares a URL and you need to read its content.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"url": map[string]interface{}{
							"type":        "string",
							"description": "The URL to fetch",
						},
					},
					"required": []string{"url"},
				},
			},
		})
	}
}

func NewClient(endpoint, apiKey, model string, maxTokens int, opts ...ClientOption) *Client {
	c := &Client{
		endpoint:       strings.TrimRight(endpoint, "/"),
		apiKey:         apiKey,
		model:          model,
		maxTokens:      maxTokens,
		temperature:    1.0,
		topP:           0.95,
		enableSearch:   false,
		enableThinking: false,
		tools:          []Tool{},
		httpClient:     &http.Client{Timeout: 120 * time.Second},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (c *Client) buildRequest(messages []Message, stream bool) ChatRequest {
	req := ChatRequest{
		Model:               c.model,
		Messages:            messages,
		MaxCompletionTokens: c.maxTokens,
		Temperature:         c.temperature,
		TopP:                c.topP,
		Stream:              stream,
		FrequencyPenalty:    0,
		PresencePenalty:     0,
	}

	if c.enableSearch {
		req.Tools = append(req.Tools, Tool{
			Type:        "web_search",
			MaxKeyword:  3,
			ForceSearch: false,
			Limit:       3,
		})
	}

	req.Tools = append(req.Tools, c.tools...)

	if c.enableThinking {
		req.Thinking = &ThinkingConfig{Type: "enabled"}
	} else {
		req.Thinking = &ThinkingConfig{Type: "disabled"}
	}

	return req
}

func (c *Client) Chat(systemPrompt string, history []Message, userMessage string) (string, error) {
	messages := []Message{
		{Role: "system", Content: systemPrompt},
	}
	messages = append(messages, history...)
	messages = append(messages, Message{Role: "user", Content: userMessage})

	return c.chatWithMessages(messages)
}

func (c *Client) ChatWithTools(systemPrompt string, history []Message, userMessage string, handlers map[string]ToolHandler) (string, error) {
	messages := []Message{
		{Role: "system", Content: systemPrompt},
	}
	messages = append(messages, history...)
	messages = append(messages, Message{Role: "user", Content: userMessage})

	for i := 0; i < 5; i++ {
		resp, err := c.chatWithMessages(messages)
		if err != nil {
			return "", err
		}

		reqBody := c.buildRequest(messages, false)
		jsonData, _ := json.Marshal(reqBody)
		
		apiURL := c.endpoint + "/chat/completions"
		req, _ := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		httpResp, err := c.httpClient.Do(req)
		if err != nil {
			return resp, nil
		}
		defer httpResp.Body.Close()

		body, _ := io.ReadAll(httpResp.Body)
		var chatResp ChatResponse
		if err := json.Unmarshal(body, &chatResp); err != nil {
			return resp, nil
		}

		if len(chatResp.Choices) == 0 {
			return resp, nil
		}

		msg := chatResp.Choices[0].Message
		if len(msg.ToolCalls) == 0 {
			return msg.Content, nil
		}

		messages = append(messages, msg)

		for _, tc := range msg.ToolCalls {
			handler, ok := handlers[tc.Function.Name]
			if !ok {
				messages = append(messages, Message{
					Role:       "tool",
					ToolCallID: tc.ID,
					Content:    "Tool not available",
				})
				continue
			}

			result, err := handler(tc.Function.Arguments)
			if err != nil {
				result = fmt.Sprintf("Error: %v", err)
			}

			messages = append(messages, Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    result,
			})
		}
	}

	return "", fmt.Errorf("max tool call iterations reached")
}

func (c *Client) chatWithMessages(messages []Message) (string, error) {
	reqBody := c.buildRequest(messages, false)

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	logger.Info("LLM Request: %s", string(jsonData))

	apiURL := c.endpoint + "/chat/completions"
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	logger.Info("LLM Response (status %d): %s", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err != nil {
			return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
		}
		return "", fmt.Errorf("API error: %s (code: %s)", errResp.Error.Message, errResp.Error.Code)
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}

func (c *Client) ChatStream(systemPrompt string, history []Message, userMessage string, callback func(chunk string)) error {
	messages := []Message{
		{Role: "system", Content: systemPrompt},
	}
	messages = append(messages, history...)
	messages = append(messages, Message{Role: "user", Content: userMessage})

	reqBody := c.buildRequest(messages, true)

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	apiURL := c.endpoint + "/chat/completions"
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err != nil {
			return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
		}
		return fmt.Errorf("API error: %s (code: %s)", errResp.Error.Message, errResp.Error.Code)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var streamResp StreamResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			continue
		}

		if len(streamResp.Choices) > 0 && streamResp.Choices[0].Delta.Content != "" {
			callback(streamResp.Choices[0].Delta.Content)
		}
	}

	return scanner.Err()
}
