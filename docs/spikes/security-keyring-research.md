# Spike: Strengthening sabx Credential Storage

## Context
- Date: 2025-11-02
- Participants: sabx maintainers (async)
- Related repos: sabx, jenkins-cli (`/Users/avivsinai/workspace/jenkins-cli`)
- Goal: identify security gaps in sabx keychain handling compared to jenkins-cli and outline potential improvements.

> **Update (Implemented)**: The recommendations below were implemented in November 2025. The “Pre-spike” section is preserved for historical context.

## Outcome (Implemented)
- Secrets store refactored to `internal/auth/store.go`, backed by `github.com/99designs/keyring` with hashed namespace keys (`profile/<name>/<sha256>`) and labeled entries.
- `sabx login` introduces `--allow-insecure-store` (encrypted file fallback) and `--store-in-config` (plaintext opt-in). Fallback choice is recorded on the profile to reopen matching backends.
- Config persistence now writes to `$SABX_CONFIG_DIR/config.yml` (defaults to `~/.config/sabx/config.yml`) using atomic temp-file swaps and `0o700` directory perms.
- Added unit coverage for backend parsing, env toggles, and key derivation (`internal/auth/store_test.go`).

## Pre-spike sabx Behavior (Archived)
- Secrets API: `internal/auth/keyring.go` wrapped `github.com/zalando/go-keyring` with minimal helpers keyed by `profile + ":" + sha1(baseURL)`.
- CLI UX: `sabx login` wrote connection metadata to YAML config and stored API keys in the OS keychain; `--insecure-store` dumped the key into the config file on disk.
- Config storage: `internal/config/config.go` wrote `~/.config/sabx/config.yaml` using `0o600` file perms but left the directory `0o755` and performed in-place writes without temp-file safety.
- Failure handling: keyring errors bubbled up; no fallback path existed except opting into plaintext storage. Missing keyring support resulted in login failure unless the user anticipated the issue.
- Observability: no structured warning when falling back to YAML; no metrics or logs around keychain failures. No automated tests covered the keyring helper.

## Jenkins CLI Highlights
- Pluggable store: `internal/secret/store.go` builds on `github.com/99designs/keyring`, supporting native keychains and an encrypted file backend gated behind `--allow-insecure-store` or `JK_ALLOW_INSECURE_STORE=1`.
- Backend selection: respects `KEYRING_BACKEND`, filters unsupported backends, and exposes `secret.IsNoKeyringError` to detect missing native support.
- Secrets metadata: uses human-readable keys (`context/<name>/token`) and applies item labels to improve UX in keychain GUIs.
- Config durability: writes configs atomically (`os.CreateTemp` + rename) and restricts the directory to `0o700`.
- UX safeguards: persists whether insecure storage was chosen so logouts re-open the correct backend; warns users when an insecure fallback is active.
- Coverage: table-driven tests validate env parsing, backend selection, passphrase handling, and error helpers.

## Gaps & Opportunities for sabx
1. **Keyring abstraction**  
   - Missing backend selection/override and ability to tolerate absent OS keychains.  
   - Hash-based keys obscure hostnames but also hide context, making manual cleanup harder. No labels for keychain UIs.
2. **Fallback strategy**  
   - Only plaintext config fallback; no encrypted file alternative.  
   - No `ErrNoKeyring` signalling, so CLI can’t prompt the user with actionable remediation.
3. **Config persistence**  
   - Non-atomic writes risk partial files during crashes; directory perms allow directory listing by other users.
4. **User guidance**  
   - CLI messaging is minimal; no docs describing secure vs insecure options beyond a warning line.  
   - SECURITY.md lacks operational guidance on handling credentials.
5. **Testing/monitoring**  
   - No unit tests around keyring usage, error propagation, or login flows.  
   - No telemetry or debug logging to help diagnose keyring failures in the field.

## Potential Directions
### A. Adopt a richer keyring wrapper
- Evaluate migrating to `github.com/99designs/keyring` for parity with Jenkins CLI (supports multiple backends, configurable fallbacks).
- Introduce a sabx-specific abstraction mirroring Jenkins’ `secret.Store`, including option hooks and environment toggles.
- Decide on key naming: keep hashed hostnames (privacy) but add context prefix and optional label for UX.

### B. Improve fallback story
- Offer encrypted file backend guarded by `--allow-insecure-store` and a confirmation prompt.  
- Persist fallback choice in config and respect it during logout/delete operations.  
- Add error classification (`auth.ErrKeyringUnavailable`) so commands can instruct users to opt into fallback when necessary.

### C. Harden config writes
- Switch to temp-file + rename flow with `0o700` config dir.  
- Consider separating sensitive fields (API keys) from the main config entirely when insecure storage is disabled.

### D. Enhance UX & documentation
- Expand `login` output to explain when the keychain was used vs skipped.  
- Update docs (README, SECURITY) with credential storage matrix and recommended practices.  
- Optionally emit analytics/structured logs for keyring failures (respecting privacy).

### E. Add automated coverage
- Unit tests for hash/key generation, keyring error paths, and config write behaviour.  
- Integration smoke test toggling `--allow-insecure-store` and environment overrides.

## Open Questions
- Is privacy worth retaining hashed host components, or should we prefer human-readable names plus labels?  
- Do we require backward compatibility with existing hashed key IDs when switching libraries?  
- Should encrypted file fallback be opt-in only, or enabled automatically when native keychain is unavailable (with prompt)?  
- How do we surface keyring diagnostics without leaking sensitive metadata?

## Next Steps (Proposed)
1. Prototype a `internal/auth/store` package using `99designs/keyring` with unit tests mirroring Jenkins coverage.  
2. Design CLI UX changes for login/logout, including messaging and prompts for fallback decisions.  
3. Draft migration plan for existing stored secrets (hash-based IDs) if the backend structure changes.  
4. Update config writer to atomic flow and adjust permissions.  
5. Schedule doc updates once technical approach is settled.

> Implementation completed in November 2025; retained for reference when considering future security enhancements.
