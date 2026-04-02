# sabx Release & Versioning Guide

This project follows **Semantic Versioning (SemVer)** using annotated git tags of the form `vMAJOR.MINOR.PATCH`.

## Release prerequisites

1. Ensure `main` is green in CI and contains the change set you want to ship.
2. Update `CHANGELOG.md` under `Unreleased`.
3. Run the full test matrix locally:
   ```bash
   make tidy
   make fmt
   SABX_E2E_DISABLE=1 make test
   make smoke # requires Docker / Testcontainers
   ```

## Creating a release

1. Use the release helper from `main`:
   ```bash
   ./scripts/release.sh X.Y.Z
   ```
   The helper verifies the worktree is clean and at `origin/main`, creates `release/vX.Y.Z`, moves `CHANGELOG.md` into a dated release entry, bumps skill/plugin metadata, runs verification, commits `chore(release): vX.Y.Z`, pushes the branch, opens a PR, and enables squash auto-merge by default.
2. Let the release PR merge. Do not create or push tags manually.
3. After the release PR merges, GitHub Actions will automatically:
   - validate the merged release commit,
   - create `vX.Y.Z` only after verification succeeds,
   - publish release artifacts via GoReleaser,
   - publish archives, checksums, Docker images, and (optionally) Homebrew/Scoop manifests,
   - publish skills from the CI-created tag.
4. On macOS, the Homebrew cask install hook re-signs the installed `sabx` binary with a stable reverse-DNS identifier so Keychain prompts stay associated across upgrades.

To retry an existing release, use the `Release` workflow’s **Run workflow** button and supply an existing tag. It is not a manual tag creation path.

## Snapshot builds

Run `make snapshot` to produce local release artifacts without publishing. Snapshots embed
metadata in the binary via ldflags and include the short commit in their version string.

## Branch policy

- `main` is always releasable. Feature branches must land via pull requests with green CI.
- Breaking changes require a MAJOR bump and should be called out in `CHANGELOG.md`.
- Patch releases (`vX.Y.Z+1`) are for bug fixes; minor releases (`vX.Y+1.0`) are for backwards compatible features.
