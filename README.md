# sabx

> A batteries-included SABnzbd CLI and automation toolkit inspired by modern OSS CLIs like [`gh`](https://github.com/cli/cli) and [`chezmoi`](https://github.com/twpayne/chezmoi).

`sabx` mirrors the complete SABnzbd surface area—queue control, RSS management, scheduler, configuration, and server administration—in a fast Go binary designed for power users and automation.

## Highlights
- **Full parity** with SABnzbd REST API: queue history, RSS CRUD, scheduler, server actions, priorities, speed limits, and diagnostics.
- **First-class UX**: human-readable tables by default, `--json` for scripting, shell completions, and keyring-backed credential storage.
- **Agent-friendly**: deterministic output, idempotent commands, and profile-aware configuration ideal for CI/CD or LLM agents.

## Installation
```bash
# Go 1.22+
go install github.com/sabx/sabx/cmd/sabx@latest

# Build from source
git clone https://github.com/sabx/sabx.git
cd sabx
go build ./cmd/sabx
./sabx --help
```

Pre-built archives, Homebrew, Scoop, winget manifests, and multi-arch Docker images are produced via [GoReleaser](tools/goreleaser.yaml) on tagged releases.

## Quickstart
```bash
# Authenticate with a SABnzbd instance (stores API key in OS keyring)
sabx login --base-url http://localhost:8080 --api-key <key>

# Inspect the active queue
sabx queue list --active

# Force-prioritize a download
sabx queue item priority <nzo_id> 2

# Manage RSS feeds
sabx rss add TVFeed --url https://example.org/rss --cat tv
sabx rss run TVFeed

# Update scheduler to pause nightly
sabx schedule set NightPause --set command=pause --set day=mon-sun --set hour=01 --set min=00

# Launch the live dashboard
sabx top
```

## Configuration & Profiles
- Config file: `~/.config/sabx/config.yaml` (auto-created).
- Credentials stored in macOS Keychain / Windows Credential Manager / Secret Service via [`go-keyring`](https://github.com/zalando/go-keyring).
- Override per invocation with `--profile`, `--base-url`, `--api-key`, or env vars `SABX_BASE_URL`, `SABX_API_KEY`.

## Command Reference
Run `sabx <command> --help` for details. Key groups mirror the SABnzbd UI:
- `queue`: add, prioritize, move, purge, and edit job metadata.
- `history`: filter, delete, and `retry` completed jobs.
- `rss`, `categories`, `schedule`: full CRUD against named config sections.
- `config`: generic `get`, `set`, and `delete` for any SABnzbd config section.
- `server` & `speed`: restart/shutdown and global speed limit controls.
- `dump`: export sanitized configuration or live state snapshots.
- `top`: Bubble Tea dashboard for real-time queue and history monitoring.
- `extension`: install/list/remove `sabx-<name>` extensions (GitHub repos or local).
- `doctor`: connectivity & health checks.

## Development
```bash
# Format & vet
go fmt ./...
go test ./...

# Run the CLI locally
./sabx status
```
See [AGENTS.md](AGENTS.md) for deeper contributor guidance, including architecture notes and testing tips.

## Extensions
- Install from GitHub: `sabx extension install avivsinai/sabx-tv-tools`
- Execute: once installed, run `sabx tv-tools ...` and the CLI will forward arguments to the `sabx-tv-tools` binary/script.
- List or remove extensions with `sabx extension list` and `sabx extension remove <name>`.

Extensions live under `~/.sabx/extensions` and can also be distributed by placing `sabx-<name>` executables on `PATH`.
