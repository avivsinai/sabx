# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

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
