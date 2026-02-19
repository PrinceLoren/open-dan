package llm

import "encoding/json"

// Message represents a chat message.
type Message struct {
	Role       string     `json:"role"` // "system", "user", "assistant", "tool"
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolDefinition describes a tool available to the LLM.
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"` // JSON Schema
}

// ToolCall represents an LLM request to invoke a tool.
type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// LLMResponse is the response from an LLM provider.
type LLMResponse struct {
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	Usage      Usage      `json:"usage"`
	StopReason string     `json:"stop_reason"`
}

// Usage tracks token consumption.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ChatRequest is the input for a chat completion.
type ChatRequest struct {
	Model        string           `json:"model"`
	Messages     []Message        `json:"messages"`
	Tools        []ToolDefinition `json:"tools,omitempty"`
	MaxTokens    int              `json:"max_tokens"`
	Temperature  float64          `json:"temperature"`
	SystemPrompt string           `json:"system_prompt,omitempty"`
}

// StreamEvent represents a chunk in a streaming response.
type StreamEvent struct {
	ContentDelta string     `json:"content_delta,omitempty"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	Usage        *Usage     `json:"usage,omitempty"`
	Done         bool       `json:"done"`
	Error        error      `json:"-"`
}

// ErrorType classifies LLM errors for fallback decisions.
type ErrorType int

const (
	ErrorUnknown       ErrorType = iota
	ErrorRateLimit               // 429
	ErrorAuth                    // 401/403
	ErrorInvalidInput            // 400
	ErrorServerError             // 500+
	ErrorTimeout                 // context deadline exceeded
	ErrorNetwork                 // connection refused, DNS, etc.
)
