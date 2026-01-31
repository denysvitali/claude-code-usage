# llm-usage

A CLI tool to display your LLM API usage statistics across multiple providers (Claude, Kimi, Z.AI).

## Features

- View real-time usage statistics from multiple LLM providers
- Multiple output formats: pretty-printed, JSON, and Waybar-compatible
- Visual progress bars showing usage utilization
- Displays reset times for usage windows
- Filter by provider or view all at once

## Prerequisites

For Claude usage, you must have the [Claude CLI](https://github.com/anthropics/claude-code) installed and authenticated.

## Installation

### From Source

```bash
go install github.com/denysvitali/llm-usage@latest
```

### From Releases

Download the appropriate binary for your platform from the [Releases](https://github.com/denysvitali/llm-usage/releases) page.

## Usage

```bash
# Show all configured providers (default)
llm-usage

# Show specific provider
llm-usage --provider=claude
llm-usage --provider=kimi
llm-usage --provider=zai

# JSON output
llm-usage --json

# Waybar-compatible JSON output
llm-usage --waybar
```

### Configuration

Credentials are stored in `~/.llm-usage/` with separate files per provider:

- `~/.llm-usage/claude.json` - Claude OAuth credentials
- `~/.llm-usage/kimi.json` - Kimi API credentials
- `~/.llm-usage/zai.json` - Z.AI API credentials

#### Migrating from claude-code-usage

```bash
# Create the config directory
mkdir -p ~/.llm-usage

# Copy existing Claude credentials
cp ~/.claude/.credentials.json ~/.llm-usage/claude.json
```

### Example Output

```
LLM Usage Statistics
====================

Claude (Pro/Max Subscription):
--------------------------------
  5-Hour Window:
    Usage:    [████████████░░░░░░░░] 60.0%
    Resets:   in 2h 15m
  7-Day Window:
    Usage:    [██████░░░░░░░░░░░░░░] 30.0%
    Resets:   in 3d 5h

Kimi:
--------------------------------
  Error: API not yet configured or not implemented

Z.AI:
--------------------------------
  Daily Usage:
    Usage:    [████░░░░░░░░░░░░░░░] 20.0%
    Limit:    1000000 tokens
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

## Supported Providers

| Provider | Status | Notes |
|----------|--------|-------|
| Claude | ✅ Implemented | Requires Claude CLI OAuth credentials |
| Kimi | 🔜 Planned | API endpoint identified, implementation pending |
| Z.AI | 🔜 Planned | API endpoint identified, implementation pending |

## License

MIT License - see [LICENSE](LICENSE) for details.
