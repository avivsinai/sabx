# Contributing

We welcome contributions that help `sabx` become the go-to SABnzbd CLI. This guide highlights the fastest way to ship high-quality changes.

## Before You Start
- **Discuss major ideas** by filing an issue or starting a discussion. Reference comparable tooling such as [`gh`](https://github.com/cli/cli) for UX inspiration.
- Ensure you have Go 1.22+, SABnzbd (locally or Docker), and `gh` for GitHub workflows.

## Local Setup
```bash
git clone https://github.com/sabx/sabx.git
cd sabx
make setup # optional helper script when available
```

## Code Style
- Run `go fmt ./...` before committing; the repository follows standard Go formatting and idioms.
- Keep packages focused: CLI commands live under `cmd/sabx/root`, shared helpers under `internal/`.
- Prefer descriptive command verbs and API mirrors, e.g. `QueueSetPriority`, `RSSNow`.

## Testing
- Unit tests: `go test ./...`
- Integration tests: spin up `linuxserver/sabnzbd` via Docker Compose (coming soon) and add cases under `internal/sabapi`.
- Include regression coverage for queue and config semantics when adding new API calls.

## Commits & PRs
- Use imperative, scoped commit subjects (e.g. `Add rss run command`).
- Squash fix-ups before review; keep PRs focused and under ~500 LOC when possible.
- PR checklist:
  - [ ] Description of change & motivation
  - [ ] `go test ./...` output in the summary
  - [ ] Screenshots or CLI transcripts for UX changes
  - [ ] Linked issue / discussion

## Release Engineering
- Packaging (Homebrew, Scoop, winget, Docker) will live under `tools/` and GitHub Actions workflows. Contributions here should include dry-run logs and documentation updates in the README.
- Tag releases `vX.Y.Z` following semver.

## Code of Conduct
All interactions are governed by the [Contributor Covenant](CODE_OF_CONDUCT.md). Report unacceptable behavior to the maintainers listed in SECURITY.md.
