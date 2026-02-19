package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// denyPatterns uses regex for robust matching that resists obfuscation.
var denyPatterns = []*regexp.Regexp{
	// Destructive file operations
	regexp.MustCompile(`(?i)\brm\s+-[rRf]{1,3}\s+[/~*]`),
	regexp.MustCompile(`(?i)\brm\s+-[rRf]{1,3}\b`),
	regexp.MustCompile(`(?i)\bmkfs\b`),
	regexp.MustCompile(`(?i)\bdd\s+if=`),
	regexp.MustCompile(`:\(\)\s*\{.*\|.*&\s*\}\s*;`), // fork bomb

	// System control
	regexp.MustCompile(`(?i)\b(shutdown|reboot|poweroff|halt)\b`),
	regexp.MustCompile(`(?i)\bchmod\s+-R\s+777\s+/`),
	regexp.MustCompile(`(?i)\bchown\s+-R\b`),

	// Device access
	regexp.MustCompile(`>\s*/dev/sd[a-z]`),

	// Remote code execution via pipe
	regexp.MustCompile(`(?i)\b(curl|wget)\b.*\|\s*(sh|bash)\b`),

	// Shell meta-execution
	regexp.MustCompile(`(?i)\beval\b`),
	regexp.MustCompile(`(?i)\bexec\b`),

	// Privilege escalation + destructive combos
	regexp.MustCompile(`(?i)\bsudo\s+(rm|dd|mkfs)\b`),

	// Process control
	regexp.MustCompile(`(?i)\b(killall|kill\s+-9)\b`),

	// User management
	regexp.MustCompile(`(?i)\b(passwd|useradd|userdel|usermod)\b`),

	// Firewall
	regexp.MustCompile(`(?i)\biptables\s+-F\b`),
	regexp.MustCompile(`(?i)\bufw\s+disable\b`),

	// Network listeners
	regexp.MustCompile(`(?i)\b(nc|ncat)\s+-l\b`),

	// Inline script execution
	regexp.MustCompile(`(?i)\b(python3?|perl|ruby)\s+-[ce]\b`),

	// Anti-forensics
	regexp.MustCompile(`(?i)\bbase64\s+-d\b`),
	regexp.MustCompile(`(?i)\bhistory\s+-c\b`),
	regexp.MustCompile(`(?i)\bshred\b`),

	// Sensitive files
	regexp.MustCompile(`/etc/(shadow|passwd)\b`),

	// Cron/service management
	regexp.MustCompile(`(?i)\bcrontab\s+-r\b`),
	regexp.MustCompile(`(?i)\bsystemctl\s+(stop|disable)\b`),
	regexp.MustCompile(`(?i)\blaunchctl\s+unload\b`),
	regexp.MustCompile(`(?i)\bdefaults\s+delete\b`),

	// Bulk deletion
	regexp.MustCompile(`(?i)\bxargs\s+rm\b`),
	regexp.MustCompile(`(?i)\bfind\s+/\s+.*-delete\b`),
	regexp.MustCompile(`(?i)\btruncate\s+-s\s+0\b`),

	// Entropy/DoS
	regexp.MustCompile(`(?i)\bcat\s+/dev/urandom\b`),
	regexp.MustCompile(`(?i)\bfork\(\)`),
	regexp.MustCompile(`(?i)\bwhile\s+true\b`),

	// Background persistence
	regexp.MustCompile(`(?i)\bnohup\b`),

	// Remote transfer destructive
	regexp.MustCompile(`(?i)\bscp\b`),
	regexp.MustCompile(`(?i)\brsync\s+--delete\b`),

	// VCS/package destructive
	regexp.MustCompile(`(?i)\bgit\s+push\s+--force\b`),
	regexp.MustCompile(`(?i)\bnpm\s+publish\b`),
	regexp.MustCompile(`(?i)\bpip\s+install\s+--`),

	// Container destructive
	regexp.MustCompile(`(?i)\bdocker\s+(rm|rmi)\s+-f\b`),
}

// ShellTool executes shell commands in a sandboxed environment.
type ShellTool struct {
	workspaceDir   string
	timeoutSecs    int
	maxOutputChars int
	sandboxEnabled bool
}

