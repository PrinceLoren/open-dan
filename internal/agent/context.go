package agent

import (
	"context"

	"open-dan/internal/llm"
)

// contextManager handles conversation context, including summarization
// when the context window approaches its limit.
type contextManager struct {
	provider      llm.Provider
	contextWindow int
	summarizeAt   int
}

func newContextManager(provider llm.Provider, contextWindow, summarizeAt int) *contextManager {
	return &contextManager{
		provider:      provider,
		contextWindow: contextWindow,
		summarizeAt:   summarizeAt,
	}
}

// estimateTokens provides a rough token estimate (4 chars ≈ 1 token).
func estimateTokens(messages []llm.Message) int {
	total := 0
	for _, m := range messages {
		total += len(m.Content) / 4
		for _, tc := range m.ToolCalls {
			total += len(tc.Arguments) / 4
		}
	}
	return total
}

// shouldSummarize returns true if the message history approaches the context limit.
func (cm *contextManager) shouldSummarize(messages []llm.Message) bool {
	return estimateTokens(messages) > cm.summarizeAt
}

// summarize compresses the conversation into a summary + recent messages.
func (cm *contextManager) summarize(ctx context.Context, messages []llm.Message) (string, []llm.Message, error) {
	if len(messages) <= 4 {
		return "", messages, nil
	}

	// Keep last 4 messages as recent context
	cutoff := len(messages) - 4
	toSummarize := messages[:cutoff]
	recent := messages[cutoff:]

	// Build summarization prompt
	var text string
	for _, m := range toSummarize {
		text += m.Role + ": " + m.Content + "\n"
	}

	summaryReq := &llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "user", Content: "Summarize this conversation concisely, preserving key facts, decisions, and context:\n\n" + text},
		},
		MaxTokens:    1024,
		Temperature:  0.3,
		SystemPrompt: "You are a conversation summarizer. Create a brief, factual summary.",
	}

	resp, err := cm.provider.Chat(ctx, summaryReq)
	if err != nil {
		// If summarization fails, just truncate
		return "", recent, nil
	}

	return resp.Content, recent, nil
}
