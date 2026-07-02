# llm-usage

A CLI tool to display your LLM API usage statistics across multiple providers (Claude, Kimi, MiniMax, Z.AI).

## Features

- View real-time usage statistics from multiple LLM providers
- Multiple named accounts per provider
- Multiple output formats: pretty-printed, JSON, and Waybar-compatible
- Web UI with JSON API (`llm-usage serve`)
- Interactive setup wizard and automatic Claude CLI credential migration
- Automatic Claude OAuth token refresh
- Visual progress bars showing usage utilization and reset times

## Supported Providers

| Provider | Status | Auth Type |
|----------|--------|-----------|
| Claude   | ✅ Implemented | OAuth (migrated from the Claude CLI, or manual tokens) |
| Kimi     | ✅ Implemented | API key |
| MiniMax  | ✅ Implemented | Cookie + Group ID |
| Z.AI     | 🔜 Planned | API key (credentials can be stored, usage fetching pending) |

## Installation

### From Source

```bash
go install github.com/denysvitali/llm-usage@latest
```

### From Releases

Download the appropriate binary for your platform from the [Releases](https://github.com/denysvitali/llm-usage/releases) page.

## Getting Started

Run the interactive setup wizard:

```bash
llm-usage setup
```

Or configure providers directly:

```bash
# Migrate Claude credentials from the Claude CLI (macOS Keychain or ~/.claude/.credentials.json)
llm-usage setup migrate-claude

# Add accounts (prompts for credentials; keys are not echoed)
llm-usage setup add claude
llm-usage setup add kimi
llm-usage setup add minimax --account work

# Manage accounts
llm-usage setup list
llm-usage setup rename kimi default personal
llm-usage setup remove kimi personal
```

For Claude, the easiest path is having the [Claude CLI](https://github.com/anthropics/claude-code) installed and authenticated — its credentials are picked up automatically (macOS Keychain or `~/.claude/.credentials.json`). Expired tokens are refreshed automatically when a refresh token is available.

## Usage

```bash
# Show all configured providers (default)
llm-usage

# Show one or more specific providers
llm-usage --provider=claude
llm-usage --provider=claude,kimi

# Show a specific account
llm-usage --provider=kimi --account=work

# JSON output
llm-usage --json

# Waybar-compatible JSON output
llm-usage --waybar

# Include raw provider API responses
llm-usage --debug

# Show version
llm-usage --version
```

The command exits non-zero if every provider fails (except with `--waybar`, which always exits 0 so the bar module keeps rendering).

### Web UI

```bash
# Start the web server (default: http://localhost:8080)
llm-usage serve
llm-usage serve --host 0.0.0.0 --port 9090
```

### Configuration

Credentials are stored following the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html), in `$XDG_CONFIG_HOME/llm-usage/` (defaults to `~/.config/llm-usage/`):

- `claude.json` - Claude OAuth credentials
- `kimi.json` - Kimi API credentials
- `minimax.json` - MiniMax cookie + group ID credentials
- `zai.json` - Z.AI API credentials

Each file supports multiple named accounts.

#### Combined credentials file

Alternatively, pass a single credentials file whose values may reference environment variables (`$VAR` or `${VAR}`), useful for CI or secret managers:

```bash
llm-usage --credentials-file=creds.json
```

```json
{
  "claude": { "accounts": { "default": { "accessToken": "$CLAUDE_TOKEN" } } },
  "kimi":   { "accounts": { "default": { "apiKey": "$KIMI_API_KEY" } } }
}
```

### Example Output

```
LLM Usage Statistics
====================

Claude (Anthropic):
--------------------------------
  5-Hour Window:
    Usage:    [████████████░░░░░░░░] 60.0%
    Resets:   in 2h 15m
  7-Day Window:
    Usage:    [██████░░░░░░░░░░░░░░] 30.0%
    Resets:   in 3d 5h

Kimi (default):
--------------------------------
  Daily Usage:
    Usage:    [████░░░░░░░░░░░░░░░░] 20.0%
```

### Waybar Integration

Add this to your Waybar config:

```json
{
  "custom/llm-usage": {
    "exec": "llm-usage --waybar",
    "return-type": "json",
    "interval": 300
  }
}
```

## Building from Source

```bash
# Clone the repository
git clone https://github.com/denysvitali/llm-usage.git
cd llm-usage

# Build
make build

# Or install directly
make install
```

## Development

```bash
# Run linter
make lint

# Run tests
make test

# Build and test everything
make all
```

## License

MIT License - see [LICENSE](LICENSE) for details.
