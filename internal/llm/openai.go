package llm

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// OpenAIProvider implements Provider using the OpenAI API.
// Also works with compatible APIs (Ollama, LM Studio, vLLM) via BaseURL.
type OpenAIProvider struct {
	client       openai.Client
	defaultModel string
}

// OpenAIConfig holds configuration for the OpenAI provider.
type OpenAIConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

// NewOpenAIProvider creates a new OpenAI provider.
func NewOpenAIProvider(cfg OpenAIConfig) *OpenAIProvider {
	opts := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey),
	}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}

	model := cfg.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	return &OpenAIProvider{
		client:       openai.NewClient(opts...),
		defaultModel: model,
	}
}

func (p *OpenAIProvider) Name() string        { return "openai" }
func (p *OpenAIProvider) DefaultModel() string { return p.defaultModel }

func (p *OpenAIProvider) Chat(ctx context.Context, req *ChatRequest) (*LLMResponse, error) {
	model := req.Model
	if model == "" {
		model = p.defaultModel
	}

	messages := p.convertMessages(req)
	tools := p.convertTools(req.Tools)

	params := openai.ChatCompletionNewParams{
		Model:    model,
		Messages: messages,
	}
	if req.MaxTokens > 0 {
		params.MaxTokens = openai.Int(int64(req.MaxTokens))
	}
	if req.Temperature > 0 {
		params.Temperature = openai.Float(req.Temperature)
	}
	if len(tools) > 0 {
		params.Tools = tools
	}

	resp, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, classifyOpenAIError(err)
	}

	return p.convertResponse(resp), nil
}

func (p *OpenAIProvider) StreamChat(ctx context.Context, req *ChatRequest) (<-chan StreamEvent, error) {
	model := req.Model
	if model == "" {
		model = p.defaultModel
	}

	messages := p.convertMessages(req)
	tools := p.convertTools(req.Tools)

	params := openai.ChatCompletionNewParams{
		Model:    model,
		Messages: messages,
	}
	if req.MaxTokens > 0 {
		params.MaxTokens = openai.Int(int64(req.MaxTokens))
	}
	if req.Temperature > 0 {
		params.Temperature = openai.Float(req.Temperature)
	}
	if len(tools) > 0 {
		params.Tools = tools
	}

	stream := p.client.Chat.Completions.NewStreaming(ctx, params)
	ch := make(chan StreamEvent, 64)

	go func() {
		defer close(ch)
		for stream.Next() {
			chunk := stream.Current()
			evt := StreamEvent{}
			if len(chunk.Choices) > 0 {
				delta := chunk.Choices[0].Delta
				evt.ContentDelta = delta.Content
				if chunk.Choices[0].FinishReason != "" {
					evt.Done = true
				}
			}
			if chunk.Usage.TotalTokens > 0 {
				evt.Usage = &Usage{
					InputTokens:  int(chunk.Usage.PromptTokens),
					OutputTokens: int(chunk.Usage.CompletionTokens),
				}
			}
			ch <- evt
		}
		if err := stream.Err(); err != nil {
			ch <- StreamEvent{Error: classifyOpenAIError(err), Done: true}
		}
	}()

	return ch, nil
}

func (p *OpenAIProvider) convertMessages(req *ChatRequest) []openai.ChatCompletionMessageParamUnion {
	var msgs []openai.ChatCompletionMessageParamUnion

	if req.SystemPrompt != "" {
		msgs = append(msgs, openai.SystemMessage(req.SystemPrompt))
	}

	for _, m := range req.Messages {
		switch m.Role {
		case "system":
			msgs = append(msgs, openai.SystemMessage(m.Content))
		case "user":
			msgs = append(msgs, openai.UserMessage(m.Content))
		case "assistant":
			if len(m.ToolCalls) > 0 {
				toolCalls := make([]openai.ChatCompletionMessageToolCallParam, len(m.ToolCalls))
				for i, tc := range m.ToolCalls {
					toolCalls[i] = openai.ChatCompletionMessageToolCallParam{
						ID: tc.ID,
						Function: openai.ChatCompletionMessageToolCallFunctionParam{
							Name:      tc.Name,
							Arguments: string(tc.Arguments),
						},
					}
				}
				asst := openai.ChatCompletionAssistantMessageParam{
					ToolCalls: toolCalls,
				}
				if m.Content != "" {
					asst.Content.OfString = openai.String(m.Content)
				}
				msgs = append(msgs, openai.ChatCompletionMessageParamUnion{OfAssistant: &asst})
			} else {
				msgs = append(msgs, openai.AssistantMessage(m.Content))
			}
		case "tool":
			msgs = append(msgs, openai.ToolMessage(m.Content, m.ToolCallID))
		}
	}
	return msgs
}

func (p *OpenAIProvider) convertTools(tools []ToolDefinition) []openai.ChatCompletionToolParam {
	if len(tools) == 0 {
		return nil
	}
	result := make([]openai.ChatCompletionToolParam, len(tools))
	for i, t := range tools {
		var params map[string]interface{}
		if t.Parameters != nil {
			_ = json.Unmarshal(t.Parameters, &params)
		}
		result[i] = openai.ChatCompletionToolParam{
			Function: openai.FunctionDefinitionParam{
				Name:        t.Name,
				Description: openai.String(t.Description),
				Parameters:  openai.FunctionParameters(params),
			},
		}
	}
	return result
}

func (p *OpenAIProvider) convertResponse(resp *openai.ChatCompletion) *LLMResponse {
	result := &LLMResponse{
		Usage: Usage{
			InputTokens:  int(resp.Usage.PromptTokens),
			OutputTokens: int(resp.Usage.CompletionTokens),
		},
	}

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		result.Content = choice.Message.Content
		result.StopReason = string(choice.FinishReason)

		for _, tc := range choice.Message.ToolCalls {
			result.ToolCalls = append(result.ToolCalls, ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: json.RawMessage(tc.Function.Arguments),
			})
		}
	}

	return result
}

func classifyOpenAIError(err error) *LLMError {
	msg := err.Error()
	lower := strings.ToLower(msg)
	llmErr := &LLMError{Err: err, Message: msg}

	switch {
	case strings.Contains(lower, "401") || strings.Contains(lower, "403") || strings.Contains(lower, "unauthorized"):
		llmErr.Type = ErrorAuth
	case strings.Contains(lower, "429") || strings.Contains(lower, "rate limit"):
		llmErr.Type = ErrorRateLimit
	case strings.Contains(lower, "400") || strings.Contains(lower, "invalid"):
		llmErr.Type = ErrorInvalidInput
	case strings.Contains(lower, "500") || strings.Contains(lower, "502") || strings.Contains(lower, "503"):
		llmErr.Type = ErrorServerError
	case strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline"):
		llmErr.Type = ErrorTimeout
	case strings.Contains(lower, "connection") || strings.Contains(lower, "dns") || strings.Contains(lower, "refused"):
		llmErr.Type = ErrorNetwork
	default:
		llmErr.Type = ErrorUnknown
	}
	return llmErr
}
