package skill

import "encoding/json"

// Manifest describes a skill plugin loaded from disk.
type Manifest struct {
	Name        string          `json:"name"`
	Version     string          `json:"version"`
	Description string          `json:"description"`
	Author      string          `json:"author,omitempty"`
	Parameters  json.RawMessage `json:"parameters"`
	Command     string          `json:"command"`
	TimeoutSecs int             `json:"timeout_secs,omitempty"`
}

// SkillInfo is a summary of an installed skill (exposed to UI).
type SkillInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Enabled     bool   `json:"enabled"`
}
