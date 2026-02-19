package skill

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"open-dan/internal/tool"
)

// SkillTool wraps an external skill script as a tool.Tool.
type SkillTool struct {
	manifest   Manifest
	dir        string
	timeoutSec int
	sandbox    bool
}

// NewSkillTool creates a SkillTool from a manifest and its directory.
func NewSkillTool(manifest Manifest, dir string, defaultTimeout int, sandbox bool) *SkillTool {
	timeout := manifest.TimeoutSecs
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	if timeout <= 0 {
		timeout = 60
	}
	return &SkillTool{
		manifest:   manifest,
		dir:        dir,
		timeoutSec: timeout,
		sandbox:    sandbox,
	}
}

func (s *SkillTool) Name() string { return "skill_" + s.manifest.Name }

func (s *SkillTool) Description() string {
	return fmt.Sprintf("[Skill] %s (v%s): %s", s.manifest.Name, s.manifest.Version, s.manifest.Description)
}

func (s *SkillTool) Parameters() json.RawMessage {
	if len(s.manifest.Parameters) > 0 {
		return s.manifest.Parameters
	}
	return json.RawMessage(`{"type":"object","properties":{}}`)
}

func (s *SkillTool) Execute(ctx context.Context, args json.RawMessage) (*tool.Result, error) {
	// Sandbox validation: block dangerous commands
	if s.sandbox {
		if err := validateSkillCommand(s.manifest.Command); err != nil {
			return &tool.Result{Error: "sandbox violation: " + err.Error(), IsError: true}, nil
		}
	}

	timeout := time.Duration(s.timeoutSec) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	parts := splitCommand(s.manifest.Command)
	if len(parts) == 0 {
		return &tool.Result{Error: "skill command is empty", IsError: true}, nil
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = s.dir
	cmd.WaitDelay = 2 * time.Second

	// Pass arguments via stdin as JSON
	cmd.Stdin = bytes.NewReader(args)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		if len(errMsg) > 10000 {
			errMsg = errMsg[:10000] + "\n... (truncated)"
		}
		return &tool.Result{Error: errMsg, IsError: true}, nil
	}

	output := stdout.String()
	if len(output) > 10000 {
		output = output[:10000] + "\n... (output truncated)"
	}

	return &tool.Result{Output: output}, nil
}

// validateSkillCommand checks that the command doesn't try path traversal
// or reference absolute paths outside the skill directory.
func validateSkillCommand(cmd string) error {
	parts := splitCommand(cmd)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	program := parts[0]

	// Block absolute paths (the executable must be on PATH or relative to skill dir)
	if filepath.IsAbs(program) {
		return fmt.Errorf("absolute paths not allowed in skill command: %s", program)
	}

	// Block path traversal
	if strings.Contains(program, "..") {
		return fmt.Errorf("path traversal not allowed in skill command: %s", program)
	}

	return nil
}

// splitCommand splits a command string into program and arguments,
// respecting single and double quotes.
func splitCommand(cmd string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	for _, ch := range cmd {
		switch {
		case ch == '"' || ch == '\'':
			inQuote = !inQuote
		case ch == ' ' && !inQuote:
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}
