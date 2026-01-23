# SABnzbd CLI Command Reference

Complete command reference for `sabx`. Run `sabx <command> --help` for details.

## Queue Commands

### List and inspect
```bash
sabx queue list                   # All queue items
sabx queue list --active          # Currently downloading
sabx queue list --search "term"   # Filter by name
sabx queue list --limit 10        # Limit results
sabx queue list --json            # JSON output
```

### Add downloads
```bash
sabx queue add url <url>                # Add from URL
sabx queue add url <url> --cat movies   # With category
sabx queue add url <url> --priority 2   # With priority (-1=low,0=normal,1=high,2=force)
sabx queue add file <path>              # Add local NZB file
sabx queue add local <server-path>      # Add file already on SAB server
```

Add options: `--cat`, `--priority`, `--script`, `--password`, `--name`

### Queue control
```bash
sabx queue pause                  # Pause all
sabx queue resume                 # Resume all
sabx queue purge --all            # Remove all items
sabx queue purge --search "term"  # Remove matching items
sabx queue sort <name|age|size|eta>  # Sort queue
sabx queue sort name --desc       # Sort descending
```

### Item operations
```bash
sabx queue item show <nzo-id>           # Show item details
sabx queue item pause <nzo-id>          # Pause specific item
sabx queue item resume <nzo-id>         # Resume specific item
sabx queue item delete <nzo-id>         # Delete from queue
sabx queue item delete <nzo-id> --with-data  # Also delete files
sabx queue item priority <nzo-id> <-1..2>    # Set priority
sabx queue item move <nzo-id> top       # Move to top
sabx queue item move <nzo-id> bottom    # Move to bottom
sabx queue item move <nzo-id> up        # Move up one position
sabx queue item move <nzo-id> down      # Move down one position
sabx queue item move <nzo-id> to 0      # Move to specific position
sabx queue item set <nzo-id> --cat movies   # Change category
sabx queue item set <nzo-id> --script pp.sh # Change script
sabx queue item set <nzo-id> --name "New Name"  # Rename
sabx queue item files <nzo-id>          # List files in NZB
```

### Item file operations
```bash
sabx queue item files <nzo-id>                              # List NZF files
sabx queue item files delete <nzo-id> <nzf-id>              # Delete specific file
sabx queue item files move <nzo-id> --action top --id <nzf-id>     # Move file to top
sabx queue item files move <nzo-id> --action up --id <nzf-id> --size 3  # Move up 3 positions
```

### Queue completion action
```bash
sabx queue complete-action shutdown      # Shutdown PC when done
sabx queue complete-action hibernate     # Hibernate when done
sabx queue complete-action standby       # Standby when done
sabx queue complete-action shutdown-program  # Stop SABnzbd when done
sabx queue complete-action none          # Clear completion action
```

## History Commands

```bash
sabx history list                 # Recent history
sabx history list --limit 50      # More items
sabx history list --failed        # Failed only
sabx history list --completed     # Completed only
sabx history retry <nzo-id>       # Retry failed download
sabx history retry --all          # Retry all failed
sabx history delete <nzo-id>      # Delete from history
sabx history delete --all         # Delete entire history
sabx history delete --failed      # Delete only failed items
sabx history mark-completed <nzo-id>  # Mark as completed
```

## Status & Diagnostics

```bash
sabx status                       # Basic status
sabx status --full                # Full status with details
sabx status --performance         # Include performance metrics
sabx status --full --performance  # Everything
sabx status orphans list          # List orphaned files
sabx status orphans delete <path> # Delete orphan
sabx status orphans delete-all    # Delete all orphans
sabx status orphans add <path>    # Add orphan to queue
sabx status orphans add-all       # Add all orphans to queue

sabx warnings list                # Runtime warnings
sabx warnings clear               # Clear warnings

sabx logs list                    # Recent log entries
sabx logs list --lines 100        # More lines
sabx logs tail                    # Last entries
sabx logs tail --follow           # Stream live

sabx doctor                       # Connectivity & health checks
```

## Speed Control

```bash
sabx speed status                 # Current speed and limits
sabx speed status --json          # For scripting
sabx speed limit --rate <value>   # Set speed limit
sabx speed limit --rate 50%       # Percentage of max
sabx speed limit --rate 800K      # Kilobytes per second
sabx speed limit --rate 4M        # Megabytes per second
sabx speed limit --rate 4MB/s     # Alternative format
sabx speed limit --rate 10Mbps    # Megabits per second
sabx speed limit --none           # Remove limit
```

