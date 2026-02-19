package memory

import (
	"context"
	"path/filepath"
	"testing"

	"open-dan/internal/llm"
)

func newTestMemory(t *testing.T) *SQLiteMemory {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	mem, err := NewSQLiteMemory(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { mem.Close() })
	return mem
}

func TestSaveAndGetMessages(t *testing.T) {
	mem := newTestMemory(t)
	ctx := context.Background()

	msgs := []llm.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
		{Role: "user", Content: "How are you?"},
	}

	for _, m := range msgs {
		if err := mem.SaveMessage(ctx, "chat1", m); err != nil {
			t.Fatal(err)
		}
	}

	history, err := mem.GetHistory(ctx, "chat1", 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(history) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(history))
	}
	if history[0].Content != "Hello" {
		t.Fatalf("expected 'Hello', got %q", history[0].Content)
	}
	if history[2].Content != "How are you?" {
		t.Fatalf("expected 'How are you?', got %q", history[2].Content)
	}
}

func TestGetHistoryLimit(t *testing.T) {
	mem := newTestMemory(t)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		mem.SaveMessage(ctx, "chat1", llm.Message{Role: "user", Content: "msg"})
	}

	history, err := mem.GetHistory(ctx, "chat1", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(history))
	}
}

func TestSaveAndGetSummary(t *testing.T) {
	mem := newTestMemory(t)
	ctx := context.Background()

	if err := mem.SaveSummary(ctx, "chat1", "User asked about weather"); err != nil {
		t.Fatal(err)
	}

	summary, err := mem.GetSummary(ctx, "chat1")
	if err != nil {
		t.Fatal(err)
	}
	if summary != "User asked about weather" {
		t.Fatalf("expected summary, got %q", summary)
	}
}

func TestGetSummaryEmpty(t *testing.T) {
	mem := newTestMemory(t)
	ctx := context.Background()

	summary, err := mem.GetSummary(ctx, "nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if summary != "" {
		t.Fatalf("expected empty summary, got %q", summary)
	}
}

func TestIsolatedChats(t *testing.T) {
	mem := newTestMemory(t)
	ctx := context.Background()

	mem.SaveMessage(ctx, "chat1", llm.Message{Role: "user", Content: "chat1 msg"})
	mem.SaveMessage(ctx, "chat2", llm.Message{Role: "user", Content: "chat2 msg"})

	h1, _ := mem.GetHistory(ctx, "chat1", 10)
	h2, _ := mem.GetHistory(ctx, "chat2", 10)

	if len(h1) != 1 || h1[0].Content != "chat1 msg" {
		t.Fatal("chat1 history incorrect")
	}
	if len(h2) != 1 || h2[0].Content != "chat2 msg" {
		t.Fatal("chat2 history incorrect")
	}
}
