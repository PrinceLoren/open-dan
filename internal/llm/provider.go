package llm

import "context"

// Provider is the interface all LLM backends must implement.
type Provider interface {
	// Chat sends a chat completion request and returns the full response.
	Chat(ctx context.Context, req *ChatRequest) (*LLMResponse, error)

	// StreamChat sends a streaming chat completion request.
	StreamChat(ctx context.Context, req *ChatRequest) (<-chan StreamEvent, error)

	// Name returns the provider name (e.g. "openai", "anthropic").
	Name() string

	// DefaultModel returns the default model for this provider.
	DefaultModel() string
}

// LLMError wraps an error with a classification for fallback logic.
type LLMError struct {
	Type    ErrorType
	Message string
	Err     error
}

func (e *LLMError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *LLMError) Unwrap() error {
	return e.Err
}
