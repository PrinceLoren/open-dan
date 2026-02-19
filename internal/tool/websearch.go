package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// WebSearchTool provides web search capability using DuckDuckGo HTML.
type WebSearchTool struct{}

func NewWebSearchTool() *WebSearchTool {
	return &WebSearchTool{}
}

func (t *WebSearchTool) Name() string        { return "web_search" }
func (t *WebSearchTool) Description() string  {
	return "Search the web for information. Returns search results with titles and URLs."
}

func (t *WebSearchTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "The search query"
			}
		},
		"required": ["query"]
	}`)
}

func (t *WebSearchTool) Execute(ctx context.Context, args json.RawMessage) (*Result, error) {
	var params struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return &Result{Error: "invalid arguments: " + err.Error(), IsError: true}, nil
	}

	if params.Query == "" {
		return &Result{Error: "query is required", IsError: true}, nil
	}

	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(params.Query))

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return &Result{Error: "failed to create request: " + err.Error(), IsError: true}, nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; OpenDan/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return &Result{Error: "search request failed: " + err.Error(), IsError: true}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 100000))
	if err != nil {
		return &Result{Error: "failed to read response: " + err.Error(), IsError: true}, nil
	}

	// Return raw HTML for the LLM to parse — simple and effective
	output := string(body)
	if len(output) > 10000 {
		output = output[:10000] + "\n... (truncated)"
	}

	return &Result{Output: output}, nil
}
