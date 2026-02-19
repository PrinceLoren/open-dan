package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"

	"open-dan/internal/agent"
	"open-dan/internal/channel"
	"open-dan/internal/config"
	"open-dan/internal/eventbus"
	"open-dan/internal/llm"
	"open-dan/internal/memory"
	"open-dan/internal/security"
	"open-dan/internal/skill"
	"open-dan/internal/tool"
)

const (
	keyringPlaceholder     = "[keyring]"
	secretNameLLMKey       = "llm_api_key"
	secretNameTelegramToken = "telegram_token"
)

// App struct holds the application state and exposes methods to the frontend.
type App struct {
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.RWMutex // protects cfg and agent
	cfg       *config.Config
	cfgLoader *config.Loader
	bus       *eventbus.Bus
	agent     *agent.Agent
	chanMgr   *channel.Manager
	mem       memory.Memory
	keyStore  *security.KeyStore
	sanitizer   *security.Sanitizer
	browserTool *tool.BrowserTool
	skillLoader *skill.Loader
	logsMu      sync.Mutex // protects logs
	logs        []LogEntry
}

// LogEntry is a log line exposed to the frontend.
type LogEntry struct {
	Level   string `json:"level"`
	Message string `json:"message"`
	Time    string `json:"time"`
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{
		bus: eventbus.New(),
	}
}

// startup is called when the Wails app starts.
func (a *App) startup(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	a.ctx = ctx
	a.cancel = cancel

	// Load config
	loader, err := config.NewLoader()
	if err != nil {
		log.Printf("failed to create config loader: %v", err)
		return
	}
	a.cfgLoader = loader

	cfg, err := loader.Load()
	if err != nil {
		log.Printf("failed to load config: %v", err)
		cfg = config.Defaults()
	}
	a.cfg = cfg

	// Initialize secure key store
	ks, err := security.NewKeyStore(nil)
	if err != nil {
		log.Printf("warning: failed to create key store: %v (secrets will stay in config file)", err)
	}
	a.keyStore = ks

	// Resolve secrets from Keychain (or migrate plaintext → Keychain)
	a.resolveSecrets()

	// Initialize sanitizer
	a.sanitizer = security.NewSanitizer(cfg.Security.PIIFiltering)

	// Initialize memory (SQLite)
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("failed to get home directory: %v", err)
		return
	}
	dbPath := filepath.Join(home, ".opendan", "memory.db")
	mem, err := memory.NewSQLiteMemory(dbPath)
	if err != nil {
		log.Printf("failed to initialize memory: %v", err)
		return
	}
	a.mem = mem

	// Initialize channel manager
	a.chanMgr = channel.NewManager()

	// If setup is completed, initialize the agent
	if cfg.SetupCompleted {
		a.initAgent()
	}

	// Subscribe to events for logging
	a.bus.Subscribe(eventbus.TopicError, func(e eventbus.Event) {
		a.addLog("error", e.Payload)
	})
	a.bus.Subscribe(eventbus.TopicStatusChange, func(e eventbus.Event) {
		a.addLog("info", e.Payload)
	})
}

// shutdown is called when the app is closing.
func (a *App) shutdown(ctx context.Context) {
	if a.cancel != nil {
		a.cancel()
	}
	if a.chanMgr != nil {
		a.chanMgr.StopAll(ctx)
	}
	if a.browserTool != nil {
		a.browserTool.Close()
	}
	if a.mem != nil {
		a.mem.Close()
	}
}

