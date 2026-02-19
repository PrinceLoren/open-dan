package config

// Defaults returns a Config with sensible default values.
func Defaults() *Config {
	return &Config{
		Agent: AgentConfig{
			SystemPrompt:    "You are OpenDan, a helpful AI assistant. You can use tools to accomplish tasks.",
			MaxTokens:       4096,
			Temperature:     0.7,
			MaxToolCalls:    20,
			ContextWindow:   100000,
			SummarizeAt:     80000,
		},
		LLM: LLMConfig{
			Provider:    "openai",
			Model:       "gpt-4o-mini",
			MaxRetries:  3,
			TimeoutSecs: 120,
		},
		Security: SecurityConfig{
			PIIFiltering: PIIFilterConfig{
				Enabled:      true,
				FilterEmails: true,
				FilterPhones: true,
				FilterCards:  true,
				FilterIPs:    false,
				FilterSSN:    true,
			},
			Sandbox: SandboxConfig{
				Enabled:        true,
				TimeoutSecs:    60,
				MaxOutputChars: 10000,
			},
		},
		Channels: ChannelsConfig{},
		Browser: BrowserConfig{
			Enabled:       false,
			Headless:      true,
			TimeoutSecs:   30,
			MaxTabs:       3,
			MaxPageSizeKB: 2048,
		},
		Plugins: PluginsConfig{
			Enabled:        true,
			TimeoutSecs:    60,
			SandboxEnabled: true,
		},
		SetupCompleted: false,
	}
}
