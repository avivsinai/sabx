# sabx

[![CI](https://github.com/sabx/sabx/actions/workflows/ci.yml/badge.svg)](https://github.com/sabx/sabx/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/sabx/sabx)](https://go.dev/)
[![License](https://img.shields.io/github/license/sabx/sabx)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/sabx/sabx)](https://goreportcard.com/report/github.com/sabx/sabx)
[![Release](https://img.shields.io/github/v/release/sabx/sabx)](https://github.com/sabx/sabx/releases/latest)

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

# Review full system diagnostics
sabx status --full --performance

# Check runtime warnings and logs
sabx warnings list
sabx logs list --lines 50

# Inspect live speed state for scripting
sabx speed status --json

# Pause post-processing while troubleshooting
sabx postprocess pause

# Test a news server definition
sabx server test primary

# Force-prioritize a download
sabx queue item priority <nzo_id> 2

# Explore SAB host filesystem and watched folder automation
sabx browse / --files --json
sabx watched scan --json

# Reset quota counters safely
sabx quota reset

# Smoke-test notifications and sort helpers
sabx notifications test email --json
sabx debug eval-sort "%sn - S%0sE%0e" --job "Example.Show" --json

# Manage RSS feeds
sabx rss add TVFeed --url https://example.org/rss --cat tv
sabx rss run TVFeed

# Update scheduler to pause nightly
sabx schedule set NightPause --set command=pause --set day=mon-sun --set hour=01 --set min=00

# Launch the live dashboard
sabx top
```

## Configuration & Profiles
- Config file: `~/Library/Application Support/sabx/config.yaml` (macOS), `%APPDATA%\sabx\config.yaml` (Windows), `~/.config/sabx/config.yaml` (Linux) - auto-created.
- Credentials stored in macOS Keychain / Windows Credential Manager / Secret Service via [`go-keyring`](https://github.com/zalando/go-keyring).
- Override per invocation with `--profile`, `--base-url`, `--api-key`, or env vars `SABX_BASE_URL`, `SABX_API_KEY`.

## Command Reference
Run `sabx <command> --help` for details. Key groups mirror the SABnzbd UI:
- `queue`: add, prioritize, move, purge, and edit job metadata.
- `history`: filter, delete, and `retry` completed jobs.
- `rss`, `categories`, `schedule`: full CRUD against named config sections.
- `config`: generic `get`, `set`, and `delete` for any SABnzbd config section.
- `server`: list, inspect stats, connectivity test, disconnect/unblock, restart/shutdown.
- `postprocess`: pause/resume global PP or cancel specific NZO IDs.
- `speed`: view current speed (`status`) and adjust the global limit.
- `browse`: inspect SABnzbd-side filesystem paths.
- `watched`: trigger watched-folder rescans.
- `quota`: reset download quota counters.
- `notifications`: run email/pushover/desktop test hooks.
- `debug`: fetch GC stats or evaluate sort expressions.
- `translate`: resolve SABnzbd UI translation keys.
- `warnings`: list and clear SABnzbd runtime warnings.
- `logs`: fetch sanitized SABnzbd logs (`list`, `tail` with optional follow).
- `scripts`: inspect available post-processing scripts.
- `dump`: export sanitized configuration or live state snapshots.
- `top`: Bubble Tea dashboard for real-time queue and history monitoring.
- `extension`: install/list/remove `sabx-<name>` extensions (GitHub repos or local).
- `doctor`: connectivity & health checks.

## API Parity Checklist

| SABnzbd Area | API mode(s) | `sabx` coverage |
| --- | --- | --- |
| Queue & Adds | `queue`, `addurl`, `addfile`, `addlocalfile`, `switch`, `sort`, `change_cat`, `change_script` | `queue list`, `queue add url|file|local`, `queue item move`, `queue item set`, `queue sort` |
| Queue File Ops | `get_files`, `move_nzf_bulk`, `delete_nzf` | `queue item files`, `queue item files move`, `queue item files delete` |
| History & Retries | `history`, `retry`, `retry_all`, `history.mark_as_completed` | `history list`, `history retry`, `history mark-completed` |
| Status & Diagnostics | `status`, `fullstatus`, `warnings`, `showlog`, `server_stats` | `status [--full]`, `warnings list|clear`, `logs list|tail`, `server stats` |
| Post-Processing | `pause_pp`, `resume_pp`, `cancel_pp` | `postprocess pause|resume|cancel` |
| Speed Control | `status`, `queue`, `speedlimit`, `set_pause` | `speed status`, `speed limit`, `config set-pause` |
| Servers | `get_config(section=servers)`, `config`, `disconnect`, `status.unblock_server`, `restart_repair` | `server list|stats|test|disconnect|unblock|restart|repair` |
| RSS & Schedule | `rss_*`, `schedule_*` | `rss list|add|set|delete|run`, `schedule list|add|set|delete` |
| Config & Keys | `config`, `get_config`, `set_config`, `del_config`, `set_apikey`, `set_nzbkey`, `regenerate_certs`, `create_backup`, `purge_log_files`, `set_config_default` | `config get|set|delete`, `config rotate-api-key|rotate-nzb-key|regenerate-certs|backup|purge-logs|reset-default` |
| Notifications | `test_email`, `test_pushover`, `test_apprise`, `test_notif`, `test_osd`, `test_windows`, `test_pushbullet`, `test_prowl`, `test_nscript` | `notifications test <type>` |
| Filesystem & Watchers | `browse`, `watched_now` | `browse`, `watched scan` |
| Quota & Usage | `reset_quota`, `gc_stats`, `server_stats` | `quota reset`, `debug gc-stats`, `server stats` |
| Extensions & Automation | `translate`, `eval_sort`, `dump`, `extension` hooks | `translate`, `debug eval-sort`, `dump config|state`, `extension list|install|remove` |

## Smoke Tests

Use the bundled harness to exercise the tricky endpoints against a live SABnzbd instance and capture fixtures for regression testing:

```bash
scripts/run-smoke.sh --base-url http://localhost:8080 --api-key $SAB_API_KEY --output testdata/smoke/latest
```

The script builds `sabx`, runs a curated set of commands (`browse`, `notifications test`, `debug eval-sort`, `status orphans`, `watched scan`, etc.), validates JSON, and writes sanitized outputs to `testdata/smoke/<command>.json` plus an aggregated `report.json`. Review the artifacts before committing to avoid leaking instance-specific secrets.

## Development
```bash
# Format & vet
go fmt ./...
go test ./...

# End-to-end smoke (requires Docker)
go test ./test/e2e -run TestSmokeAgainstSABContainer -count=1

# Run the CLI locally
./sabx status
```
See [AGENTS.md](AGENTS.md) for deeper contributor guidance, including architecture notes and testing tips. Release automation and tagging instructions live in [docs/RELEASING.md](docs/RELEASING.md).

Set `SABX_E2E_DISABLE=1` to skip container-based smoke tests when Docker is unavailable.

## Extensions
- Install from GitHub: `sabx extension install avivsinai/sabx-tv-tools`
- Execute: once installed, run `sabx tv-tools ...` and the CLI will forward arguments to the `sabx-tv-tools` binary/script.
- List or remove extensions with `sabx extension list` and `sabx extension remove <name>`.

Extensions live under `~/.sabx/extensions` and can also be distributed by placing `sabx-<name>` executables on `PATH`.