func (a *App) initAgent() {
	if a.cfg.LLM.APIKey == "" {
		log.Println("LLM API key not configured, skipping agent init")
		return
	}

	// Create LLM provider
	provider, err := llm.NewProvider(a.cfg.LLM)
	if err != nil {
		log.Printf("failed to create LLM provider: %v", err)
		return
	}

	// Add fallback if configured
	if a.cfg.FallbackLLM != nil && a.cfg.FallbackLLM.APIKey != "" {
		fallback, err := llm.NewProvider(*a.cfg.FallbackLLM)
		if err == nil {
			provider = llm.NewFallbackProvider(provider, fallback)
		}
	}

	// Create tool registry
	registry := tool.NewRegistry()

	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("failed to get home directory: %v", err)
		return
	}
	workspaceDir := a.cfg.Security.Sandbox.WorkspaceDir
	if workspaceDir == "" {
		workspaceDir = filepath.Join(home, ".opendan", "workspace")
	}
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		log.Printf("failed to create workspace directory: %v", err)
		return
	}

	registry.Register(tool.NewShellTool(tool.ShellConfig{
		WorkspaceDir:   workspaceDir,
		TimeoutSecs:    a.cfg.Security.Sandbox.TimeoutSecs,
		MaxOutputChars: a.cfg.Security.Sandbox.MaxOutputChars,
		SandboxEnabled: a.cfg.Security.Sandbox.Enabled,
	}))
	registry.Register(tool.NewWebSearchTool())
	registry.Register(tool.NewFilesystemTool(workspaceDir))

	// Browser tool
	if a.cfg.Browser.Enabled {
		a.browserTool = tool.NewBrowserTool(a.cfg.Browser)
		registry.Register(a.browserTool)
	}

	// Skills
	if a.cfg.Plugins.Enabled {
		skillsDir := a.cfg.Plugins.SkillsDir
		if skillsDir == "" {
			skillsDir = filepath.Join(home, ".opendan", "skills")
		}
		if err := os.MkdirAll(skillsDir, 0755); err != nil {
			log.Printf("failed to create skills directory: %v", err)
		}
		a.skillLoader = skill.NewLoader(skillsDir, a.cfg.Plugins.TimeoutSecs, a.cfg.Plugins.SandboxEnabled)
		skills, err := a.skillLoader.LoadAll(a.cfg.Plugins.EnabledSkills)
		if err != nil {
			log.Printf("failed to load skills: %v", err)
		}
		for _, s := range skills {
			registry.Register(s)
		}
		log.Printf("Loaded %d skills", len(skills))
	}

	// Create agent
	ag := agent.New(
		a.cfg.Agent,
		provider,
		registry,
		a.mem,
		a.bus,
		a.chanMgr,
	)
	a.mu.Lock()
	a.agent = ag
	a.mu.Unlock()

	// Start Telegram if configured
	if a.cfg.Channels.Telegram != nil && a.cfg.Channels.Telegram.Token != "" {
		tg := channel.NewTelegramChannel(channel.TelegramConfig{
			Token:      a.cfg.Channels.Telegram.Token,
			AllowedIDs: a.cfg.Channels.Telegram.AllowedIDs,
		})
		a.chanMgr.Register(tg)
		if err := a.chanMgr.StartAll(a.ctx); err != nil {
			log.Printf("failed to start channels: %v", err)
		}
	}

	a.agent.Start(a.ctx)
	log.Println("Agent initialized and running")

	debug.FreeOSMemory()
}

