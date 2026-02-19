package config

// Config is the top-level application configuration.
type Config struct {
	Agent          AgentConfig    `json:"agent"`
	LLM            LLMConfig      `json:"llm"`
	FallbackLLM    *LLMConfig     `json:"fallback_llm,omitempty"`
	Channels       ChannelsConfig `json:"channels"`
	Security       SecurityConfig `json:"security"`
	Browser        BrowserConfig  `json:"browser"`
	Plugins        PluginsConfig  `json:"plugins"`
	SetupCompleted bool           `json:"setup_completed"`
}

type AgentConfig struct {
	SystemPrompt  string  `json:"system_prompt"`
	MaxTokens     int     `json:"max_tokens"`
	Temperature   float64 `json:"temperature"`
	MaxToolCalls  int     `json:"max_tool_calls"`
	ContextWindow int     `json:"context_window"`
	SummarizeAt   int     `json:"summarize_at"`
}

type LLMConfig struct {
	Provider    string `json:"provider"`
	Model       string `json:"model"`
	APIKey      string `json:"api_key,omitempty"`
	BaseURL     string `json:"base_url,omitempty"`
	MaxRetries  int    `json:"max_retries"`
	TimeoutSecs int    `json:"timeout_secs"`
}

type ChannelsConfig struct {
	Telegram *TelegramConfig `json:"telegram,omitempty"`
}

type TelegramConfig struct {
	Token      string   `json:"token"`
	AllowedIDs []int64  `json:"allowed_ids,omitempty"`
}

type SecurityConfig struct {
	MasterPasswordHash string          `json:"master_password_hash,omitempty"`
	PIIFiltering       PIIFilterConfig `json:"pii_filtering"`
	Sandbox            SandboxConfig   `json:"sandbox"`
}

type PIIFilterConfig struct {
	Enabled      bool `json:"enabled"`
	FilterEmails bool `json:"filter_emails"`
	FilterPhones bool `json:"filter_phones"`
	FilterCards  bool `json:"filter_cards"`
	FilterIPs    bool `json:"filter_ips"`
	FilterSSN    bool `json:"filter_ssn"`
}

type SandboxConfig struct {
	Enabled        bool   `json:"enabled"`
	WorkspaceDir   string `json:"workspace_dir,omitempty"`
	TimeoutSecs    int    `json:"timeout_secs"`
	MaxOutputChars int    `json:"max_output_chars"`
}

type BrowserConfig struct {
	Enabled        bool     `json:"enabled"`
	Headless       bool     `json:"headless"`
	TimeoutSecs    int      `json:"timeout_secs"`
	MaxTabs        int      `json:"max_tabs"`
	AllowedDomains []string `json:"allowed_domains,omitempty"`
	DeniedDomains  []string `json:"denied_domains,omitempty"`
	MaxPageSizeKB  int      `json:"max_page_size_kb"`
}

type PluginsConfig struct {
	Enabled        bool     `json:"enabled"`
	SkillsDir      string   `json:"skills_dir,omitempty"`
	EnabledSkills  []string `json:"enabled_skills,omitempty"`
	TimeoutSecs    int      `json:"timeout_secs"`
	SandboxEnabled bool     `json:"sandbox_enabled"`
}
