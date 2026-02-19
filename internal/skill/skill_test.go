package skill

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestManifestParsing(t *testing.T) {
	// Valid manifest
	dir := t.TempDir()
	valid := `{
		"name": "test_skill",
		"version": "1.0.0",
		"description": "A test skill",
		"parameters": {"type": "object", "properties": {"input": {"type": "string"}}},
		"command": "echo hello"
	}`
	os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(valid), 0644)

	m, err := parseManifest(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Name != "test_skill" {
		t.Fatalf("expected 'test_skill', got %s", m.Name)
	}
	if m.Version != "1.0.0" {
		t.Fatalf("expected '1.0.0', got %s", m.Version)
	}

	// Invalid JSON
	badDir := t.TempDir()
	os.WriteFile(filepath.Join(badDir, "manifest.json"), []byte("{invalid"), 0644)
	_, err = parseManifest(filepath.Join(badDir, "manifest.json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}

	// Missing required fields
	emptyDir := t.TempDir()
	os.WriteFile(filepath.Join(emptyDir, "manifest.json"), []byte(`{"name":"","command":""}`), 0644)
	_, err = parseManifest(filepath.Join(emptyDir, "manifest.json"))
	if err == nil {
		t.Fatal("expected error for missing fields")
	}
}

func TestSkillToolExecute(t *testing.T) {
	dir := t.TempDir()

	// Create a simple echo skill
	manifest := Manifest{
		Name:        "echo_test",
		Version:     "1.0.0",
		Description: "Echoes input back",
		Command:     "cat",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"message":{"type":"string"}}}`),
	}

	st := NewSkillTool(manifest, dir, 10, false)

	if st.Name() != "skill_echo_test" {
		t.Fatalf("expected 'skill_echo_test', got %s", st.Name())
	}

	args := json.RawMessage(`{"message":"hello world"}`)
	result, err := st.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", result.Error)
	}
	if !strings.Contains(result.Output, "hello world") {
		t.Fatalf("expected output to contain 'hello world', got: %s", result.Output)
	}
}

func TestLoaderLoadAll(t *testing.T) {
	dir := t.TempDir()

	// Create two skill directories
	for _, name := range []string{"skill_a", "skill_b"} {
		skillDir := filepath.Join(dir, name)
		os.MkdirAll(skillDir, 0755)
		manifest := Manifest{
			Name:    name,
			Version: "1.0.0",
			Command: "echo ok",
		}
		data, _ := json.Marshal(manifest)
		os.WriteFile(filepath.Join(skillDir, "manifest.json"), data, 0644)
	}

	loader := NewLoader(dir, 30, false)

	// Load all
	tools, err := loader.LoadAll(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}

	// Load only enabled
	tools, err = loader.LoadAll([]string{"skill_a"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0].Name() != "skill_skill_a" {
		t.Fatalf("expected 'skill_skill_a', got %s", tools[0].Name())
	}
}

func TestLoaderListInstalled(t *testing.T) {
	dir := t.TempDir()

	skillDir := filepath.Join(dir, "test_skill")
	os.MkdirAll(skillDir, 0755)
	manifest := Manifest{
		Name:        "test_skill",
		Version:     "2.0.0",
		Description: "Test",
		Author:      "tester",
		Command:     "echo hi",
	}
	data, _ := json.Marshal(manifest)
	os.WriteFile(filepath.Join(skillDir, "manifest.json"), data, 0644)

	loader := NewLoader(dir, 30, false)
	skills := loader.ListInstalled(nil)

	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if skills[0].Name != "test_skill" {
		t.Fatalf("expected 'test_skill', got %s", skills[0].Name)
	}
	if skills[0].Version != "2.0.0" {
		t.Fatalf("expected '2.0.0', got %s", skills[0].Version)
	}
	if !skills[0].Enabled {
		t.Fatal("expected skill to be enabled")
	}
}

func TestSkillToolTimeout(t *testing.T) {
	dir := t.TempDir()

	// Create a script that sleeps forever
	script := filepath.Join(dir, "slow.sh")
	os.WriteFile(script, []byte("#!/bin/sh\nsleep 60\n"), 0755)

	manifest := Manifest{
		Name:        "slow_skill",
		Version:     "1.0.0",
		Description: "Slow skill",
		Command:     "sh slow.sh",
		TimeoutSecs: 1,
	}

	st := NewSkillTool(manifest, dir, 1, false)

	start := time.Now()
	result, err := st.Execute(context.Background(), json.RawMessage(`{}`))
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected timeout error")
	}
	if elapsed > 5*time.Second {
		t.Fatalf("timeout took too long: %v", elapsed)
	}
}