// resolveSecrets loads secrets from Keychain into in-memory config.
// On first run, migrates plaintext secrets from config.json to Keychain.
func (a *App) resolveSecrets() {
	if a.keyStore == nil {
		return
	}

	migrated := false

	// LLM API Key
	switch {
	case a.cfg.LLM.APIKey == keyringPlaceholder:
		if val, err := a.keyStore.Get(secretNameLLMKey); err == nil {
			a.cfg.LLM.APIKey = val
		} else {
			log.Printf("warning: failed to read LLM key from keyring: %v", err)
		}
	case a.cfg.LLM.APIKey != "":
		if err := a.keyStore.Set(secretNameLLMKey, a.cfg.LLM.APIKey); err == nil {
			migrated = true
			log.Println("Migrated LLM API key to secure storage")
		}
	}

	// Telegram Token
	if a.cfg.Channels.Telegram != nil {
		switch {
		case a.cfg.Channels.Telegram.Token == keyringPlaceholder:
			if val, err := a.keyStore.Get(secretNameTelegramToken); err == nil {
				a.cfg.Channels.Telegram.Token = val
			} else {
				log.Printf("warning: failed to read Telegram token from keyring: %v", err)
			}
		case a.cfg.Channels.Telegram.Token != "":
			if err := a.keyStore.Set(secretNameTelegramToken, a.cfg.Channels.Telegram.Token); err == nil {
				migrated = true
				log.Println("Migrated Telegram token to secure storage")
			}
		}
	}

	// Rewrite config.json with placeholders instead of real keys
	if migrated {
		if err := a.saveConfig(); err != nil {
			log.Printf("warning: failed to save config after secret migration: %v", err)
		}
	}
}

// saveConfig writes config to disk with secrets replaced by [keyring] placeholders.
// In-memory a.cfg always retains real keys; only the file gets placeholders.
func (a *App) saveConfig() error {
	if a.keyStore == nil {
		return a.saveConfig()
	}

	// Store secrets in Keychain
	if a.cfg.LLM.APIKey != "" && a.cfg.LLM.APIKey != keyringPlaceholder {
		if err := a.keyStore.Set(secretNameLLMKey, a.cfg.LLM.APIKey); err != nil {
			log.Printf("warning: failed to store LLM key in keyring: %v", err)
			return a.saveConfig() // fallback: save plaintext
		}
	}
	if a.cfg.Channels.Telegram != nil && a.cfg.Channels.Telegram.Token != "" && a.cfg.Channels.Telegram.Token != keyringPlaceholder {
		if err := a.keyStore.Set(secretNameTelegramToken, a.cfg.Channels.Telegram.Token); err != nil {
			log.Printf("warning: failed to store Telegram token in keyring: %v", err)
			return a.saveConfig()
		}
	}

	// Create shallow copy with placeholders for disk
	cfgForDisk := *a.cfg
	if cfgForDisk.LLM.APIKey != "" {
		cfgForDisk.LLM.APIKey = keyringPlaceholder
	}
	if cfgForDisk.Channels.Telegram != nil && cfgForDisk.Channels.Telegram.Token != "" {
		tgCopy := *cfgForDisk.Channels.Telegram
		tgCopy.Token = keyringPlaceholder
		cfgForDisk.Channels.Telegram = &tgCopy
	}

	return a.cfgLoader.Save(&cfgForDisk)
}

func (a *App) addLog(level string, payload any) {
	entry := LogEntry{
		Level:   level,
		Message: log.Prefix(),
	}
	switch v := payload.(type) {
	case string:
		entry.Message = v
	case error:
		entry.Message = v.Error()
	}
	a.logsMu.Lock()
	a.logs = append(a.logs, entry)
	if len(a.logs) > 1000 {
		a.logs = a.logs[len(a.logs)-500:]
	}
	a.logsMu.Unlock()
}

// --- Wails Bindings (exposed to frontend) ---

// IsSetupCompleted returns whether the initial setup has been done.
func (a *App) IsSetupCompleted() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cfg != nil && a.cfg.SetupCompleted
}

// GetConfig returns the current config (with masked API keys).
func (a *App) GetConfig() map[string]any {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.cfg == nil {
		return nil
	}
	skillsCount := 0
	if a.skillLoader != nil {
		skillsCount = len(a.skillLoader.ListInstalled(a.cfg.Plugins.EnabledSkills))
	}

	return map[string]any{
		"provider":         a.cfg.LLM.Provider,
		"model":            a.cfg.LLM.Model,
		"api_key_masked":   security.MaskKey(a.cfg.LLM.APIKey),
		"base_url":         a.cfg.LLM.BaseURL,
		"has_telegram":     a.cfg.Channels.Telegram != nil && a.cfg.Channels.Telegram.Token != "",
		"pii_filtering":    a.cfg.Security.PIIFiltering.Enabled,
		"browser_enabled":  a.cfg.Browser.Enabled,
		"browser_headless": a.cfg.Browser.Headless,
		"plugins_enabled":  a.cfg.Plugins.Enabled,
		"skills_count":     skillsCount,
		"setup_completed":  a.cfg.SetupCompleted,
	}
}

