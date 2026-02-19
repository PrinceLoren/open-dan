package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

const (
	configDir  = ".opendan"
	configFile = "config.json"
)

// Loader manages reading and writing the config file.
type Loader struct {
	mu       sync.RWMutex
	config   *Config
	filePath string
}

// NewLoader creates a loader that stores config in ~/.opendan/config.json.
func NewLoader() (*Loader, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, configDir)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &Loader{
		filePath: filepath.Join(dir, configFile),
	}, nil
}

// Load reads the config from disk. If the file doesn't exist, returns defaults.
func (l *Loader) Load() (*Config, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	cfg := Defaults()

	data, err := os.ReadFile(l.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			l.config = cfg
			return cfg, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	l.config = cfg
	return cfg, nil
}

// Save writes the current config to disk.
func (l *Loader) Save(cfg *Config) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	l.config = cfg
	return os.WriteFile(l.filePath, data, 0600)
}

// Get returns the currently loaded config (or defaults if not loaded yet).
func (l *Loader) Get() *Config {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.config == nil {
		return Defaults()
	}
	return l.config
}

// FilePath returns the config file path.
func (l *Loader) FilePath() string {
	return l.filePath
}
