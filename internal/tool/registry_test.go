package tool

import (
	"context"
	"encoding/json"
	"testing"
)

// mockTool is a simple tool for testing.
type mockTool struct {
	name string
}

func (m *mockTool) Name() string        { return m.name }
func (m *mockTool) Description() string { return "test tool" }
func (m *mockTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{}}`)
}
func (m *mockTool) Execute(ctx context.Context, args json.RawMessage) (*Result, error) {
	return &Result{Output: "executed " + m.name}, nil
}

func TestRegistryRegisterAndGet(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockTool{name: "test1"})
	r.Register(&mockTool{name: "test2"})

	tool, err := r.Get("test1")
	if err != nil {
		t.Fatal(err)
	}
	if tool.Name() != "test1" {
		t.Fatalf("expected test1, got %s", tool.Name())
	}

	_, err = r.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent tool")
	}
}

func TestRegistryList(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockTool{name: "a"})
	r.Register(&mockTool{name: "b"})

	tools := r.List()
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
}

func TestRegistryDefinitions(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockTool{name: "shell"})

	defs := r.Definitions()
	if len(defs) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(defs))
	}
	if defs[0].Name != "shell" {
		t.Fatalf("expected 'shell', got %s", defs[0].Name)
	}
}
