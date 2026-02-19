package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	loader := &Loader{
		filePath: filepath.Join(t.TempDir(), "config.json"),
	}

	cfg, err := loader.Load()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.LLM.Provider != "openai" {
		t.Fatalf("expected openai, got %s", cfg.LLM.Provider)
	}
	if cfg.Agent.MaxTokens != 4096 {
		t.Fatalf("expected 4096, got %d", cfg.Agent.MaxTokens)
	}
}

func TestSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	loader := &Loader{filePath: path}

	cfg := Defaults()
	cfg.LLM.Provider = "anthropic"
	cfg.LLM.APIKey = "test-key"
	cfg.SetupCompleted = true

	if err := loader.Save(cfg); err != nil {
		t.Fatal(err)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Load back
	loaded, err := loader.Load()
	if err != nil {
		t.Fatal(err)
	}

	if loaded.LLM.Provider != "anthropic" {
		t.Fatalf("expected anthropic, got %s", loaded.LLM.Provider)
	}
	if loaded.LLM.APIKey != "test-key" {
		t.Fatalf("expected test-key, got %s", loaded.LLM.APIKey)
	}
	if !loaded.SetupCompleted {
		t.Fatal("expected setup_completed to be true")
	}
}
