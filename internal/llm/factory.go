package llm

import (
	"fmt"

	"open-dan/internal/config"
)

// NewProvider creates an LLM provider from config.
func NewProvider(cfg config.LLMConfig) (Provider, error) {
	switch cfg.Provider {
	case "openai", "openrouter", "local":
		return NewOpenAIProvider(OpenAIConfig{
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
		}), nil
	case "anthropic":
		return NewAnthropicProvider(AnthropicConfig{
			APIKey: cfg.APIKey,
			Model:  cfg.Model,
		}), nil
	default:
		return nil, fmt.Errorf("unknown LLM provider: %s", cfg.Provider)
	}
}
