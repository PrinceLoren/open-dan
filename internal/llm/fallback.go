package llm

import (
	"context"
	"errors"
	"log"
)

// FallbackProvider tries providers in order, falling back on retryable errors.
type FallbackProvider struct {
	providers []Provider
}

// NewFallbackProvider creates a provider chain. The first provider is primary.
func NewFallbackProvider(providers ...Provider) *FallbackProvider {
	return &FallbackProvider{providers: providers}
}

func (f *FallbackProvider) Name() string {
	if len(f.providers) > 0 {
		return f.providers[0].Name() + "+fallback"
	}
	return "fallback"
}

func (f *FallbackProvider) DefaultModel() string {
	if len(f.providers) > 0 {
		return f.providers[0].DefaultModel()
	}
	return ""
}

func (f *FallbackProvider) Chat(ctx context.Context, req *ChatRequest) (*LLMResponse, error) {
	var lastErr error
	for _, p := range f.providers {
		resp, err := p.Chat(ctx, req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !isRetryable(err) {
			return nil, err
		}
		log.Printf("[fallback] provider %s failed: %v, trying next", p.Name(), err)
	}
	return nil, lastErr
}

func (f *FallbackProvider) StreamChat(ctx context.Context, req *ChatRequest) (<-chan StreamEvent, error) {
	var lastErr error
	for _, p := range f.providers {
		ch, err := p.StreamChat(ctx, req)
		if err == nil {
			return ch, nil
		}
		lastErr = err
		if !isRetryable(err) {
			return nil, err
		}
		log.Printf("[fallback] provider %s stream failed: %v, trying next", p.Name(), err)
	}
	return nil, lastErr
}

// isRetryable returns true for errors that warrant trying a different provider.
func isRetryable(err error) bool {
	var llmErr *LLMError
	if !errors.As(err, &llmErr) {
		return true // unknown errors are retryable
	}
	switch llmErr.Type {
	case ErrorAuth, ErrorInvalidInput:
		return false // these won't succeed on retry
	case ErrorRateLimit, ErrorServerError, ErrorTimeout, ErrorNetwork:
		return true
	default:
		return true
	}
}