// SaveLLMConfig saves LLM provider settings.
func (a *App) SaveLLMConfig(provider, apiKey, model, baseURL string) error {
	if baseURL != "" {
		if err := validateBaseURL(baseURL); err != nil {
			return err
		}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cfg.LLM.Provider = provider
	a.cfg.LLM.APIKey = apiKey
	if model != "" {
		a.cfg.LLM.Model = model
	}
	a.cfg.LLM.BaseURL = baseURL
	return a.saveConfig()
}

// SaveTelegramConfig saves Telegram settings.
func (a *App) SaveTelegramConfig(token string, allowedIDs []int64) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cfg.Channels.Telegram = &config.TelegramConfig{
		Token:      token,
		AllowedIDs: allowedIDs,
	}
	return a.saveConfig()
}

// SaveSecurityConfig saves security settings.
func (a *App) SaveSecurityConfig(piiEnabled, filterEmails, filterPhones, filterCards, filterIPs, filterSSN bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cfg.Security.PIIFiltering = config.PIIFilterConfig{
		Enabled:      piiEnabled,
		FilterEmails: filterEmails,
		FilterPhones: filterPhones,
		FilterCards:  filterCards,
		FilterIPs:    filterIPs,
		FilterSSN:    filterSSN,
	}
	return a.saveConfig()
}

// CompleteSetup marks setup as done and initializes the agent.
func (a *App) CompleteSetup() error {
	a.mu.Lock()
	a.cfg.SetupCompleted = true
	if err := a.saveConfig(); err != nil {
		a.mu.Unlock()
		return err
	}
	a.mu.Unlock()
	a.initAgent()
	return nil
}

// TestLLMConnection tests the LLM connection with current config.
func (a *App) TestLLMConnection(provider, apiKey, model, baseURL string) string {
	if baseURL != "" {
		if err := validateBaseURL(baseURL); err != nil {
			return "Error: " + err.Error()
		}
	}
	cfg := config.LLMConfig{
		Provider: provider,
		APIKey:   apiKey,
		Model:    model,
		BaseURL:  baseURL,
	}
	p, err := llm.NewProvider(cfg)
	if err != nil {
		return "Error: " + err.Error()
	}

	tmpAgent := agent.New(
		config.Defaults().Agent,
		p,
		tool.NewRegistry(),
		a.mem,
		a.bus,
		channel.NewManager(),
	)

	if err := tmpAgent.TestConnection(a.ctx); err != nil {
		return "Connection failed: " + err.Error()
	}
	return "OK"
}

// TestTelegramConnection tests a Telegram bot token.
func (a *App) TestTelegramConnection(token string) string {
	tg := channel.NewTelegramChannel(channel.TelegramConfig{Token: token})
	if err := tg.Start(a.ctx); err != nil {
		return "Connection failed: " + err.Error()
	}
	tg.Stop(a.ctx)
	return "OK"
}

// SendMessage sends a message to the agent from the GUI.
func (a *App) SendMessage(text string) string {
	a.mu.RLock()
	ag := a.agent
	a.mu.RUnlock()
	if ag == nil {
		return "Agent not initialized. Please complete setup first."
	}
	// Sanitize PII
	sanitized := a.sanitizer.Sanitize(text)
	response, err := ag.HandleDirectMessage(a.ctx, "gui", sanitized)
	if err != nil {
		return "Error: " + err.Error()
	}
	// Restore PII in response
	return a.sanitizer.Restore(response)
}

