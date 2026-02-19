package skill

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"open-dan/internal/tool"
)

const maxManifestSize = 64 * 1024 // 64KB limit for manifest.json

// Loader discovers and loads skill plugins from a directory.
type Loader struct {
	skillsDir      string
	defaultTimeout int
	sandbox        bool
}

// NewLoader creates a new skill loader.
func NewLoader(skillsDir string, defaultTimeout int, sandbox bool) *Loader {
	if defaultTimeout <= 0 {
		defaultTimeout = 60
	}
	return &Loader{
		skillsDir:      skillsDir,
		defaultTimeout: defaultTimeout,
		sandbox:        sandbox,
	}
}

// LoadAll scans the skills directory and returns Tool implementations for enabled skills.
// If enabledSkills is nil or empty, all discovered skills are loaded.
func (l *Loader) LoadAll(enabledSkills []string) ([]tool.Tool, error) {
	if l.skillsDir == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(l.skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read skills dir: %w", err)
	}

	enabledSet := make(map[string]bool)
	for _, name := range enabledSkills {
		enabledSet[name] = true
	}

	var tools []tool.Tool

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		// If enabledSkills filter is set, skip disabled skills
		if len(enabledSet) > 0 && !enabledSet[name] {
			continue
		}

		dir := filepath.Join(l.skillsDir, name)
		manifestPath := filepath.Join(dir, "manifest.json")

		manifest, err := parseManifest(manifestPath)
		if err != nil {
			continue // Skip invalid skills
		}

		tools = append(tools, NewSkillTool(*manifest, dir, l.defaultTimeout, l.sandbox))
	}

	return tools, nil
}

// ListInstalled returns info about all installed skills.
func (l *Loader) ListInstalled(enabledSkills []string) []SkillInfo {
	if l.skillsDir == "" {
		return nil
	}

	entries, err := os.ReadDir(l.skillsDir)
	if err != nil {
		return nil
	}

	enabledSet := make(map[string]bool)
	for _, name := range enabledSkills {
		enabledSet[name] = true
	}

	var skills []SkillInfo

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		dir := filepath.Join(l.skillsDir, name)
		manifestPath := filepath.Join(dir, "manifest.json")

		manifest, err := parseManifest(manifestPath)
		if err != nil {
			continue
		}

		// If no enabledSkills filter, all are enabled
		enabled := len(enabledSet) == 0 || enabledSet[name]

		skills = append(skills, SkillInfo{
			Name:        manifest.Name,
			Version:     manifest.Version,
			Description: manifest.Description,
			Author:      manifest.Author,
			Enabled:     enabled,
		})
	}

	return skills
}

func parseManifest(path string) (*Manifest, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Limit read size to prevent OOM from oversized manifest files
	data, err := io.ReadAll(io.LimitReader(f, maxManifestSize))
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}

	if m.Name == "" || m.Command == "" {
		return nil, fmt.Errorf("manifest missing required fields (name, command)")
	}

	return &m, nil
}
