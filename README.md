# llm-usage

`llm-usage` is a Go CLI for checking subscription and quota usage across
multiple LLM providers. It is designed for both humans at a terminal and
small integrations such as Waybar, scripts, and a local HTTP server.

## What it does

- Queries Claude, Codex, Grok, Kimi, MiniMax, and Z.AI through one interface.
- Discovers existing Codex and Grok CLI sessions without asking you to copy
  tokens into another configuration file.
- Supports named accounts for providers that use llm-usage-managed
  credentials.
- Renders Lip Gloss terminal output, JSON, and Waybar-compatible JSON.
- Provides setup, diagnostics, shell completion, continuous refresh, and a
  local web server.
- Exposes reusable public provider packages for Go applications.

## Providers and authentication

| ID | Provider | Authentication | Usage |
| --- | --- | --- | --- |
| `claude` | Claude (Anthropic) | Claude CLI session, or managed OAuth credentials | Implemented |
| `codex` | Codex (OpenAI) | Codex CLI session in `~/.codex/auth.json` | Implemented |
| `grok` | Grok (xAI) | Grok CLI session in `~/.grok/auth.json` | Implemented |
| `kimi` | Kimi | Managed API key | Implemented |
| `minimax` | MiniMax | Managed cookie and group ID | Implemented |
| `zai` | Z.AI | Managed API key | Credential storage only |

Codex and Grok are automatically included when their local CLI sessions are
available. They do not need a `setup add` step. Claude credentials can be
read from the Claude CLI or migrated into llm-usage-managed storage.

## Install

From source:

```bash
go install github.com/denysvitali/llm-usage@latest
```

Or download a platform binary from the
[releases page](https://github.com/denysvitali/llm-usage/releases).

## Quick start

If you already use Codex or Grok locally, this is enough:

```bash
llm-usage
```

Select providers explicitly when you want a predictable result:

```bash
llm-usage --provider=codex,grok
llm-usage --provider=claude,kimi --timeout=10s
```

Useful checks:

```bash
llm-usage provider list
llm-usage doctor
llm-usage --help
```

The command returns a non-zero status when every selected provider fails.
Waybar output always returns zero so a temporary provider error does not stop
the bar module.

## CLI reference

### Query usage

```bash
# Human-readable terminal output
llm-usage

# One provider or a comma-separated selection
llm-usage --provider=codex
llm-usage --provider=claude,kimi

# Select a managed account
llm-usage --provider=kimi --account=work

# Query every account for a provider
llm-usage --provider=kimi --all-accounts

# Machine-readable output
llm-usage --json
llm-usage --waybar

# Include provider response details when debugging an integration
llm-usage --provider=codex --debug

# Bound slow provider requests
llm-usage --timeout=15s
```

The provider selector accepts `all` or these IDs: `claude`, `codex`, `grok`,
`kimi`, `minimax`, and `zai`.

### Configure managed credentials

Launch the interactive setup wizard:

```bash
llm-usage setup
```

Non-interactive account management commands:

```bash
llm-usage setup add claude
llm-usage setup add kimi --account=work
llm-usage setup add minimax --account=personal
llm-usage setup list
llm-usage setup list kimi
llm-usage setup rename kimi work home
llm-usage setup remove kimi home --yes
llm-usage setup migrate-claude
```

`setup migrate-claude` imports credentials from the Claude CLI when needed.
Codex and Grok should be authenticated with their own CLIs instead.

### Configuration and diagnostics

```bash
llm-usage config init
llm-usage config path
llm-usage config validate
llm-usage config --file ./llm-usage.yaml validate
llm-usage doctor
```

The default configuration directory is `$XDG_CONFIG_HOME/llm-usage`, or
`~/.config/llm-usage` when `XDG_CONFIG_HOME` is unset. Use
`--credentials-file` to load a combined credentials file, which is useful for
CI and secret managers:

```bash
llm-usage --credentials-file ./credentials.json --provider=kimi
```

Values in that file may reference environment variables with `$VAR` or
`${VAR}`. Do not commit the file.

### Watch and serve

Refresh the terminal view continuously:

```bash
llm-usage watch
llm-usage watch --provider=codex,grok --interval=2m
```

Start the local web UI and JSON API:

```bash
llm-usage serve
llm-usage serve --host=127.0.0.1 --port=9090
```

The default server address is `http://localhost:8080`. Use `--web-dir` when
the web assets are stored outside the repository.

### Shell completion

```bash
llm-usage completion zsh > "${fpath[1]}/_llm-usage"
llm-usage completion bash > /etc/bash_completion.d/llm-usage
llm-usage completion fish > ~/.config/fish/completions/llm-usage.fish
llm-usage completion powershell > llm-usage.ps1
```

## Waybar

Use separate modules when you want each provider to have its own icon, color,
and status. The Waybar output contains `text`, `tooltip`, and a CSS class.

```json
{
  "modules-center": ["clock", "custom/codex-usage", "custom/grok-usage"],
  "custom/codex-usage": {
    "exec": "llm-usage --provider=codex --waybar",
    "return-type": "json",
    "interval": 300,
    "escape": false,
    "tooltip": true,
    "format": "{}"
  },
  "custom/grok-usage": {
    "exec": "llm-usage --provider=grok --waybar",
    "return-type": "json",
    "interval": 300,
    "escape": false,
    "tooltip": true,
    "format": "{}"
  }
}
```

Codex displays its 5-hour and 7-day windows in order. Grok displays its
weekly window. Use `llm-usage --waybar` without `--provider` for one combined
module instead.

## JSON output

`--json` emits normalized provider reports. Each report contains a provider
ID, zero or more usage windows, optional provider-specific `extra` data, and a
normalized error when the provider is unavailable.

```json
{
  "providers": [
    {
      "provider": "codex",
      "windows": [
        {"label": "5-Hour", "utilization": 8, "resets_at": "..."},
        {"label": "7-Day", "utilization": 17, "resets_at": "..."}
      ]
    }
  ]
}
```

Provider failures are represented in the response instead of being mixed into
the normal usage text. This makes the output safe to consume from scripts.

## Go library

The normalized types live in the public `provider` package. Provider-specific
clients are available under `providers/<name>` and accept caller-owned context,
credentials, and HTTP clients where applicable.

```go
package main

import (
	"context"
	"fmt"

	"github.com/denysvitali/llm-usage/providers/grok"
)

func main() {
	client, err := grok.NewClient(grok.ClientOptions{AccessToken: "token"})
	if err != nil {
		panic(err)
	}
	usage, err := client.GetUsage(context.Background())
	if err != nil {
		panic(err)
	}
	fmt.Println(usage.Windows)
}
```

The public registry in `providers` exposes provider capabilities and the
application-facing loading contract. Credential discovery remains internal to
the CLI so reusable clients do not depend on local config files.

## Development

Requirements: Go 1.23 or newer.

```bash
git clone https://github.com/denysvitali/llm-usage.git
cd llm-usage

make fmt
make test
make lint
make build
```

Run the complete local verification target with:

```bash
make all
```

Build artifacts such as `llm-usage`, coverage reports, credentials, and local
CLI session files should remain uncommitted.

## License

MIT. See [LICENSE](LICENSE).
