# CLAUDE.md

This is the master agent instruction file for this repository. Keep repository policy here. `AGENTS.md` exists only as a Codex compatibility shim and should contain only Codex-specific notes.

## Project Overview

sabx is a CLI tool for managing SABnzbd download servers. It uses Cobra for command structure, Viper for configuration, and OS keyrings for secure API key storage.

## Release Contract

- Release from `main` only; do not create manual GitHub releases.
- A push to `main` updates the AvivSinai marketplace immediately for the `sabx` skill.
- For a versioned release, keep `CHANGELOG.md` and skill/plugin metadata on one version, then push the tag and let CI publish the GitHub release.

## Build, Test, and Development Commands

```bash
go build ./cmd/sabx        # Build the CLI
go test ./...             # Run the full test suite
go run ./cmd/sabx --help  # Quick local smoke test
go fmt ./...              # Format before committing
go run ./cmd/sabx --profile home status
```

## Repository Layout

- `cmd/sabx/` contains the Cobra command tree and CLI entry point.
- `cmd/sabx/root/` groups commands by SABnzbd feature such as queue, history, config, and RSS.
- `internal/sabapi/` is the typed SABnzbd API client and canonical integration layer.
- `internal/config`, `internal/auth`, and `internal/output` provide shared configuration, keyring, and output helpers.
- `internal/extensions` manages the `sabx extension` lifecycle.
- `internal/ui/top` contains the Bubble Tea dashboard used by `sabx top`.

## Architecture Overview

### Command Structure

- Entry point: `cmd/sabx/main.go` -> `root.Execute()`.
- `cmd/sabx/root/root.go` uses `PersistentPreRunE` to load config, create a printer, resolve connection details, create the client, wrap everything in `cobraext.App`, and attach it to the command context.
- Commands that do not need a SABnzbd connection should set `Annotations: map[string]string{"skipPersistent": "true"}`.

### Global Flags

- `--profile` selects the SABnzbd profile.
- `--base-url` overrides the base URL.
- `--api-key` overrides the API key.
- `--json` enables JSON output.
- `--quiet` suppresses non-error output.

### API Client

- `internal/sabapi.Client` uses one `call()` path for the API surface.
- Requests go to `/api` with query parameters.
- `Boolish` normalizes SABnzbd’s inconsistent boolean values.
- File uploads should follow the multipart `AddFile()` pattern.

### Configuration and Auth

- Config lives under `$SABX_CONFIG_DIR/config.yml`, defaulting to the platform config dir.
- Writes are atomic and the directory is enforced at `0o700`.
- API keys default to OS keyrings via `github.com/99designs/keyring`, with optional encrypted file fallback.
- Resolution order is flags, environment, keyring, then config profile.

### Output and Dependency Injection

- `internal/output.Printer` handles JSON, quiet, string, and structured output.
- `internal/cobraext.App` carries Config, Client, Printer, ProfileName, and BaseURL through `context.Context`.
- Commands should use `getApp(cmd)` and `timeoutContext(cmd.Context())`.

## Testing Guidelines

- Use Go’s `testing` package with table-driven tests.
- Place tests alongside code in `*_test.go`.
- Mock SABnzbd HTTP interactions with `httptest.Server`.
- Keep API fixtures aligned with the real SABnzbd schema.

## Code Style and Contribution Notes

- Use standard Go formatting with `gofmt`.
- Prefer descriptive function names mirroring SABnzbd operations such as `QueueSetPriority` or `RSSList`.
- Keep commits focused and imperative.
- User-facing CLI changes should include testing evidence and CLI output examples when useful.

## Security Considerations

- Never log API keys; redact them in diagnostics.
- Keyring storage is the default; config-file secret storage is opt-in.
- Respect profile-aware configuration to avoid corrupting user data.
