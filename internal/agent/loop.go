package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"open-dan/internal/llm"
)

// processMessage runs the agent loop for a single user message.
// Loop: think → act → observe, repeating until the LLM produces a final text response.
func (a *Agent) processMessage(ctx context.Context, chatID, userText string) (string, error) {
	// Load history from memory
	history, err := a.memory.GetHistory(ctx, chatID, 50)
	if err != nil {
		log.Printf("[agent] failed to load history: %v", err)
		history = nil
	}

	// Check for existing summary
	summary, _ := a.memory.GetSummary(ctx, chatID)

	// Build messages
	messages := make([]llm.Message, 0, len(history)+2)

	if summary != "" {
		messages = append(messages, llm.Message{
			Role:    "user",
			Content: "[Previous conversation summary]: " + summary,
		})
		messages = append(messages, llm.Message{
			Role:    "assistant",
			Content: "I understand the previous context. How can I help?",
		})
	}

	messages = append(messages, history...)
	messages = append(messages, llm.Message{Role: "user", Content: userText})

	// Save user message
	_ = a.memory.SaveMessage(ctx, chatID, llm.Message{Role: "user", Content: userText})

	// Agent loop
	toolCallCount := 0
	for {
		// Check context window, summarize if needed
		if a.ctxManager.shouldSummarize(messages) {
			newSummary, recent, err := a.ctxManager.summarize(ctx, messages)
			if err == nil && newSummary != "" {
				_ = a.memory.SaveSummary(ctx, chatID, newSummary)
				messages = append([]llm.Message{
					{Role: "user", Content: "[Conversation summary]: " + newSummary},
					{Role: "assistant", Content: "I understand the context. Continuing..."},
				}, recent...)
			}
		}

		// Think: send to LLM
		req := &llm.ChatRequest{
			Messages:     messages,
			Tools:        a.tools.Definitions(),
			MaxTokens:    a.cfg.MaxTokens,
			Temperature:  a.cfg.Temperature,
			SystemPrompt: a.cfg.SystemPrompt,
		}

		a.bus.Publish("llm_request", req)

		resp, err := a.provider.Chat(ctx, req)
		if err != nil {
			return "", fmt.Errorf("LLM error: %w", err)
		}

		a.bus.Publish("llm_response", resp)

		// If no tool calls, we have the final response
		if len(resp.ToolCalls) == 0 {
			_ = a.memory.SaveMessage(ctx, chatID, llm.Message{Role: "assistant", Content: resp.Content})
			return resp.Content, nil
		}

		// Guard against infinite tool call loops
		toolCallCount += len(resp.ToolCalls)
		if toolCallCount > a.cfg.MaxToolCalls {
			msg := "I've reached the maximum number of tool calls for this request. Here's what I have so far: " + resp.Content
			_ = a.memory.SaveMessage(ctx, chatID, llm.Message{Role: "assistant", Content: msg})
			return msg, nil
		}

		// Record assistant message with tool calls
		assistantMsg := llm.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		}
		messages = append(messages, assistantMsg)

		// Act: execute each tool call
		for _, tc := range resp.ToolCalls {
			a.bus.Publish("tool_call", tc)

			t, err := a.tools.Get(tc.Name)
			var result string
			if err != nil {
				result = fmt.Sprintf("Error: tool '%s' not found", tc.Name)
			} else {
				res, err := t.Execute(ctx, tc.Arguments)
				if err != nil {
					result = "Error executing tool: " + err.Error()
				} else if res.IsError {
					result = "Error: " + res.Error
				} else {
					result = res.Output
				}
			}

			a.bus.Publish("tool_result", map[string]string{"id": tc.ID, "result": result})

			// Observe: add tool result to messages
			toolMsg := llm.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			}
			messages = append(messages, toolMsg)
		}
	}
}

// TestConnection sends a simple message to verify the LLM provider works.
func (a *Agent) TestConnection(ctx context.Context) error {
	req := &llm.ChatRequest{
		Messages:  []llm.Message{{Role: "user", Content: "Say 'OK' if you can hear me."}},
		MaxTokens: 32,
	}
	_, err := a.provider.Chat(ctx, req)
	return err
}

// SetProvider replaces the LLM provider (e.g., after config change).
func (a *Agent) SetProvider(p llm.Provider) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.provider = p
	a.ctxManager = newContextManager(p, a.cfg.ContextWindow, a.cfg.SummarizeAt)
}

// ProcessingResult is returned to the caller with the response.
type ProcessingResult struct {
	Response string
	Error    string
}

// MarshalJSON implements json.Marshaler for ProcessingResult.
func (r ProcessingResult) MarshalJSON() ([]byte, error) {
	type alias ProcessingResult
	return json.Marshal(alias(r))
}
