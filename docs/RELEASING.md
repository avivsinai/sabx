# sabx Release & Versioning Guide

This project follows **Semantic Versioning (SemVer)** using annotated git tags of the form `vMAJOR.MINOR.PATCH`.

## Release prerequisites

1. Ensure `main` is green in CI and contains the change set you want to ship.
2. Update `RELEASE_NOTES.md` with a concise summary for the upcoming tag.
3. Run the full test matrix locally:
   ```bash
   make tidy
   make fmt
   SABX_E2E_DISABLE=1 make test
   make smoke # requires Docker / Testcontainers
   ```

## Creating a release

1. Create and push a tag:
   ```bash
   git tag -a vX.Y.Z -m "sabx vX.Y.Z"
   git push origin vX.Y.Z
   ```
   The `v` prefix is required for the automation.
2. GitHub Actions will automatically:
   - run lint/unit/e2e smoke tests on Linux,
   - build signed artifacts via GoReleaser,
   - publish archives, checksums, Docker images, and (optionally) Homebrew/Scoop manifests.

To retry or cut a release from an existing commit, use the `Release` workflow’s **Run workflow** button and supply the tag.

## Snapshot builds

Run `make snapshot` to produce local release artifacts without publishing. Snapshots embed
metadata in the binary via ldflags and include the short commit in their version string.

## Branch policy

- `main` is always releasable. Feature branches must land via pull requests with green CI.
- Breaking changes require a MAJOR bump and must be called out in `RELEASE_NOTES.md`.
- Patch releases (`vX.Y.Z+1`) are for bug fixes; minor releases (`vX.Y+1.0`) are for backwards compatible features.
