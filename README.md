# claude-code-usage

A CLI tool to display your Claude AI API usage statistics for Pro and Max subscriptions.

## Features

- View real-time usage statistics from your Claude subscription
- Multiple output formats: pretty-printed, JSON, and Waybar-compatible
- Visual progress bars showing usage utilization
- Displays reset times for usage windows

## Prerequisites

You must have the [Claude CLI](https://github.com/anthropics/claude-code) installed and authenticated. The tool reads OAuth credentials from `~/.claude/.credentials.json`, which is created when you run the `claude` command for the first time.

## Installation

### From Source

```bash
go install github.com/denysvitali/claude-code-usage/cmd/claude-usage@latest
```

### From Releases

Download the appropriate binary for your platform from the [Releases](https://github.com/denysvitali/claude-code-usage/releases) page.

## Usage

```bash
# Pretty-printed output (default)
claude-usage

# JSON output
claude-usage --json

# Waybar-compatible JSON output
claude-usage --waybar

# Show version
claude-usage --version
```

### Example Output

```
Claude Usage (Pro/Max Subscription)
====================================

5-Hour Window:
  Usage:    ████████████░░░░░░░░  60.0%
  Resets:   in 2h 15m

7-Day Window:
  Usage:    ██████░░░░░░░░░░░░░░  30.0%
  Resets:   in 3d 5h

Token expires: 23h 45m
```

### Waybar Integration

Add this to your Waybar config:

```json
{
  "custom/claude": {
    "exec": "claude-usage --waybar",
    "return-type": "json",
    "interval": 300
  }
}
```

## Building from Source

```bash
# Clone the repository
git clone https://github.com/denysvitali/claude-code-usage.git
cd claude-code-usage

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
