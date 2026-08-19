# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Changed
- Publish the Homebrew formula to `avivsinai/homebrew-tap` via GoReleaser `brews` (same token and Formula layout as bitbucket-cli, jk, and amq).

### Fixed
- Documented the live install command `brew install avivsinai/tap/sabx` alongside `go install`, GitHub release archives, Docker, and source.
- Consolidated skill publishing into the release workflow so it no longer depends on tag-push events that `GITHUB_TOKEN` cannot trigger.
- Pinned all GitHub Actions to commit SHAs across every workflow for supply-chain safety.
- Added missing `timeout-minutes` and `concurrency` blocks to all workflows.
- Standalone publish-skill workflow now accepts `workflow_dispatch` with an explicit `tag` input.

## [0.1.11] - 2026-04-02
### Fixed

- Passed the temp release-notes path directly to GoReleaser so GitHub Actions preserves the `--release-notes` argument during publishing.


## [0.1.10] - 2026-04-02
### Fixed

- Wrote generated GitHub release notes to the runner temp directory so GoReleaser can publish without modifying the checked-in `RELEASE_NOTES.md`.


## [0.1.9] - 2026-04-02
### Changed

- Switched releases to the shared PR-based `scripts/release.sh` flow, with `CHANGELOG.md` supplying the GitHub release notes and CI creating the version tag only after the merged release commit verifies.

### Fixed

- Removed deprecated release shims so there is exactly one supported release entrypoint.


## [0.1.8] - 2026-04-01

### Fixed

- Keyed manual release rerun concurrency by tag so rerunning one release tag cannot cancel a different release recovery run.

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
