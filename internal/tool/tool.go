package tool

import (
	"context"
	"encoding/json"
)

// Tool is the interface for agent tools.
type Tool interface {
	Name() string
	Description() string
	Parameters() json.RawMessage // JSON Schema
	Execute(ctx context.Context, args json.RawMessage) (*Result, error)
}

// Result is the output of a tool execution.
type Result struct {
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
	IsError bool   `json:"is_error"`
}