// SaveBrowserConfig saves browser control settings.
func (a *App) SaveBrowserConfig(enabled, headless bool, timeoutSecs, maxTabs int, allowedDomains, deniedDomains string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cfg.Browser.Enabled = enabled
	a.cfg.Browser.Headless = headless
	if timeoutSecs > 0 {
		a.cfg.Browser.TimeoutSecs = timeoutSecs
	}
	if maxTabs > 0 {
		a.cfg.Browser.MaxTabs = maxTabs
	}
	if allowedDomains != "" {
		a.cfg.Browser.AllowedDomains = splitAndTrim(allowedDomains)
	} else {
		a.cfg.Browser.AllowedDomains = nil
	}
	if deniedDomains != "" {
		a.cfg.Browser.DeniedDomains = splitAndTrim(deniedDomains)
	} else {
		a.cfg.Browser.DeniedDomains = nil
	}
	return a.saveConfig()
}

// SavePluginsConfig saves skills/plugins settings.
func (a *App) SavePluginsConfig(enabled bool, enabledSkills []string, timeoutSecs int, sandboxEnabled bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cfg.Plugins.Enabled = enabled
	a.cfg.Plugins.EnabledSkills = enabledSkills
	if timeoutSecs > 0 {
		a.cfg.Plugins.TimeoutSecs = timeoutSecs
	}
	a.cfg.Plugins.SandboxEnabled = sandboxEnabled
	return a.saveConfig()
}

// GetInstalledSkills returns the list of installed skills.
func (a *App) GetInstalledSkills() []skill.SkillInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.skillLoader == nil {
		// Create a temporary loader to list skills even before agent init
		home, err := os.UserHomeDir()
		if err != nil {
			log.Printf("failed to get home directory: %v", err)
			return nil
		}
		skillsDir := a.cfg.Plugins.SkillsDir
		if skillsDir == "" {
			skillsDir = filepath.Join(home, ".opendan", "skills")
		}
		loader := skill.NewLoader(skillsDir, a.cfg.Plugins.TimeoutSecs, a.cfg.Plugins.SandboxEnabled)
		return loader.ListInstalled(a.cfg.Plugins.EnabledSkills)
	}
	return a.skillLoader.ListInstalled(a.cfg.Plugins.EnabledSkills)
}

// validateBaseURL checks that a base URL is valid and uses http/https scheme.
func validateBaseURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}
	switch u.Scheme {
	case "http", "https":
	default:
		return fmt.Errorf("base URL must use http or https scheme, got: %s", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("base URL must have a host")
	}
	return nil
}

func splitAndTrim(s string) []string {
	var result []string
	for _, part := range strings.Split(s, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// GetLogs returns recent log entries.
func (a *App) GetLogs() []LogEntry {
	a.logsMu.Lock()
	copied := make([]LogEntry, len(a.logs))
	copy(copied, a.logs)
	a.logsMu.Unlock()
	return copied
}

// GetChannelStatus returns the status of all channels.
func (a *App) GetChannelStatus() map[string]bool {
	if a.chanMgr == nil {
		return nil
	}
	return a.chanMgr.List()
}

// GetMemStats returns current memory usage statistics.
func (a *App) GetMemStats() map[string]any {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return map[string]any{
		"alloc_mb":        float64(m.Alloc) / 1024 / 1024,
		"total_alloc_mb":  float64(m.TotalAlloc) / 1024 / 1024,
		"sys_mb":          float64(m.Sys) / 1024 / 1024,
		"heap_alloc_mb":   float64(m.HeapAlloc) / 1024 / 1024,
		"heap_sys_mb":     float64(m.HeapSys) / 1024 / 1024,
		"heap_objects":    m.HeapObjects,
		"goroutines":      runtime.NumGoroutine(),
		"gc_cycles":       m.NumGC,
	}
}
