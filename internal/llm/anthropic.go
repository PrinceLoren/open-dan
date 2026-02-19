package llm

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// AnthropicProvider implements Provider using the Anthropic API.
type AnthropicProvider struct {
	client       anthropic.Client
	defaultModel string
}

// AnthropicConfig holds configuration for the Anthropic provider.
type AnthropicConfig struct {
	APIKey string
	Model  string
}

// NewAnthropicProvider creates a new Anthropic provider.
func NewAnthropicProvider(cfg AnthropicConfig) *AnthropicProvider {
	model := cfg.Model
	if model == "" {
		model = "claude-sonnet-4-5-20250514"
	}
	return &AnthropicProvider{
		client:       anthropic.NewClient(option.WithAPIKey(cfg.APIKey)),
		defaultModel: model,
	}
}

func (p *AnthropicProvider) Name() string        { return "anthropic" }
func (p *AnthropicProvider) DefaultModel() string { return p.defaultModel }

func (p *AnthropicProvider) Chat(ctx context.Context, req *ChatRequest) (*LLMResponse, error) {
	model := req.Model
	if model == "" {
		model = p.defaultModel
	}

	messages := p.convertMessages(req)
	tools := p.convertTools(req.Tools)

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		Messages:  messages,
		MaxTokens: int64(req.MaxTokens),
	}
	if req.SystemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: req.SystemPrompt},
		}
	}
	if req.Temperature > 0 {
		params.Temperature = anthropic.Float(req.Temperature)
	}
	if len(tools) > 0 {
		params.Tools = tools
	}

	resp, err := p.client.Messages.New(ctx, params)
	if err != nil {
		return nil, classifyAnthropicError(err)
	}

	return p.convertResponse(resp), nil
}

func (p *AnthropicProvider) StreamChat(ctx context.Context, req *ChatRequest) (<-chan StreamEvent, error) {
	model := req.Model
	if model == "" {
		model = p.defaultModel
	}

	messages := p.convertMessages(req)
	tools := p.convertTools(req.Tools)

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		Messages:  messages,
		MaxTokens: int64(req.MaxTokens),
	}
	if req.SystemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: req.SystemPrompt},
		}
	}
	if req.Temperature > 0 {
		params.Temperature = anthropic.Float(req.Temperature)
	}
	if len(tools) > 0 {
		params.Tools = tools
	}

	stream := p.client.Messages.NewStreaming(ctx, params)
	ch := make(chan StreamEvent, 64)

	go func() {
		defer close(ch)
		for stream.Next() {
			event := stream.Current()
			evt := StreamEvent{}
			switch e := event.AsAny().(type) {
			case anthropic.ContentBlockDeltaEvent:
				if e.Delta.Type == "text_delta" {
					evt.ContentDelta = e.Delta.Text
				}
			case anthropic.MessageDeltaEvent:
				evt.Done = true
				if e.Usage.OutputTokens > 0 {
					evt.Usage = &Usage{OutputTokens: int(e.Usage.OutputTokens)}
				}
			}
			ch <- evt
		}
		if err := stream.Err(); err != nil {
			ch <- StreamEvent{Error: classifyAnthropicError(err), Done: true}
		}
	}()

	return ch, nil
}

func (p *AnthropicProvider) convertMessages(req *ChatRequest) []anthropic.MessageParam {
	var msgs []anthropic.MessageParam

	for _, m := range req.Messages {
		switch m.Role {
		case "user":
			msgs = append(msgs, anthropic.NewUserMessage(
				anthropic.NewTextBlock(m.Content),
			))
		case "assistant":
			if len(m.ToolCalls) > 0 {
				var blocks []anthropic.ContentBlockParamUnion
				if m.Content != "" {
					blocks = append(blocks, anthropic.NewTextBlock(m.Content))
				}
				for _, tc := range m.ToolCalls {
					var input map[string]any
					_ = json.Unmarshal(tc.Arguments, &input)
					blocks = append(blocks, anthropic.NewToolUseBlock(tc.ID, input, tc.Name))
				}
				msgs = append(msgs, anthropic.NewAssistantMessage(blocks...))
			} else {
				msgs = append(msgs, anthropic.NewAssistantMessage(
					anthropic.NewTextBlock(m.Content),
				))
			}
		case "tool":
			msgs = append(msgs, anthropic.NewUserMessage(
				anthropic.NewToolResultBlock(m.ToolCallID, m.Content, false),
			))
		}
	}
	return msgs
}

func (p *AnthropicProvider) convertTools(tools []ToolDefinition) []anthropic.ToolUnionParam {
	if len(tools) == 0 {
		return nil
	}
	result := make([]anthropic.ToolUnionParam, len(tools))
	for i, t := range tools {
		var schema anthropic.ToolInputSchemaParam
		if t.Parameters != nil {
			_ = json.Unmarshal(t.Parameters, &schema)
		}
		result[i] = anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        t.Name,
				Description: anthropic.String(t.Description),
				InputSchema: schema,
			},
		}
	}
	return result
}

func (p *AnthropicProvider) convertResponse(resp *anthropic.Message) *LLMResponse {
	result := &LLMResponse{
		StopReason: string(resp.StopReason),
		Usage: Usage{
			InputTokens:  int(resp.Usage.InputTokens),
			OutputTokens: int(resp.Usage.OutputTokens),
		},
	}

	for _, block := range resp.Content {
		switch b := block.AsAny().(type) {
		case anthropic.TextBlock:
			result.Content += b.Text
		case anthropic.ToolUseBlock:
			args, _ := json.Marshal(b.Input)
			result.ToolCalls = append(result.ToolCalls, ToolCall{
				ID:        b.ID,
				Name:      b.Name,
				Arguments: args,
			})
		}
	}

	return result
}

func classifyAnthropicError(err error) *LLMError {
	msg := err.Error()
	lower := strings.ToLower(msg)
	llmErr := &LLMError{Err: err, Message: msg}

	switch {
	case strings.Contains(lower, "401") || strings.Contains(lower, "authentication"):
		llmErr.Type = ErrorAuth
	case strings.Contains(lower, "429") || strings.Contains(lower, "rate_limit"):
		llmErr.Type = ErrorRateLimit
	case strings.Contains(lower, "400") || strings.Contains(lower, "invalid_request"):
		llmErr.Type = ErrorInvalidInput
	case strings.Contains(lower, "500") || strings.Contains(lower, "overloaded"):
		llmErr.Type = ErrorServerError
	case strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline"):
		llmErr.Type = ErrorTimeout
	default:
		llmErr.Type = ErrorUnknown
	}
	return llmErr
}
