package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateWorkspace checks that a workspace directory is safe to use.
func ValidateWorkspace(dir string) error {
	if dir == "" {
		return fmt.Errorf("workspace directory is empty")
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("invalid workspace path: %w", err)
	}

	// Prevent using root or home directory directly
	home, _ := os.UserHomeDir()
	if abs == "/" || abs == home {
		return fmt.Errorf("cannot use root or home directory as workspace")
	}

	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(abs, 0755)
		}
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("workspace path is not a directory")
	}

	return nil
}

// IsPathSafe checks if a path stays within the workspace.
func IsPathSafe(path, workspace string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return false
	}
	return strings.HasPrefix(absPath, absWorkspace)
}
