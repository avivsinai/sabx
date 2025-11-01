# Repository Guidelines

## Project Structure & Module Organization
- `cmd/sabx/` — Cobra command tree; `main.go` compiles the CLI binary. Subpackages under `cmd/sabx/root/` group commands by SABnzbd feature (queue, history, config, rss, etc.).
- `internal/sabapi/` — typed HTTP client and models for the SABnzbd API; treat as the canonical integration layer.
- `internal/{config,auth,output}` — configuration loading, keyring integration, and output formatting helpers shared by commands.
- `internal/extensions` — manages the `sabx extension` lifecycle (install/list/remove + PATH discovery).
- `internal/ui/top` — Bubble Tea dashboard powering `sabx top`.
- `go.mod` / `go.sum` — Go module definition; keep dependencies minimal and reproducible.

## Build, Test, and Development Commands
- `go build ./...` — compile the entire module to ensure the CLI and helpers stay healthy.
- `go test ./...` — execute all Go tests; add coverage near integration points.
- `go run ./cmd/sabx --help` — quick local smoke test of the CLI surface.

## Coding Style & Naming Conventions
- Go source: format with `gofmt` (tabs for indentation, standard Go style). Run `go fmt ./...` before submitting.
- Package naming follows Go norms (short, lower-case). Files under `cmd/sabx/root` are named after verbs (e.g., `queue.go`).
- Prefer descriptive function names mirroring SABnzbd operations (`QueueSetPriority`, `RSSList`). Export only what downstream commands require.

## Testing Guidelines
- Use Go’s built-in `testing` package; table-driven tests for client behaviours are encouraged.
- Place tests alongside code (`*_test.go`) and name test functions `Test<Subject>`.
- Mock HTTP interactions with `httptest.Server` when covering `internal/sabapi`; keep responses aligned with the SABnzbd API schema.

## Commit & Pull Request Guidelines
- Commit messages: short imperative subject (`Add queue purge command`), optional body for context. Group logically related changes; avoid mixed-format commits.
- Pull requests should include: summary of changes, testing evidence (`go test ./...` output), any SABnzbd version assumptions, and linked issue/ticket IDs when available.
- Screenshots or CLI transcripts are helpful for UX-altering changes.

## Security & Configuration Tips
- Never log SABnzbd API keys; the CLI stores secrets via the OS keyring. When adding diagnostics, redact sensitive fields.
- Respect profile-aware configuration; updates should pass through `internal/config` to avoid clobbering user data.