## Post-Processing

```bash
sabx postprocess pause            # Pause all post-processing
sabx postprocess resume           # Resume post-processing
sabx postprocess cancel <nzo-id>  # Cancel specific job's PP
```

## Server Management

```bash
sabx server list                  # List news servers
sabx server stats                 # Per-server statistics (all servers)
sabx server test <server-name>    # Test server connectivity
sabx server disconnect            # Disconnect from all servers
sabx server unblock <server-name> # Unblock server
sabx server restart               # Restart SABnzbd
sabx server shutdown              # Shutdown SABnzbd
sabx server repair                # Repair queue
```

## RSS Feeds

```bash
sabx rss list                     # List all feeds
sabx rss add <name> --url <url>   # Add feed
sabx rss add <name> --url <url> --cat tv --priority 1
sabx rss set <name> --set uri=<new-url>
sabx rss set <name> --set enabled=1
sabx rss delete <name>            # Delete feed
sabx rss run <name>               # Manually run specific feed
sabx rss run                      # Run all feeds
```

## Categories

```bash
sabx categories list              # List categories
sabx categories add <name>        # Add category
sabx categories add <name> --dir /path --script pp.sh
sabx categories set <name> --set dir=/new/path
sabx categories delete <name>     # Delete category
```

## Scheduler

```bash
sabx schedule list                # List scheduled tasks
sabx schedule add <name> --set command=<cmd> --set day=<days> --set hour=<HH> --set min=<MM>
# Commands: pause, resume, speedlimit, scan_folder, etc.
# Days: daily, weekdays, mon, tue, wed, thu, fri, sat, sun, mon-fri, etc.
sabx schedule set <name> --set hour=02
sabx schedule delete <name>       # Delete task
```

## Configuration

```bash
sabx config get <section>         # Get config section
sabx config get misc              # General settings
sabx config get servers           # Server configs
sabx config get <section> --key <keyword>  # Get specific key
sabx config set <section> --set key=value
sabx config set <section> --name <item> --set key=value  # Named item
sabx config delete <section> --key <keyword>   # Delete by keyword
sabx config delete <section> --name <itemname> # Delete named item

# Special config operations
sabx config rotate-api-key        # Generate new API key
sabx config rotate-nzb-key        # Generate new NZB key
sabx config regenerate-certs      # Regenerate SSL certs
sabx config backup                # Create config backup
sabx config purge-logs            # Purge log files
sabx config reset-default <key>   # Reset to default
sabx config set-pause <minutes>   # Set auto-resume timer
```

## Filesystem & Watched Folders

```bash
sabx browse /                     # Browse SAB server filesystem
sabx browse /downloads --files    # Include files
sabx browse /downloads --json     # JSON output

sabx watched scan                 # Trigger watched folder scan
sabx watched scan --json          # JSON output
```

## Notifications

```bash
sabx notifications test email     # Test email notification
sabx notifications test pushover  # Test Pushover
sabx notifications test apprise   # Test Apprise
sabx notifications test prowl     # Test Prowl
sabx notifications test pushbullet
sabx notifications test desktop   # Desktop notification
sabx notifications test windows   # Windows notification
sabx notifications test script    # Notification script
```

## Quota

```bash
sabx quota reset                  # Reset download quota counters
```

## Debug & Development

```bash
sabx debug gc-stats               # Go GC statistics
sabx debug eval-sort "<pattern>" --job "Show.Name"  # Test sort pattern

sabx dump config                  # Export sanitized config
sabx dump state                   # Export current state

sabx translate <key>              # Resolve UI translation key
```

## Scripts

```bash
sabx scripts list                 # Available post-processing scripts
```

## Extensions

```bash
sabx extension list               # List installed extensions
sabx extension install <repo>     # Install from GitHub
sabx extension install avivsinai/sabx-tv-tools
sabx extension remove <name>      # Remove extension
```

## Live Dashboard

```bash
sabx top                          # Interactive Bubble Tea dashboard
```

## Global Options

All commands support:
- `--json` — JSON output for scripting
- `--quiet` — Suppress non-error output
- `--profile <name>` — Use specific profile
- `--base-url <url>` — Override SABnzbd URL
- `--api-key <key>` — Override API key
- `--help` — Command help

## Environment Variables

- `SABX_BASE_URL` — SABnzbd server URL
- `SABX_API_KEY` — API key (prefer keyring)
- `SABX_CONFIG_DIR` — Config directory override
- `SABX_ALLOW_INSECURE_STORE` — Allow file-based credential storage
