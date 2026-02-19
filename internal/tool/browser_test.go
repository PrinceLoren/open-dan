package tool

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"open-dan/internal/config"
)

func TestBrowserToolInterface(t *testing.T) {
	bt := NewBrowserTool(config.BrowserConfig{
		Headless:      true,
		TimeoutSecs:   10,
		MaxTabs:       3,
		MaxPageSizeKB: 1024,
	})

	if bt.Name() != "browser" {
		t.Fatalf("expected 'browser', got %s", bt.Name())
	}

	if bt.Description() == "" {
		t.Fatal("description should not be empty")
	}

	var schema map[string]any
	if err := json.Unmarshal(bt.Parameters(), &schema); err != nil {
		t.Fatalf("invalid parameters JSON: %v", err)
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("parameters should have 'properties'")
	}

	if _, ok := props["action"]; !ok {
		t.Fatal("parameters should have 'action' property")
	}
}

func TestBrowserDomainValidation(t *testing.T) {
	tests := []struct {
		name           string
		allowedDomains []string
		deniedDomains  []string
		url            string
		expectError    bool
	}{
		{
			name:        "no restrictions",
			url:         "https://example.com",
			expectError: false,
		},
		{
			name:          "denied domain",
			deniedDomains: []string{"evil.com"},
			url:           "https://evil.com/path",
			expectError:   true,
		},
		{
			name:          "denied subdomain",
			deniedDomains: []string{"evil.com"},
			url:           "https://sub.evil.com/path",
			expectError:   true,
		},
		{
			name:           "allowed domain",
			allowedDomains: []string{"example.com"},
			url:            "https://example.com/page",
			expectError:    false,
		},
		{
			name:           "not in allowed list",
			allowedDomains: []string{"example.com"},
			url:            "https://other.com/page",
			expectError:    true,
		},
		{
			name:        "block localhost (SSRF)",
			url:         "http://localhost:8080/admin",
			expectError: true,
		},
		{
			name:        "block private IP (SSRF)",
			url:         "http://192.168.1.1/admin",
			expectError: true,
		},
		{
			name:        "block file scheme",
			url:         "file:///etc/passwd",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bt := NewBrowserTool(config.BrowserConfig{
				Headless:       true,
				TimeoutSecs:    10,
				MaxTabs:        3,
				MaxPageSizeKB:  1024,
				AllowedDomains: tt.allowedDomains,
				DeniedDomains:  tt.deniedDomains,
			})

			err := bt.validateURL(tt.url)
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestBrowserMaxTabs(t *testing.T) {
	bt := NewBrowserTool(config.BrowserConfig{
		Headless:      true,
		TimeoutSecs:   10,
		MaxTabs:       2,
		MaxPageSizeKB: 1024,
	})

	// Simulate having max tabs already open
	bt.pages["page_1"] = nil
	bt.pages["page_2"] = nil

	args, _ := json.Marshal(browserParams{Action: "navigate", URL: "https://example.com"})
	result, err := bt.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for max tabs")
	}
	if !strings.Contains(result.Error, "max tabs") {
		t.Fatalf("expected 'max tabs' in error, got: %s", result.Error)
	}
}

func TestBrowserUnknownAction(t *testing.T) {
	bt := NewBrowserTool(config.BrowserConfig{
		Headless:      true,
		TimeoutSecs:   10,
		MaxTabs:       3,
		MaxPageSizeKB: 1024,
	})

	args, _ := json.Marshal(browserParams{Action: "unknown"})
	result, err := bt.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for unknown action")
	}
}
