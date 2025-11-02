# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

---

## ⚠️ CRITICAL: READ AGENTS.MD FIRST

**BEFORE doing ANY work in this repository, you MUST read `AGENTS.md` located at the root of this project.**

`AGENTS.md` is the **master agent documentation** for this repository and contains authoritative guidelines for:
- Project structure and module organization
- Build, test, and development commands
- Coding style and naming conventions
- Testing guidelines
- Commit and pull request standards
- Security and configuration best practices

This file (CLAUDE.md) provides **supplementary** Claude Code-specific guidance on architecture and development patterns. Both documents work together:
- **AGENTS.md** = Master rules and standards (READ THIS FIRST)
- **CLAUDE.md** = Architecture deep-dive and development patterns (supplementary reference)

**Note:** This repository is also used by Codex and other AI agents. Always cross-reference AGENTS.md to ensure consistency across all agent interactions.

---

## Project Overview

sabx is a CLI tool for managing SABnzbd download servers. It uses Cobra for command structure, Viper for configuration, and integrates with OS keyrings for secure API key storage.

## Build, Test, and Development Commands

```bash
# Build the CLI binary
go build ./cmd/sabx

# Run all tests
go test ./...

# Quick smoke test
go run ./cmd/sabx --help

# Format code before committing
go fmt ./...

# Run the CLI with a specific profile
go run ./cmd/sabx --profile home status
```

## Architecture Overview

### Command Structure (Cobra-based)
- Entry point: `cmd/sabx/main.go` → `root.Execute()`
- Root command: `cmd/sabx/root/root.go` with `PersistentPreRunE` hook that:
  1. Loads config from `$SABX_CONFIG_DIR/config.yml` (defaults to `~/.config/sabx/config.yml`)
  2. Creates output `Printer` object
  3. Resolves connection details (baseURL, apiKey, profile)
  4. Creates SABnzbd API `Client`
  5. Wraps everything in `cobraext.App` struct
  6. Attaches `App` to command context
- Subcommands in `cmd/sabx/root/` organized by feature (queue, history, config, rss, etc.)

### Global Flags (Available on All Commands)
- `--profile`: Select SABnzbd profile (default: "default")
- `--base-url`: Override base URL
- `--api-key`: Override API key
- `--json`: Output as JSON
- `--quiet`: Suppress non-error output

### API Client (`internal/sabapi`)
- Single `Client` type with one `call()` method for all API interactions
- All requests go to `/api` endpoint with query parameters
- Custom `Boolish` type handles SABnzbd's inconsistent boolean representations (true/false, "true"/"false", 1/0, "yes"/"no")
- Envelope pattern for responses (e.g., `QueueEnvelope` wraps `QueueResponse`)
- Special handling for file uploads via multipart form in `AddFile()`
- Default timeout: 15 seconds

### Configuration System (`internal/config`)
- Storage: `config.yml` under `$SABX_CONFIG_DIR` (defaults to platform-appropriate config dir). Directory enforced at `0o700`, writes are atomic (`CreateTemp` + rename).
- Supports multiple profiles for different SABnzbd instances
- Thread-safe with `sync.RWMutex`
- Structure:
  ```yaml
  default_profile: "default"
  profiles:
    default:
      base_url: "http://localhost:8080"
      api_key: ""  # omit if using keyring
  ```

### Authentication (`internal/auth`)
- Uses `github.com/99designs/keyring` with a lightweight wrapper (`Store`) enabling native keychains and optional encrypted file fallback.
- Service name: `"sabx"`
- Keyring key format: `"profile/<name>/<sha256(baseURL)>` with human-readable labels for OS dialogs.
- Helpers: `SaveAPIKey()`, `LoadAPIKey()`, `DeleteAPIKey()` accept optional store options (e.g., `auth.WithAllowFileFallback(true)`).
- Fallbacks: enable encrypted file backend via `--allow-insecure-store` or plaintext config via `--store-in-config` (discouraged).

