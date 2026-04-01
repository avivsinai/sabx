# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.1.7] - 2026-04-01

### Fixed

- Treated `Version already exists` as success when a skill publish reruns after retrying without an alias, preventing false-negative publish failures on release reruns.

## [0.1.6] - 2026-04-01

### Changed

- Simplified the GitHub release workflow to rely on `actions/setup-go`'s built-in caching instead of a separate cache step, reducing failure modes during tag reruns.

## [0.1.5] - 2026-04-01

### Fixed

- Serialized macOS keychain reads and writes behind an inter-process lock to prevent prompt storms when multiple `sabx` processes access the same API key concurrently.
- Added stable release metadata verification so skill/plugin versions must match the release tag before publishing binaries or skills.

### Changed

- Ad-hoc signed local macOS builds and Homebrew-installed binaries with the stable identifier `io.github.avivsinai.sabx` so Keychain approvals survive upgrades.

## [0.1.2] - 2026-02-21

### Features

- Add plugin manifest and agent-scoped skills
- Add sabx skill for Claude Code and Codex (#2)

### Miscellaneous

- Migrate golangci-lint from v1 to v2 (#12)
- Bump Go module dependencies: keyring, cobra, viper, testcontainers (#6)
- Bump GitHub Actions: checkout v6, cache v5, upload-artifact v6, setup-node v6 (#4, #5, #7, #8)
- Add dependabot for Go modules and GitHub Actions (#3)
- Add skild.sh and skills.sh installation methods
- Align skill config with amq-cli pattern
- Add check-skills to CI workflow
- Prevent release race condition with concurrency control
- Remove scheduled gitleaks scan, keep push/PR triggers

## [0.1.1] - 2026-01-18

### Bug Fixes

- Update goreleaser config for v2 schema compatibility
- Improve extensions and redact API key

### Miscellaneous

- Add gitleaks secret scanning and pre-commit hooks
- Align tooling with module path
- Remove prerelease flag from release badge

## [0.1.0] - 2025-11-03

### Features

- Initial release
- Full SABnzbd API parity (queue, history, config, RSS, admin)
- Keyring-backed authentication with profile support
- Interactive TUI dashboard (`sabx top`)
- Extension system for custom commands
- JSON/YAML/table output modes with `--jq` support
- Smoke tests with Testcontainers
- Homebrew, Scoop, and binary distribution

### Bug Fixes

- Correct module path to match repository URL
- Respect SABX_ALLOW_INSECURE_STORE env var in all commands
- Skip keyring deletion for config-stored API keys
- Allow commands without config when flags provided
