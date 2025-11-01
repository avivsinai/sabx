# SABnzbd CLI Implementation Plan

## Phase 1: Critical Gaps (Immediate) 🔴

These are essential features discovered during real-world usage.

### 1.1 Warnings Command
**Priority**: CRITICAL
**Effort**: Low
**User Story**: As a user, I need to see SABnzbd warnings to troubleshoot issues (e.g., "must set maximum bandwidth")

**Status**: ✅ Completed 2025-11-01 (`sabx warnings list`, `sabx warnings clear`)

```bash
sabx warnings list [--json]
sabx warnings clear
```

**API Mapping**:
- `mode=warnings` → List all warnings
- Clear is typically done by interacting with specific warnings

**Files to modify**:
- `internal/sabapi/client.go` - Add `Warnings()` method
- `cmd/sabx/root/warnings.go` - New command file

---

### 1.2 Logs Command
**Priority**: HIGH
**Effort**: Low
**User Story**: As a user, I need to view SABnzbd logs for debugging

**Status**: ✅ Completed 2025-11-01 (`sabx logs show`, `sabx logs tail`)

```bash
sabx logs show [--lines N]
sabx logs tail [--follow]
```

**API Mapping**:
- `mode=showlog` → Get log entries

**Files to modify**:
- `internal/sabapi/client.go` - Add `ShowLog()` method
- `cmd/sabx/root/logs.go` - New command file

---

### 1.3 Scripts Management
**Priority**: HIGH
**Effort**: Low
**User Story**: As a user, I need to list available scripts when setting queue item scripts

**Status**: ✅ Completed 2025-11-01 (`sabx scripts list`)

```bash
sabx scripts list [--json]
```

**API Mapping**:
- `mode=get_scripts` → List available scripts

**Files to modify**:
- `internal/sabapi/client.go` - Add `GetScripts()` method
- `cmd/sabx/root/scripts.go` - New command file

---

### 1.4 Queue Files Management
**Priority**: HIGH
**Effort**: Medium
**User Story**: As a user, I need to see and manage individual files within a queue item

**Status**: ✅ Completed 2025-11-01 (`sabx queue item files`, `sabx queue item files delete`)

```bash
sabx queue item files <nzo-id> [--json]
sabx queue item files <nzo-id> delete <file-id>
```

**API Mapping**:
- `mode=get_files&value=<nzo-id>` → List files in queue item
- `mode=queue&name=delete_nzf&value=<nzo-id>&value2=<file-id>` → Delete file from queue item

**Files to modify**:
- `internal/sabapi/client.go` - Add `GetFiles()`, `QueueDeleteFile()` methods
- `cmd/sabx/root/queue.go` - Add files subcommand

---

## Phase 2: Enhanced Status & Monitoring 🟡

### 2.1 Full Status
**Priority**: MEDIUM
**Effort**: Low
**User Story**: As a user, I want more detailed status information

**Status**: ✅ Completed 2025-11-01 (`sabx status --full`, `--performance`, `--skip-dashboard`)

```bash
sabx status --full [--json]
```

**API Mapping**:
- `mode=fullstatus` → Get comprehensive status

**Files to modify**:
- `internal/sabapi/client.go` - Add `FullStatus()` method
- `cmd/sabx/root/status.go` - Add `--full` flag

---

### 2.2 Server Management
**Priority**: MEDIUM
**Effort**: Medium
**User Story**: As a user, I need to manage news server connections

**Status**: ✅ Completed 2025-11-01 (`sabx server list|stats|test|disconnect|unblock`)

```bash
sabx server list [--json]
sabx server stats [--json]
sabx server test <name>
sabx server disconnect [--duration MINUTES]
sabx server unblock <name>
```

**API Mapping**:
- `mode=server_stats` → Get server statistics
- `mode=config&name=test_server&kwargs` → Test server
- `mode=disconnect` → Disconnect temporarily
- `mode=status&name=unblock_server&value=<server>` → Unblock server

**Files to modify**:
- `internal/sabapi/client.go` - Add server methods
- `cmd/sabx/root/server.go` - Enhance existing command

---

### 2.3 Post-Processing Control
**Priority**: MEDIUM
**Effort**: Low
**User Story**: As a user, I need to control post-processing operations

**Status**: ✅ Completed 2025-11-01 (`sabx postprocess pause|resume|cancel`)

```bash
sabx postprocess pause
sabx postprocess resume
sabx postprocess cancel <nzo-id>
```

**API Mapping**:
- `mode=pause_pp` → Pause post-processing
- `mode=resume_pp` → Resume post-processing
- `mode=cancel_pp&value=<nzo-id>` → Cancel post-processing for item

**Files to modify**:
- `internal/sabapi/client.go` - Add postprocessing methods
- `cmd/sabx/root/postprocess.go` - New command file

---

## Phase 3: Advanced Operations 🟢

### 3.1 History Enhancements
**Priority**: LOW
**Effort**: Low

**Status**: ✅ Completed 2025-11-01 (`sabx history mark-completed`)

```bash
sabx history mark-completed <nzo-id>
```

**API Mapping**:
- `mode=history&name=mark_as_completed&value=<nzo-id>`

**Files to modify**:
- `internal/sabapi/client.go` - Add `HistoryMarkCompleted()` method
- `cmd/sabx/root/history.go` - Add mark-completed command

---

### 3.2 Orphan Management
**Priority**: LOW
**Effort**: Medium

**Status**: ✅ Completed 2025-11-01 (`sabx status orphans ...`)

```bash
sabx status orphans list
sabx status orphans delete <id>
sabx status orphans delete-all
sabx status orphans add <id>
sabx status orphans add-all
```

