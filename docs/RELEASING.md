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
   - publish archives, checksums, and Docker images,
   - push `Formula/sabx.rb` to [`avivsinai/homebrew-tap`](https://github.com/avivsinai/homebrew-tap) via GoReleaser `brews` (same path as `bitbucket-cli`, `jk`, and `amq`),
   - publish skills from the CI-created tag.
4. The release workflow needs `HOMEBREW_TAP_GITHUB_TOKEN` with write access to `avivsinai/homebrew-tap`. That is the same secret used by the other CLI release workflows.
5. macOS binaries are signed as described in [macOS signing](#macos-signing).

Users install with:

```bash
brew install avivsinai/tap/sabx
```

The tap formula GoReleaser publishes is `Formula/sabx.rb`: class `Sabx`, `desc "Full-fidelity SABnzbd CLI"`, `bin.install "sabx"`, and `test` via `sabx --version`. Archive names stay `sabx_<version>_<os>_<arch>.tar.gz` (`x86_64` / `arm64`). The initial v0.1.11 formula lives in [homebrew-tap#17](https://github.com/avivsinai/homebrew-tap/pull/17); later tagged releases overwrite that file.

To retry an existing release, use the `Release` workflow’s **Run workflow** button and supply an existing tag. It is not a manual tag creation path.

## macOS signing

GoReleaser signs the darwin binaries with a Developer ID and notarizes them (`notarize.macos` in `tools/goreleaser.yaml`, through quill on the Linux runner) when these repository secrets are set:

- `APPLE_TEAM_ID`
- `APPLE_DEVELOPER_ID_P12_BASE64`: base64 of a .p12 with the "Developer ID Application" certificate and its key
- `APPLE_DEVELOPER_ID_P12_PASSWORD`
- `APPLE_NOTARY_ISSUER_ID`
- `APPLE_NOTARY_KEY_ID`
- `APPLE_NOTARY_KEY_P8_BASE64`: base64 of an App Store Connect API key (.p8)

Set all six or none. With none, the release warns and ships the Go linker's ad-hoc signature. With some, the release fails. Encode the two base64 values without line breaks (`base64 -i FILE | tr -d '\n'`).

Notarization waits for Apple's verdict (`wait: true`, at most 20 minutes per binary). The release fails, before it publishes anything, when Apple rejects a binary or the wait times out; the error contains Apple's notarization log. Do not remove `wait: true`: without it, GoReleaser only submits and the release publishes binaries that are not notarized.

At install time, the Homebrew formula reads the installed binary with `codesign -dv`. When it has no `TeamIdentifier`, the formula re-signs it ad-hoc with `io.github.avivsinai.sabx`. A Developer ID binary keeps its signature.

Ad-hoc binaries get a new Keychain partition ID (`cdhash:`) per build, so users must approve Keychain access again after each upgrade. Developer ID binaries share one `teamid:` partition ID, so one "Always Allow" survives upgrades. Signed binaries have the identifier `sabx` (quill uses the binary name). Keep the binary name and the team: a changed identifier or team makes every user approve again.

## Snapshot builds

Run `make snapshot` to produce local release artifacts without publishing. Snapshots embed
metadata in the binary via ldflags and include the short commit in their version string.

## Branch policy

- `main` is always releasable. Feature branches must land via pull requests with green CI.
- Breaking changes require a MAJOR bump and should be called out in `CHANGELOG.md`.
- Patch releases (`vX.Y.Z+1`) are for bug fixes; minor releases (`vX.Y+1.0`) are for backwards compatible features.
