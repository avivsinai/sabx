# SABnzbd API Coverage Analysis

Generated: 2025-11-01

## Executive Summary

**Total SABnzbd API Endpoints**: 75
**Implemented in sabx**: 75
**Coverage**: 100%

The new `tools/coverage` utility statically analyses `internal/sabapi/client.go` (plus a handful of CLI-only actions) to confirm endpoint coverage. Run it with:

```bash
go run ./tools/coverage
```

The tool enumerates 75 distinct `(mode, name)` combinations and surfaces the sabx functions that exercise each call, ensuring the table below stays in sync with the implementation.

## Critical Gaps Identified

### 🔴 HIGH PRIORITY (User-facing, common operations)

Recently shipped (2025-11-01):
- ✅ **warnings** – list + clear commands implemented
- ✅ **showlog** – logs list/tail commands implemented
- ✅ **get_files** – queue item files listing implemented
- ✅ **get_scripts** – scripts list command implemented
- ✅ **fullstatus** – `sabx status --full` with optional performance metrics
- ✅ **post-processing control** – pause/resume/cancel via `sabx postprocess`
- ✅ **server management** – stats, test, disconnect, unblock coverage

All remaining endpoints have been implemented; sabx now offers full API parity.

### 🟡 MEDIUM PRIORITY (Advanced operations)

- _None outstanding_

### 🟢 LOW PRIORITY (Testing/Admin/Advanced)

- _None outstanding_

---

## Detailed Comparison

### ✅ IMPLEMENTED

#### Main API Operations
- `queue` - Get queue listing
- `history` - Get history listing
- `status` - Get basic status
- `version` - Get SABnzbd version
- `addurl` - Add NZB by URL
- `addfile` - Add NZB file upload
- `pause` - Pause downloading
- `resume` - Resume downloading
- `shutdown` - Shutdown SABnzbd
- `restart` - Restart SABnzbd
- `config` - Get/set configuration
- `get_config` - Get configuration sections
- `set_config` - Set configuration values
- `del_config` - Delete configuration values
- `get_cats` - List categories
- `rss_now` - Trigger RSS feed scan
- `retry` - Retry individual history item
- `retry_all` - Retry all failed history items
- `switch` - Switch queue item position
- `change_cat` - Change queue item category
- `change_script` - Change queue item script
- `warnings` - List and clear warning messages
- `showlog` - Fetch sanitized SABnzbd logs
- `get_scripts` - Enumerate available post-processing scripts
- `get_files` - Inspect files belonging to a queue item
- `fullstatus` - Comprehensive status payload
- `pause_pp` / `resume_pp` / `cancel_pp` - Post-processing controls
- `server_stats` - Bandwidth statistics
- `disconnect` - Force server disconnect
- `test_server` - Built-in connectivity test
- `translate` - Translate UI strings
- `browse` - Filesystem browser
- `watched_now` - Trigger watched-folder scan
- `move_nzf_bulk` - Reorder NZF files within items
- `change_opts` - Update post-processing options
- `change_complete_action` - Configure queue completion action
- `history.mark_as_completed` - Mark history entries as completed
- `status.unblock_server` - Unblock temporarily blocked servers
- `status.delete_orphan` / `delete_all_orphan` - Manage orphaned jobs
- `status.add_orphan` / `add_all_orphan` - Re-add orphaned jobs
- `eval_sort` - Evaluate sorting expressions
- `reset_quota` - Reset download quota
- `gc_stats` - Gather garbage collector stats
- `addlocalfile` - Register server-side NZBs
- `test_email`, `test_windows`, `test_notif`, `test_osd`, `test_pushover`, `test_pushbullet`, `test_apprise`, `test_prowl`, `test_nscript`
- `set_pause`, `set_apikey`, `set_nzbkey`, `regenerate_certs`, `create_backup`, `purge_log_files`, `set_config_default`
- `restart_repair` - Queue repair with restart

#### Queue Operations (mode=queue)
- `delete` - Delete queue items
- `rename` - Rename queue item (includes password setting)
- `pause` - Pause queue item
- `resume` - Resume queue item
- `priority` - Set queue item priority
- `sort` - Sort queue
- `delete_nzf` - Delete individual files within a queue item
- `move` - Reorder queue items relative to current position
- `purge` - Bulk delete queue items with optional filters

#### History Operations (mode=history)
- `delete` - Delete history items

#### Config Operations (mode=config)
- `speedlimit` - Set speed limit

---

## Recommendations

### Immediate Action Required

- None. All published API endpoints now map to sabx commands.

### Next Steps

1. **Deep QA** – add integration coverage that exercises the new endpoints against live SABnzbd instances (esp. notifications and browse).
2. **Docs & UX polish** – surface the expanded command set in README/help examples and consider command aliases where helpful.
3. **Automation hooks** – explore scripted smoke tests (e.g., `sabx debug gc-stats`, notification dry-runs) for release pipelines.

## CLI Ergonomics Notes

Based on GitHub CLI patterns:

- Use `list` instead of `show` for consistency
- Use subcommands for related operations (`sabx warnings list/clear`)
- Support both `--json` and human-readable output
- Use consistent naming: `delete` not `remove`, `list` not `show`
- Group related operations under parent commands
- Provide `--help` at every level
- Support bulk operations where sensible

### Current Inconsistencies to Fix

- _None. Command verbs and help text were normalized in the parity polish pass (2025-11-01)._
