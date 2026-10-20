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
   - publish release artifacts via GoReleaser on a macOS runner,
   - publish archives and checksums,
   - build and push the `ghcr.io/avivsinai/sabx` image (with SBOM and provenance attestations) on a Linux runner from the same linux binaries,
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

The release job runs GoReleaser on `macos-latest`. The darwin binaries are built with `CGO_ENABLED=1`, because the macOS Keychain backend of `github.com/99designs/keyring` exists only in cgo builds. Without it, `sabx login` fails with "Specified keyring backend not available". Linux and Windows binaries stay static (`CGO_ENABLED=0`) cross builds. macOS runners have no Docker, so a separate Linux job builds the container image from the linux binaries of the release job.

The GoReleaser post-build hook `scripts/codesign-macos.sh` signs each darwin binary with the identifier `io.github.avivsinai.sabx`. The release workflow imports a Developer ID into a temporary keychain when these repository secrets are set:

- `APPLE_TEAM_ID`
- `APPLE_DEVELOPER_ID_P12_BASE64`: base64 of a .p12 with the "Developer ID Application" certificate, its key, and the Developer ID intermediate certificate
- `APPLE_DEVELOPER_ID_P12_PASSWORD`
- `APPLE_NOTARY_ISSUER_ID`
- `APPLE_NOTARY_KEY_ID`
- `APPLE_NOTARY_KEY_P8_BASE64`: base64 of an App Store Connect API key (.p8)

Set all six or none. With none, the release warns and the script signs ad-hoc with a Designated Requirement pinned to the identifier. With some, the release fails.

With all six, the script signs with `--options runtime --timestamp`, checks that the `TeamIdentifier` of the signature is `APPLE_TEAM_ID`, and submits the binary to `notarytool --wait --timeout 20m`. The release fails before it publishes anything when the status is not `Accepted`; the job log then contains Apple's notarization log. A bare Mach-O binary cannot be stapled, so Gatekeeper checks the ticket online. The workflow deletes the temporary keychain and the key files after GoReleaser, also when it fails.

At install time, the Homebrew formula reads the installed binary with `codesign -dv`. When it has no `TeamIdentifier`, the formula re-signs it ad-hoc with the same identifier and pinned requirement. A Developer ID binary keeps its signature.

Ad-hoc binaries get a new Keychain partition ID (`cdhash:`) per build, so users must approve Keychain access again after each upgrade. Developer ID binaries share one `teamid:` partition ID, so one "Always Allow" survives upgrades. Keep the identifier `io.github.avivsinai.sabx` and the team: a changed identifier or team makes every user approve again.

## Snapshot builds

Run `make snapshot` on macOS to produce local release artifacts without publishing (the darwin builds need cgo and a macOS host). Snapshots embed
metadata in the binary via ldflags and include the short commit in their version string.

## Branch policy

- `main` is always releasable. Feature branches must land via pull requests with green CI.
- Breaking changes require a MAJOR bump and should be called out in `CHANGELOG.md`.
- Patch releases (`vX.Y.Z+1`) are for bug fixes; minor releases (`vX.Y+1.0`) are for backwards compatible features.