### Connection Resolution Priority (High to Low)
1. Command-line flags (`--base-url`, `--api-key`)
2. Environment variables (`SABX_BASE_URL`, `SABX_API_KEY`)
3. OS keyring (for API key)
4. Config file profile

### Output System (`internal/output`)
- `Printer` type with smart polymorphic `Print()` method:
  - Respects `--json` and `--quiet` flags
  - Handles strings, `fmt.Stringer`, and arbitrary structs
- `Table()` method for tabular output (uses `text/tabwriter` in human mode)
- Commands typically branch on `app.Printer.JSON` to provide format-specific output

### Context & Dependency Injection (`internal/cobraext`)
- `App` struct holds all command dependencies (Config, Client, Printer, ProfileName, BaseURL)
- Attached to `context.Context` via `WithApp()`
- Retrieved in commands via `getApp(cmd)` helper

## Common Command Pattern

```go
func myCmd() *cobra.Command {
    var flagVar string

    cmd := &cobra.Command{
        Use:   "mycommand [args]",
        Short: "Short description",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            app, err := getApp(cmd)
            if err != nil {
                return err
            }

            ctx, cancel := timeoutContext(cmd.Context())
            defer cancel()

            // Make API call
            result, err := app.Client.SomeOperation(ctx, args...)
            if err != nil {
                return err
            }

            // Format output
            if app.Printer.JSON {
                return app.Printer.Print(result)
            }
            // Build human-readable output
            return app.Printer.Print("Success!")
        },
    }

    cmd.Flags().StringVar(&flagVar, "flag", "", "Description")
    return cmd
}
```

For commands that don't need SABnzbd connection (login, logout, version, completion):
```go
Annotations: map[string]string{
    "skipPersistent": "true",
},
```

## Adding New Commands

1. Create new file in `cmd/sabx/root/` (e.g., `myfeature.go`)
2. Define command function returning `*cobra.Command`
3. Register in `init()` function: `rootCmd.AddCommand(myFeatureCmd())`
4. Use `getApp(cmd)` to access Client, Printer, Config
5. Use `timeoutContext(cmd.Context())` for API calls
6. Provide both JSON and human-readable output paths

## Adding New API Methods

1. Define request/response structs in `internal/sabapi/client.go`
2. Add method to `Client` type
3. Use `c.call(ctx, "mode_name", params, &response)` pattern
4. For file uploads, follow `AddFile()` multipart pattern
5. Handle envelope responses if SABnzbd returns nested JSON

## Testing Guidelines

- Use Go's `testing` package with table-driven tests
- Place tests alongside code (`*_test.go`)
- Mock SABnzbd API with `httptest.Server`
- Keep responses aligned with actual SABnzbd API schema
- Focus coverage on API client methods and command handlers

## Code Style

- Standard Go formatting (`gofmt` with tabs)
- Descriptive function names mirroring SABnzbd operations (e.g., `QueueSetPriority`, `RSSList`)
- Export only what downstream commands require
- Package naming: short, lowercase (follow Go norms)

## Security Considerations

- Never log API keys; redact in diagnostics
- Keyring storage is default; insecure config storage is opt-in
- Config files created with mode 0o600 (owner read/write only)
- Respect profile-aware configuration to avoid data corruption

## Key Files Reference

- `cmd/sabx/main.go` - CLI entry point
- `cmd/sabx/root/root.go` - Root command and `PersistentPreRunE` setup
- `internal/sabapi/client.go` - Complete API client implementation
- `internal/config/config.go` - Configuration loading and profile management
- `internal/auth/keyring.go` - Secure API key storage
- `internal/output/printer.go` - Output formatting system
- `internal/cobraext/context.go` - Context-based dependency injection

## Common Helper Functions (in `cmd/sabx/root/common.go`)

- `profileOrDefault(profile string)` - Returns "default" if empty
- `firstNonEmpty(values ...string)` - First non-whitespace string
- `priorityLabel(priority string)` - Maps priority codes to labels
- `buildAddOptions(...)` - Parses flags to `AddOptions` struct
