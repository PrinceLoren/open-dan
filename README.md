<p align="center">
  <img src="build/appicon.png" width="128" height="128" alt="OpenDan logo">
</p>

<h1 align="center">OpenDan</h1>

<p align="center">
  <strong>Autonomous AI bot agent as a native desktop app</strong><br>
  Built with Go + React &middot; Single binary &middot; ~25 MB
</p>

<p align="center">
  <a href="#features">Features</a> &middot;
  <a href="#quick-start">Quick Start</a> &middot;
  <a href="#configuration">Configuration</a> &middot;
  <a href="#skills--plugins">Skills & Plugins</a> &middot;
  <a href="#architecture">Architecture</a> &middot;
  <a href="#security">Security</a>
</p>

---

## Features

- **Multi-provider LLM** — Anthropic Claude, OpenAI, and any OpenAI-compatible API (Ollama, LM Studio, vLLM) with automatic fallback
- **Think → Act → Observe loop** — agent autonomously reasons, uses tools, and iterates until the task is complete
- **Built-in tools** — shell (sandboxed), filesystem (path-safe), web search (DuckDuckGo), browser automation (headless Chromium)
- **Skills & Plugins** — extend the agent with external scripts in any language, no recompilation needed
- **Telegram integration** — connect your bot token, control access with user allowlists
- **GUI chat** — built-in chat interface in the desktop app with real-time streaming
- **Persistent memory** — conversation history and summaries stored in SQLite
- **Secure by default** — API keys in OS Keychain, PII filtering, sandbox enforcement, SSRF protection

## Quick Start

### Prerequisites

- **Go 1.24+**
- **Node.js 18+**
- **Wails CLI v2**: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

### Development

```bash
git clone https://github.com/PrinceLoren/open-dan.git
cd open-dan
wails dev
```

The app window opens automatically. A setup wizard will guide you through:

1. Choose your LLM provider and enter the API key
2. (Optional) Connect a Telegram bot
3. Configure security preferences
4. Done — start chatting

### Production Build

```bash
wails build
```

The binary is at `build/bin/open-dan` (macOS: `open-dan.app`).

### Run Tests

```bash
go test ./internal/...
```

## Configuration

All settings are stored in `~/.opendan/config.json`. API keys are stored securely in your OS Keychain (macOS Keychain / Linux Secret Service) — the config file only contains a `[keyring]` placeholder.

| Setting | Location |
|---------|----------|
| Config file | `~/.opendan/config.json` |
| SQLite database | `~/.opendan/memory.db` |
| Skills directory | `~/.opendan/skills/` |
| Workspace (sandbox) | `~/.opendan/workspace/` |

You can edit all settings through the GUI (Settings page) or by editing the JSON file directly.

### LLM Providers

| Provider | Config `provider` value | Notes |
|----------|------------------------|-------|
| Anthropic | `anthropic` | Claude models |
| OpenAI | `openai` | GPT models |
| Ollama | `openai` | Set `base_url` to `http://localhost:11434/v1` |
| LM Studio | `openai` | Set `base_url` to `http://localhost:1234/v1` |

### Telegram Bot

