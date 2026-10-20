#!/usr/bin/env bash
set -euo pipefail

binary_path="${1:?usage: ./scripts/codesign-macos.sh /path/to/binary identifier [target-os]}"
identifier="${2:?usage: ./scripts/codesign-macos.sh /path/to/binary identifier [target-os]}"
target_os="${3:-darwin}"

if [[ "$target_os" != "darwin" ]]; then
  exit 0
fi

if [[ ! -f "$binary_path" ]]; then
  echo "error: binary not found: $binary_path" >&2
  exit 1
fi

if [[ "$(uname -s)" != "Darwin" ]]; then
  exit 0
fi

# Release builds: sign with the Developer ID that release.yml imports, then
# notarize.
#
# Keychain checks two things on each read. The item ACL matches the caller by
# its Designated Requirement, and the item partition list matches the caller by
# a partition ID. For ad-hoc code the partition ID is `cdhash:<hash>`, which
# changes on every build, so each upgrade prompts again even with a pinned DR.
# For Developer ID code it is `teamid:<TEAM_ID>`, which is stable, so one
# "Always Allow" survives upgrades. Keep the default DR here: it anchors to the
# Apple-issued certificate of our team, not only to the identifier.
if [[ -n "${SABX_MACOS_TEAM_ID:-}" ]]; then
  : "${SABX_MACOS_KEYCHAIN:?SABX_MACOS_KEYCHAIN must be set with SABX_MACOS_TEAM_ID}"
  : "${SABX_MACOS_SIGN_IDENTITY:?SABX_MACOS_SIGN_IDENTITY must be set with SABX_MACOS_TEAM_ID}"
  : "${SABX_NOTARY_KEY_PATH:?SABX_NOTARY_KEY_PATH must be set with SABX_MACOS_TEAM_ID}"
  : "${SABX_NOTARY_KEY_ID:?SABX_NOTARY_KEY_ID must be set with SABX_MACOS_TEAM_ID}"
  : "${SABX_NOTARY_ISSUER_ID:?SABX_NOTARY_ISSUER_ID must be set with SABX_MACOS_TEAM_ID}"

  codesign \
    --force \
    --options runtime \
    --timestamp \
    --keychain "$SABX_MACOS_KEYCHAIN" \
    --sign "$SABX_MACOS_SIGN_IDENTITY" \
    --identifier "$identifier" \
    "$binary_path"

  team_id="$(codesign -dv "$binary_path" 2>&1 | sed -n 's/^TeamIdentifier=//p')"
  if [[ "$team_id" != "$SABX_MACOS_TEAM_ID" ]]; then
    echo "error: $binary_path has TeamIdentifier '$team_id', expected '$SABX_MACOS_TEAM_ID'" >&2
    exit 1
  fi

  # notarytool takes a zip; a bare Mach-O cannot be stapled, so Gatekeeper
  # checks the ticket online.
  work_dir="$(mktemp -d "${RUNNER_TEMP:-${TMPDIR:-/tmp}}/sabx-notary.XXXXXX")"
  trap 'rm -rf "$work_dir"' EXIT
  ditto -c -k --keepParent "$binary_path" "$work_dir/sabx.zip"
  submit_rc=0
  xcrun notarytool submit "$work_dir/sabx.zip" \
    --key "$SABX_NOTARY_KEY_PATH" \
    --key-id "$SABX_NOTARY_KEY_ID" \
    --issuer "$SABX_NOTARY_ISSUER_ID" \
    --wait \
    --timeout 20m \
    --output-format json >"$work_dir/result.json" || submit_rc=$?

  status="$(plutil -extract status raw -o - "$work_dir/result.json" 2>/dev/null || echo "unknown (notarytool exit $submit_rc)")"
  if [[ "$status" != "Accepted" ]]; then
    submission_id="$(plutil -extract id raw -o - "$work_dir/result.json" 2>/dev/null || true)"
    echo "error: notarization of $binary_path ended with status '$status' (submission $submission_id)" >&2
    if [[ -n "$submission_id" ]]; then
      xcrun notarytool log "$submission_id" \
        --key "$SABX_NOTARY_KEY_PATH" \
        --key-id "$SABX_NOTARY_KEY_ID" \
        --issuer "$SABX_NOTARY_ISSUER_ID" >&2 || true
    fi
    exit 1
  fi
  exit 0
fi

# Local, fork and snapshot builds: sign ad-hoc with an explicit Designated
# Requirement pinned to the identifier.
#
# Without -r, codesign derives a DR of the form `cdhash H"..."` which changes
# on every rebuild and breaks the Keychain ACL match. The pin keeps the ACL
# match stable, but ad-hoc code still gets a new `cdhash:` partition ID per
# build (see above), so these builds prompt once after each rebuild.
#
# Security trade-off: any other ad-hoc binary that claims this identifier
# satisfies the DR. Acceptable because the Keychain item only stores a user-
# scoped SABnzbd API key, trust is granted per-profile by the user explicitly
# via `sabx login`, and the identifier is namespaced under our GitHub org.
codesign \
  --force \
  --sign - \
  --identifier "$identifier" \
  -r="designated => identifier \"$identifier\"" \
  "$binary_path"