// ShellConfig configures the shell tool.
type ShellConfig struct {
	WorkspaceDir   string
	TimeoutSecs    int
	MaxOutputChars int
	SandboxEnabled bool
}

// NewShellTool creates a new shell tool.
func NewShellTool(cfg ShellConfig) *ShellTool {
	if cfg.TimeoutSecs <= 0 {
		cfg.TimeoutSecs = 60
	}
	if cfg.MaxOutputChars <= 0 {
		cfg.MaxOutputChars = 10000
	}
	return &ShellTool{
		workspaceDir:   cfg.WorkspaceDir,
		timeoutSecs:    cfg.TimeoutSecs,
		maxOutputChars: cfg.MaxOutputChars,
		sandboxEnabled: cfg.SandboxEnabled,
	}
}

func (t *ShellTool) Name() string { return "shell" }
func (t *ShellTool) Description() string {
	return "Execute a shell command. Use this to run system commands, scripts, and programs. Commands are sandboxed to the workspace directory."
}

func (t *ShellTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"command": {
				"type": "string",
				"description": "The shell command to execute"
			}
		},
		"required": ["command"]
	}`)
}

func (t *ShellTool) Execute(ctx context.Context, args json.RawMessage) (*Result, error) {
	var params struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return &Result{Error: "invalid arguments: " + err.Error(), IsError: true}, nil
	}

	if params.Command == "" {
		return &Result{Error: "command is required", IsError: true}, nil
	}

	// Sandbox checks
	if t.sandboxEnabled {
		if reason := t.checkDenyList(params.Command); reason != "" {
			return &Result{
				Error:   fmt.Sprintf("command blocked by sandbox: %s", reason),
				IsError: true,
			}, nil
		}
		// Block path traversal
		if strings.Contains(params.Command, "../") {
			return &Result{Error: "command blocked: path traversal detected", IsError: true}, nil
		}
		// Block absolute paths outside workspace to limit filesystem reach
		if t.workspaceDir != "" && containsAbsolutePathOutsideWorkspace(params.Command, t.workspaceDir) {
			return &Result{Error: "command blocked: absolute path outside workspace", IsError: true}, nil
		}
	}

	timeout := time.Duration(t.timeoutSecs) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", params.Command)
	if t.workspaceDir != "" {
		cmd.Dir = t.workspaceDir
	}

	output, err := cmd.CombinedOutput()
	result := string(output)

	// Truncate if needed
	if len(result) > t.maxOutputChars {
		result = result[:t.maxOutputChars] + "\n... (output truncated)"
	}

	if err != nil {
		return &Result{
			Output:  result,
			Error:   err.Error(),
			IsError: true,
		}, nil
	}

	return &Result{Output: result}, nil
}

func (t *ShellTool) checkDenyList(command string) string {
	// Normalize whitespace to prevent multi-space bypass
	normalized := collapseWhitespace(command)
	for _, pattern := range denyPatterns {
		if pattern.MatchString(normalized) {
			return fmt.Sprintf("matches deny pattern: %s", pattern.String())
		}
	}
	return ""
}

// collapseWhitespace replaces multiple whitespace chars with a single space.
func collapseWhitespace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inSpace := false
	for _, ch := range s {
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			if !inSpace {
				b.WriteByte(' ')
				inSpace = true
			}
		} else {
			b.WriteRune(ch)
			inSpace = false
		}
	}
	return b.String()
}

// containsAbsolutePathOutsideWorkspace checks for /etc, /usr etc references.
func containsAbsolutePathOutsideWorkspace(command, workspace string) bool {
	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return false
	}
	// Find absolute paths in the command
	absPathPattern := regexp.MustCompile(`(?:^|\s)(/[a-zA-Z][a-zA-Z0-9_/.-]*)`)
	matches := absPathPattern.FindAllStringSubmatch(command, -1)
	for _, m := range matches {
		if len(m) > 1 {
			path := m[1]
			if !strings.HasPrefix(path, absWorkspace) && !strings.HasPrefix(path, "/dev/null") {
				return true
			}
		}
	}
	return false
}