1. Create a bot via [@BotFather](https://t.me/BotFather)
2. Enter the token in Settings → Telegram
3. Add allowed user IDs to restrict access (recommended)

## Skills & Plugins

Extend the agent with custom skills — executable scripts in any language.

### Creating a Skill

Create a directory in `~/.opendan/skills/<name>/` with two files:

**`manifest.json`**
```json
{
  "name": "weather",
  "version": "1.0.0",
  "description": "Get current weather for a location",
  "author": "you",
  "command": "python3 run.py",
  "timeout_secs": 30,
  "parameters": {
    "type": "object",
    "properties": {
      "location": {
        "type": "string",
        "description": "City name or coordinates"
      }
    },
    "required": ["location"]
  }
}
```

**`run.py`** (or any executable)
```python
import json, sys

args = json.load(sys.stdin)
location = args["location"]
print(json.dumps({"weather": "sunny", "location": location}))
```

The agent receives the skill as a tool and can call it autonomously. Arguments are passed via stdin as JSON, output is read from stdout.

### Managing Skills

- Enable/disable individual skills in Settings → Skills & Plugins
- Skills run in a sandbox by default (no absolute paths, timeout enforced)
- The agent sees skills as tools named `skill_<name>`

## Browser Automation

When enabled, the agent can control a headless Chromium browser:

- **Navigate** to URLs and read page content
- **Click** elements and **fill** forms by CSS selector
- **Take screenshots** (base64 JPEG)
- **Execute JavaScript** on pages
- **Extract links** from pages

Enable in Settings → Browser Control. Security features:
- Only `http://https` URLs allowed
- Private/loopback IPs blocked (SSRF protection)
- Domain allowlist/denylist support
- Tab limit (default: 3)

## Architecture

```
User (Telegram / GUI)
  → Channel.OnMessage()
    → EventBus.Publish("inbound_message")
      → Agent.handleMessage()
        → Agent.processMessage()         [loop]
          → LLM Provider.Chat()          [think]
          → Tool.Execute()               [act]
          → append result to context     [observe]
          → repeat until final answer
        → Channel.Send()                 [respond]
```

### Project Structure

```
open-dan/
├── main.go                     # Wails entry point, GC tuning
├── app.go                      # Wails bindings, module init, Keychain integration
├── internal/
│   ├── agent/                  # Agent core (think-act-observe loop)
│   ├── llm/                    # LLM providers (Anthropic, OpenAI, fallback)
│   ├── channel/                # Messaging (Telegram, console, GUI)
│   ├── tool/                   # Tools (shell, filesystem, websearch, browser)
│   ├── skill/                  # Plugin system (manifest, loader, executor)
│   ├── memory/                 # SQLite persistence (messages, summaries)
│   ├── security/               # Keychain, encryption, PII sanitizer, sandbox
│   ├── eventbus/               # Pub/sub event system
│   └── config/                 # Configuration management
└── frontend/                   # React + TypeScript + Vite
    └── src/
        ├── pages/              # Dashboard, Settings, SetupWizard
        └── components/         # StatusCard, ProviderForm, ChannelForm
```

### Key Interfaces

```go
// LLM Provider
type Provider interface {
    Chat(ctx context.Context, req *ChatRequest) (*LLMResponse, error)
    StreamChat(ctx context.Context, req *ChatRequest) (<-chan StreamEvent, error)
}

// Tool
type Tool interface {
    Name() string
    Description() string
    Parameters() json.RawMessage
    Execute(ctx context.Context, args json.RawMessage) (*Result, error)
}

// Channel
type Channel interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Send(ctx context.Context, msg OutboundMessage) error
    OnMessage(handler MessageHandler)
}
```

## Security

| Feature | Implementation |
|---------|---------------|
| API key storage | macOS Keychain / Linux Secret Service, AES-256-GCM encrypted vault fallback |
| Shell sandbox | 40+ regex deny patterns, whitespace normalization, workspace restriction |
| Filesystem | Path traversal protection, symlink escape detection, 0600 file permissions |
| Browser | SSRF blocking (private IPs), scheme validation, domain allowlist/denylist |
| PII filtering | Auto-redaction of emails, phones, credit cards, IPs, SSNs |
| Telegram auth | User ID allowlist |
| Skills sandbox | No absolute paths, timeout enforcement, output truncation |
| Memory | GC tuning (GOGC=50, GOMEMLIMIT=64 MiB) for lower footprint |

## Dependencies

| Package | Purpose |
|---------|---------|
| [Wails v2](https://wails.io) | Desktop app framework (Go + WebView) |
| [anthropic-sdk-go](https://github.com/anthropics/anthropic-sdk-go) | Anthropic Claude API |
| [openai-go](https://github.com/openai/openai-go) | OpenAI API |
| [go-rod](https://github.com/go-rod/rod) | Browser automation (Chrome DevTools Protocol) |
| [telebot.v3](https://gopkg.in/telebot.v3) | Telegram Bot API |
| [modernc.org/sqlite](https://modernc.org/sqlite) | SQLite (pure Go, no CGO) |
| [go-keyring](https://github.com/zalando/go-keyring) | OS Keychain access |

## License

MIT
