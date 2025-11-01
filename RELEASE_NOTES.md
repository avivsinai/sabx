# Release Notes – SABnzbd API Parity Lock (November 1, 2025)

## Overview
This release completes the SABnzbd API parity initiative: every published endpoint (75 total) now has a first-class `sabx` command, verified by tooling and backed by smoke/integration coverage. The CLI is ready for a tagged parity release.

## Highlights
- ✅ **API coverage audit** – `go run ./tools/coverage` enumerates all `(mode, name)` combinations exercised by `sabx`, confirming 75/75 endpoint coverage and surfacing the responsible functions.
- ✅ **Smoke harness** – `scripts/run-smoke.sh` (backed by `tools/smoke`) runs end-to-end checks against a live SABnzbd instance, capturing sanitized JSON fixtures for tricky endpoints (filesystem browse, watched_now, notification testers, eval_sort, orphan lifecycle).
- ✅ **Integration tests** – `internal/sabapi` tests now assert response parsing for queue file moves, API/NZB key rotation, and notification testers—not just request shapes.
- ✅ **CLI polish** – command verbs and help text are consistent: `logs list` (alias `show`), `speed status`, and every parity-era command advertises `--json` behaviour. Help strings mention error surfaces.
- ✅ **Documentation** – README quickstart expanded with new examples (browse, watched, quota, notifications, eval-sort). Added an “API Parity Checklist” table and smoke-test guidance. `API_COVERAGE.md` references the new coverage tool and reflects queue purge/move semantics.

## Compatibility Notes
- `sabx logs list` replaces the old `logs show` name; the legacy verb remains available as an alias.
- `sabx speed status` supersedes `speed show` (alias preserved) and documents JSON output expectations.
- No other breaking changes; command normalization is additive.

## Testing
- `go test ./...`
- `go run ./tools/coverage` (75 operations detected)
- Smoke harness ready via `scripts/run-smoke.sh --base-url <url> --api-key <key> --output testdata/smoke/latest`

## Next Steps
- Wire the smoke harness into CI once a SABnzbd Docker service is available.
- Review captured fixtures and promote stable subsets into regression tests.
- Tag the release and share parity highlights with the community.
