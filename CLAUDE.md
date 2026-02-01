# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`llm-usage` is a Go CLI tool that displays LLM API usage statistics across multiple providers (Claude, Kimi, Z.AI). It supports multiple output formats (pretty-printed, JSON, Waybar-compatible) and manages credentials with platform-specific keychain integration.

## Build & Development Commands

```bash
# Build the binary (builds from ./main.go, not ./cmd/)
make build

# Run all checks: lint, test, build
make all

# Run tests with race detection
make test

# Run linter (golangci-lint with extensive config)
make lint

# Format code (gofmt + goimports)
make fmt

# Generate coverage report
make coverage

# Install to GOPATH/bin
make install

# Create release (requires GITHUB_TOKEN)
make release

# Create snapshot release (for testing)
make snapshot
```

## Architecture

### Entry Point

**`main.go`** (root directory, not in `cmd/`) - The main CLI application using Go's `flag` package. Handles:
- Command-line flag parsing
- Provider selection and filtering
- Credential loading from multiple sources
- Output formatting (pretty, JSON, Waybar)
- Setup subcommand handling

### Package Structure

```
internal/
├── credentials/     # Credential management (multi-account support)
│   ├── loader.go    # Loads provider credentials from $XDG_CONFIG_HOME/llm-usage/
│   └── manager.go   # Manages credential storage and retrieval
├── keychain/        # Platform-specific secure storage
│   ├── keychain_darwin.go     # macOS implementation
│   └── keychain_fallback.go   # Linux/Windows fallback
├── provider/        # Provider abstraction layer
│   ├── provider.go  # Provider interface and Usage types
│   ├── claude/      # Claude provider implementation (OAuth)
│   ├── kimi/        # Kimi provider implementation (API key)
│   └── zai/         # Z.AI provider implementation (API key)
├── setup/           # Setup commands
│   ├── setup.go     # CLI setup commands (add, list, remove, rename, migrate)
│   └── tui/         # Interactive TUI setup wizard (Bubble Tea)
└── version/         # Version information (injected at build time)
```

### Provider Interface

The `provider.Provider` interface (`internal/provider/provider.go:9`) defines the contract for all LLM providers:

```go
type Provider interface {
    Name() string           // Display name
    ID() string             // Unique identifier
    GetUsage() (*Usage, error)
}
```

All providers implement this interface and return a `Usage` struct with:
- `Windows`: Usage time windows with utilization percentages
- `Extra`: Provider-specific metadata
- `Error`: Allows partial results on failure

### Credential Management

Credentials are stored in `$XDG_CONFIG_HOME/llm-usage/` (defaults to `~/.config/llm-usage/`) as JSON files:
- `claude.json` - OAuth credentials (multi-account support)
- `kimi.json` - API key credentials (multi-account support)
- `zai.json` - API key credentials (multi-account support)

The app also supports migration from Claude CLI credentials at `~/.claude/.credentials.json`.

### Multi-Account Support

Each provider supports multiple named accounts. The `--account` flag filters to a specific account, while omitting it shows all configured accounts aggregated.

## Key Dependencies

- **Bubble Tea** (`github.com/charmbracelet/bubbletea`) - TUI framework for setup wizard
- **Lip Gloss** (`github.com/charmbracelet/lipgloss`) - TUI styling
- No Cobra/Viper usage despite being in dependencies (legacy)

## Provider Status

| Provider | Status | Auth Type |
|----------|--------|-----------|
| Claude   | Fully implemented | OAuth (from Claude CLI or manual) |
| Kimi     | Fully implemented | API Key |
| Z.AI     | Fully implemented | API Key |

## Testing

Run tests with race detection:
```bash
make test
```

Coverage report:
```bash
make coverage
```

## Release Process

Releases are automated via Goreleaser:
- Cross-platform builds (Linux, macOS, Windows)
- Multiple architectures (amd64, arm64)
- SHA256 checksums generated automatically
- Changelog via git-cliff

Set version when building:
```bash
make build VERSION=1.2.3
```