**API Mapping**:
- `mode=status&name=delete_orphan&value=<id>`
- `mode=status&name=delete_all_orphan`
- `mode=status&name=add_orphan&value=<id>`
- `mode=status&name=add_all_orphan`

**Files to modify**:
- `internal/sabapi/client.go` - Add orphan methods
- `cmd/sabx/root/status.go` - Add orphans subcommand

---

### 3.3 Filesystem Browser
**Priority**: LOW
**Effort**: Medium

**Status**: ✅ Completed 2025-11-01 (`sabx browse`)

```bash
sabx browse <path>
```

**API Mapping**:
- `mode=browse&path=<path>`

**Files to modify**:
- `internal/sabapi/client.go` - Add `Browse()` method
- `cmd/sabx/root/browse.go` - New command file

---

### 3.4 Watched Folder Trigger
**Priority**: LOW
**Effort**: Low

**Status**: ✅ Completed 2025-11-01 (`sabx watched scan`)

```bash
sabx watched scan
```

**API Mapping**:
- `mode=watched_now`

**Files to modify**:
- `internal/sabapi/client.go` - Add `WatchedNow()` method
- `cmd/sabx/root/watched.go` - New command file

---

### 3.5 Queue Enhancements
**Priority**: LOW
**Effort**: Low

**Status**: ✅ Completed 2025-11-01 (`sabx queue complete-action`, `sabx queue item opts`, `sabx queue item files move`)

```bash
sabx queue item change-action <nzo-id> <action>
sabx queue item change-opts <nzo-id> [--pp N] [--repair] [--unpack] [--delete]
sabx queue move-files <nzo-id> <file-ids...> <position>
```

**API Mapping**:
- `mode=queue&name=change_complete_action&value=<nzo-id>&value2=<action>`
- `mode=change_opts&value=<nzo-id>&value2=<pp_value>`
- `mode=move_nzf_bulk` → Bulk move files

**Files to modify**:
- `internal/sabapi/client.go` - Add methods
- `cmd/sabx/root/queue.go` - Add commands

---

## Phase 4: Admin & Testing Features 🔵

### 4.1 Notification Testing
**Priority**: VERY LOW
**Effort**: Low-Medium

```bash
sabx test email
sabx test notification <type>  # pushover, pushbullet, apprise, prowl, etc.
```

**API Mapping**:
- `mode=test_email`
- `mode=test_pushover`, `test_pushbullet`, etc.

---

### 4.2 Configuration Management
**Priority**: LOW
**Effort**: Low-Medium

```bash
sabx config backup
sabx config reset <section> <key>
sabx config purge-logs
sabx config regenerate-certs
```

**API Mapping**:
- `mode=config&name=create_backup`
- `mode=set_config_default`
- `mode=config&name=purge_log_files`
- `mode=config&name=regenerate_certs`

---

### 4.3 Quota Management
**Priority**: VERY LOW
**Effort**: Low

```bash
sabx quota reset
```

**API Mapping**:
- `mode=reset_quota`

---

### 4.4 Advanced Debugging
**Priority**: VERY LOW
**Effort**: Low

```bash
sabx debug gc-stats
sabx debug eval-sort <expression>
```

**API Mapping**:
- `mode=gc_stats`
- `mode=eval_sort&value=<expression>`

---

## CLI Consistency Improvements

Based on GitHub CLI patterns, we should standardize our command structure:

### Verb Consistency
- ✅ `list` (not `show`) for listing items
- ✅ `view` for viewing single item details
- ✅ `create` / `add` for creation
- ✅ `delete` / `remove` for deletion
- ✅ `edit` / `set` for modification

### Current Inconsistencies to Fix
1. `sabx status` → Should it be `sabx status show`?
2. Mix of `list` vs implicit list (e.g., `queue` vs `queue list`)
3. `queue item show` vs `queue show <id>` - pick one pattern

### Recommended Structure
```bash
# Resource-verb pattern (GitHub CLI style)
sabx <resource> <verb> [args] [flags]

# Examples:
sabx queue list
sabx queue pause
sabx queue item view <id>
sabx queue item delete <id>

sabx history list
sabx history delete <id>

sabx warnings list
sabx warnings clear

sabx logs show
sabx logs tail
```

---

## Implementation Order

### Sprint 1 (Immediate - This Week)
1. ✅ Warnings command (DONE - discovered the gap!)
2. ✅ Logs command (shipped 2025-11-01)
3. ✅ Scripts list command (shipped 2025-11-01)
4. ✅ Queue files management (shipped 2025-11-01)

### Sprint 2 (Next Week)
5. ✅ Full status (shipped 2025-11-01)
6. ✅ Server management enhancements (shipped 2025-11-01)
7. ✅ Post-processing control (shipped 2025-11-01)

### Sprint 3 (Following Week)
8. ✅ History enhancements (shipped 2025-11-01)
9. ✅ Orphan management (shipped 2025-11-01)
10. CLI consistency refactor

### Sprint 4 (Future)
11. ✅ Filesystem browser (shipped 2025-11-01)
12. ✅ Watched folder trigger (shipped 2025-11-01)
13. ✅ Advanced queue operations (shipped 2025-11-01)

### Backlog
- Integration/QA automation for newly covered endpoints
- CLI UX polish (naming consistency refactor)

---

## Testing Strategy

For each new command:
1. Add API client method with tests in `internal/sabapi/client_test.go`
2. Add command with `--json` support
3. Test against real SABnzbd instance
4. Document in README
5. Add to API_COVERAGE.md when complete

---

## Success Metrics

- **API Coverage**: Target 90%+ (68/75 endpoints)
- **User Experience**: All common operations should be <3 commands deep
- **AI Agent Friendly**: Clear `--help`, consistent patterns, full JSON support
- **Documentation**: Every command documented with examples
