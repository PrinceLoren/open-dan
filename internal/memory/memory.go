package memory

import (
	"context"

	"open-dan/internal/llm"
)

// Memory is the interface for persistent conversation storage.
type Memory interface {
	SaveMessage(ctx context.Context, chatID string, msg llm.Message) error
	GetHistory(ctx context.Context, chatID string, limit int) ([]llm.Message, error)
	SaveSummary(ctx context.Context, chatID string, summary string) error
	GetSummary(ctx context.Context, chatID string) (string, error)
	Close() error
}
