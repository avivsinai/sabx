# sabx Release & Versioning Guide

This project follows **Semantic Versioning (SemVer)** using annotated git tags of the form `vMAJOR.MINOR.PATCH`.

## Release prerequisites

1. Ensure `main` is green in CI and contains the change set you want to ship.
2. If you want a human-written summary in the release archives, update `RELEASE_NOTES.md`.
3. Run the full test matrix locally:
   ```bash
   make tidy
   make fmt
   SABX_E2E_DISABLE=1 make test
   make smoke # requires Docker / Testcontainers
   ```

## Creating a release

1. Use the release helper to bump skill metadata, validate release versions, and create the local release tag:
   ```bash
   ./scripts/release-skills.sh X.Y.Z
   ```
   The helper creates the annotated `vX.Y.Z` tag locally.
2. GitHub Actions will automatically:
   - run lint/unit/e2e smoke tests on Linux,
   - verify release metadata matches the tag before publishing,
   - publish release artifacts via GoReleaser,
   - publish archives, checksums, Docker images, and (optionally) Homebrew/Scoop manifests.
3. Push the release commit and tag:
   ```bash
   git push origin HEAD
   git push origin vX.Y.Z
   ```
4. On macOS, the Homebrew cask install hook re-signs the installed `sabx` binary with a stable reverse-DNS identifier so Keychain prompts stay associated across upgrades.

To retry or cut a release from an existing commit, use the `Release` workflow’s **Run workflow** button and supply the tag.

## Snapshot builds

Run `make snapshot` to produce local release artifacts without publishing. Snapshots embed
metadata in the binary via ldflags and include the short commit in their version string.

## Branch policy

- `main` is always releasable. Feature branches must land via pull requests with green CI.
- Breaking changes require a MAJOR bump and should be called out in `RELEASE_NOTES.md` when you want them in the archive metadata.
- Patch releases (`vX.Y.Z+1`) are for bug fixes; minor releases (`vX.Y+1.0`) are for backwards compatible features.
