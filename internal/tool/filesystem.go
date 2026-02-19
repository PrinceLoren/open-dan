package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FilesystemTool provides sandboxed file read/write operations.
type FilesystemTool struct {
	workspaceDir string
}

func NewFilesystemTool(workspaceDir string) *FilesystemTool {
	return &FilesystemTool{workspaceDir: workspaceDir}
}

func (t *FilesystemTool) Name() string        { return "filesystem" }
func (t *FilesystemTool) Description() string  {
	return "Read or write files within the workspace directory. Use action 'read' to read a file, 'write' to create/overwrite a file, 'list' to list directory contents."
}

func (t *FilesystemTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"enum": ["read", "write", "list"],
				"description": "The file operation to perform"
			},
			"path": {
				"type": "string",
				"description": "Relative path within workspace"
			},
			"content": {
				"type": "string",
				"description": "Content to write (only for 'write' action)"
			}
		},
		"required": ["action", "path"]
	}`)
}

func (t *FilesystemTool) Execute(ctx context.Context, args json.RawMessage) (*Result, error) {
	var params struct {
		Action  string `json:"action"`
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return &Result{Error: "invalid arguments: " + err.Error(), IsError: true}, nil
	}

	// Resolve and validate path
	fullPath, err := t.resolvePath(params.Path)
	if err != nil {
		return &Result{Error: err.Error(), IsError: true}, nil
	}

	switch params.Action {
	case "read":
		return t.readFile(fullPath)
	case "write":
		return t.writeFile(fullPath, params.Content)
	case "list":
		return t.listDir(fullPath)
	default:
		return &Result{Error: "unknown action: " + params.Action, IsError: true}, nil
	}
}

func (t *FilesystemTool) resolvePath(relPath string) (string, error) {
	if t.workspaceDir == "" {
		return "", fmt.Errorf("workspace directory not configured")
	}

	// Prevent path traversal
	if strings.Contains(relPath, "..") {
		return "", fmt.Errorf("path traversal not allowed")
	}

	fullPath := filepath.Join(t.workspaceDir, filepath.Clean(relPath))

	// Verify the resolved path is within workspace
	absWorkspace, _ := filepath.Abs(t.workspaceDir)
	absPath, _ := filepath.Abs(fullPath)
	if !strings.HasPrefix(absPath, absWorkspace) {
		return "", fmt.Errorf("path outside workspace")
	}

	// Check symlinks
	if resolved, err := filepath.EvalSymlinks(filepath.Dir(fullPath)); err == nil {
		if !strings.HasPrefix(resolved, absWorkspace) {
			return "", fmt.Errorf("symlink escapes workspace")
		}
	}

	return fullPath, nil
}

func (t *FilesystemTool) readFile(path string) (*Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return &Result{Error: "failed to read file: " + err.Error(), IsError: true}, nil
	}
	output := string(data)
	if len(output) > 50000 {
		output = output[:50000] + "\n... (file truncated)"
	}
	return &Result{Output: output}, nil
}

func (t *FilesystemTool) writeFile(path, content string) (*Result, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &Result{Error: "failed to create directory: " + err.Error(), IsError: true}, nil
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return &Result{Error: "failed to write file: " + err.Error(), IsError: true}, nil
	}
	return &Result{Output: fmt.Sprintf("File written: %s (%d bytes)", path, len(content))}, nil
}

func (t *FilesystemTool) listDir(path string) (*Result, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return &Result{Error: "failed to list directory: " + err.Error(), IsError: true}, nil
	}
	var lines []string
	for _, e := range entries {
		prefix := "  "
		if e.IsDir() {
			prefix = "d "
		}
		lines = append(lines, prefix+e.Name())
	}
	return &Result{Output: strings.Join(lines, "\n")}, nil
}
