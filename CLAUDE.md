# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`llm-usage` is a Go CLI tool that displays LLM API usage statistics across multiple providers (Claude, Kimi, MiniMax, Z.AI). It supports multiple output formats (pretty-printed, JSON, Waybar-compatible), a web UI with embedded API, and manages credentials with platform-specific keychain integration.

## Build & Development Commands

```bash
# Build the binary
make build

# Run all checks: lint, test, build
make all

# Run tests with race detection
make test

# Run a single test
go test -v -run TestName ./internal/cache/

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

**`main.go`** (root directory) - Entry point that calls `cmd.Execute()`. The actual CLI logic lives in the `cmd/` package.

**`cmd/`** - Contains Cobra commands:
- `cmd/root.go` - Main usage command with provider/account flags and output formatting options
- `cmd/serve.go` - Web server subcommand (start HTTP server with UI and JSON API)
- `cmd/setup*.go` - Setup subcommands (add, list, remove, rename, migrate)

### Package Structure

```
cmd/               # Cobra CLI commands
├── root.go       # Main usage command
├── serve.go      # Web server subcommand
├── setup.go      # Setup parent command
├── setup_add.go      # Add account
├── setup_list.go     # List accounts
├── setup_remove.go   # Remove account
├── setup_rename.go   # Rename account
└── setup_migrate.go  # Migrate credentials

internal/
├── cache/        # File-based TTL caching
│   ├── cache.go  # Cache manager with TTL support
│   └── cache_test.go
├── credentials/ # Credential management (multi-account support)
│   ├── loader.go # Loads provider credentials from $XDG_CONFIG_HOME/llm-usage/
│   └── manager.go # Manages credential storage and retrieval
├── keychain/    # Platform-specific secure storage
│   ├── keychain_darwin.go    # macOS implementation
│   └── keychain_fallback.go  # Linux/Windows fallback
├── provider/    # Provider abstraction layer
│   ├── provider.go  # Provider interface and Usage types
│   ├── claude/  # Claude provider implementation (OAuth)
│   ├── kimi/    # Kimi provider implementation (API key)
│   ├── minimax/ # MiniMax provider implementation (cookie + group ID)
│   └── zai/     # Z.AI stub (not implemented)
├── serve/       # Web server + embedded UI
│   ├── server.go # HTTP server with JSON API
│   └── web/      # Static assets embedded via go:embed
├── setup/       # Setup commands
│   ├── setup.go     # CLI setup commands (add, list, remove, rename, migrate)
│   └── tui/         # Interactive TUI setup wizard (Bubble Tea)
├── usage/       # Provider loading, concurrent fetching, output formatting
│   ├── errors.go  # Error types
│   ├── output.go  # Output formatters (pretty, JSON, Waybar)
│   └── providers.go # Provider loading and concurrent fetching
└── version/     # Version information (injected at build time)
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

### Data Flow

1. CLI entry point (`cmd/root.go`) parses flags and calls `usage.GetProviders()`
2. `usage.GetProviders()` loads credentials from `$XDG_CONFIG_HOME/llm-usage/` and instantiates providers
3. `usage.FetchAllUsage()` fetches usage from all providers concurrently
4. Output is formatted (pretty, JSON, or Waybar) and printed to stdout

### Credential Management

Credentials are stored in `$XDG_CONFIG_HOME/llm-usage/` (defaults to `~/.config/llm-usage/`) as JSON files:
- `claude.json` - OAuth credentials (multi-account support)
- `kimi.json` - API key credentials (multi-account support)
- `minimax.json` - Cookie + group ID credentials (multi-account support)
- `zai.json` - API key credentials (multi-account support)

The app also supports migration from Claude CLI credentials at `~/.claude/.credentials.json`.

### Multi-Account Support

Each provider supports multiple named accounts. The `--account` flag filters to a specific account, `--all-accounts` aggregates usage across all accounts of a provider, and omitting both shows all configured providers/accounts.

The `--credentials-file` flag on `cmd/root.go` accepts a path to a combined credentials file where values may reference environment variables via `$VAR` or `${VAR}` syntax, expanded at load time.

## Key Dependencies

- **Cobra** (`github.com/spf13/cobra`) - CLI framework
- **Bubble Tea** (`github.com/charmbracelet/bubbletea`) - TUI framework for setup wizard
- **Lip Gloss** (`github.com/charmbracelet/lipgloss`) - TUI styling
- **XDG** (`github.com/adrg/xdg`) - XDG Base Directory support for config paths

## Provider Status

| Provider | Status | Auth Type |
|----------|--------|-----------|
| Claude   | Fully implemented | OAuth (from Claude CLI or manual) |
| Kimi     | Fully implemented | API Key |
| MiniMax  | Fully implemented | Cookie + Group ID |
| Z.AI     | Not implemented (stub) | API Key |

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
