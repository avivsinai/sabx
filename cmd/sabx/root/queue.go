package root

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/sabx/sabx/internal/sabapi"
)

func queueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "queue",
		Short: jsonShort("Inspect and control the active queue"),
		Long:  appendJSONLong("Manage SABnzbd queue items: list, add, reorder, or modify jobs."),
	}

	cmd.AddCommand(queueListCmd())
	cmd.AddCommand(queueAddCmd())
	cmd.AddCommand(queuePauseCmd())
	cmd.AddCommand(queueResumeCmd())
	cmd.AddCommand(queuePurgeCmd())
	cmd.AddCommand(queueCompleteActionCmd())
	cmd.AddCommand(queueItemCmd())
	cmd.AddCommand(queueSortCmd())

	return cmd
}

func queueListCmd() *cobra.Command {
	var search string
	var limit int
	var onlyActive bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: jsonShort("List queue entries"),
		Long:  appendJSONLong("Lists queue items, optionally filtering by search term or active download state."),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			queue, err := app.Client.Queue(ctx, 0, limit, search)
			if err != nil {
				return err
			}

			slots := queue.Slots
			if onlyActive {
				filtered := make([]sabapi.QueueSlot, 0, len(slots))
				for _, slot := range slots {
					if strings.EqualFold(slot.Status, "Downloading") || strings.EqualFold(slot.Status, "Fetching") {
						filtered = append(filtered, slot)
					}
				}
				slots = filtered
			}

			if app.Printer.JSON {
				payload := map[string]any{
					"slots":      slots,
					"paused":     queue.Paused,
					"speed_kbps": queue.Speed,
					"limit_kbps": queue.SpeedLimit,
				}
				return app.Printer.Print(payload)
			}

			headers := []string{"ID", "Name", "Status", "Done/Left (MB)", "ETA", "Priority"}
			rows := make([][]string, 0, len(slots))
			for _, slot := range slots {
				rows = append(rows, []string{
					slot.NZOID,
					slot.Filename,
					slot.Status,
					fmt.Sprintf("%s/%s", slot.MB, slot.MBLeft),
					slot.Eta,
					priorityLabel(slot.Priority),
				})
			}
			if err := app.Printer.Table(headers, rows); err != nil {
				return err
			}
			summary := fmt.Sprintf("%d items | Speed %s KB/s (limit %s) | Paused=%v", len(slots), queue.Speed, queue.SpeedLimit, queue.Paused)
			return app.Printer.Print(summary)
		},
	}

	cmd.Flags().StringVar(&search, "search", "", "Filter queue by search string")
	cmd.Flags().IntVar(&limit, "limit", 0, "Limit number of results (0 = all)")
	cmd.Flags().BoolVar(&onlyActive, "active", false, "Show only actively downloading items")

	return cmd
}

func queueAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: jsonShort("Add NZBs to the queue"),
		Long:  appendJSONLong("Add NZBs via URL, file upload, or server-side path."),
	}

	cmd.AddCommand(queueAddURLCmd())
	cmd.AddCommand(queueAddFileCmd())
	cmd.AddCommand(queueAddLocalCmd())

	return cmd
}

func queueAddURLCmd() *cobra.Command {
	var category string
	var priorityStr string
	var script string
	var password string
	var name string

	cmd := &cobra.Command{
		Use:   "url <nzb-url>",
		Short: jsonShort("Add an NZB by URL"),
		Long:  appendJSONLong("Fetch an NZB from a remote URL and enqueue it. Errors surface when SABnzbd rejects the NZB."),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			nzbURL := args[0]
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			opts, err := buildAddOptions(priorityStr, category, script, password, name)
			if err != nil {
				return err
			}

			resp, err := app.Client.AddURL(ctx, nzbURL, opts)
			if err != nil {
				return err
			}
			if !resp.Success() {
				return fmt.Errorf("sabnzbd refused nzb: %s", firstNonEmpty(resp.Error, resp.Message, "unknown error"))
			}

			if app.Printer.JSON {
				return app.Printer.Print(resp)
			}
			return app.Printer.Print(fmt.Sprintf("Queued %s", strings.Join(resp.NZOIDs, ",")))
		},
	}

	bindAddFlags(cmd.Flags(), &category, &priorityStr, &script, &password, &name)
	return cmd
}

func queueAddFileCmd() *cobra.Command {
	var category string
	var priorityStr string
	var script string
	var password string
	var name string

	cmd := &cobra.Command{
		Use:   "file <path>",
		Short: jsonShort("Upload an NZB file"),
		Long:  appendJSONLong("Upload a local NZB file to SABnzbd. Errors surface if the file cannot be read or SABnzbd rejects it."),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			path := args[0]
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			opts, err := buildAddOptions(priorityStr, category, script, password, name)
			if err != nil {
				return err
			}

			resp, err := app.Client.AddFile(ctx, path, opts)
			if err != nil {
				return err
			}
			if !resp.Success() {
				return fmt.Errorf("sabnzbd refused nzb: %s", firstNonEmpty(resp.Error, resp.Message, "unknown error"))
			}

			if app.Printer.JSON {
				return app.Printer.Print(resp)
			}
			return app.Printer.Print(fmt.Sprintf("Uploaded %s", strings.Join(resp.NZOIDs, ",")))
		},
	}

	bindAddFlags(cmd.Flags(), &category, &priorityStr, &script, &password, &name)
	return cmd
}

func queueAddLocalCmd() *cobra.Command {
	var category string
	var priorityStr string
	var script string
	var password string
	var name string

	cmd := &cobra.Command{
		Use:   "local <path>",
		Short: jsonShort("Register an NZB that already exists on the SABnzbd host"),
		Long:  appendJSONLong("Register an NZB file already present on the SABnzbd server. Useful for shared storage."),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			remotePath := args[0]
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			opts, err := buildAddOptions(priorityStr, category, script, password, name)
			if err != nil {
				return err
			}

			resp, err := app.Client.AddLocalFile(ctx, remotePath, opts)
			if err != nil {
				return err
			}
			if !resp.Success() {
				return errors.New("sabnzbd refused nzb")
			}

			if app.Printer.JSON {
				return app.Printer.Print(resp)
			}
			return app.Printer.Print(fmt.Sprintf("Queued %s", strings.Join(resp.NZOIDs, ",")))
		},
	}

	bindAddFlags(cmd.Flags(), &category, &priorityStr, &script, &password, &name)
	return cmd
}

func bindAddFlags(flags *pflag.FlagSet, category, priority, script, password, name *string) {
	flags.StringVar(category, "cat", "", "Category to assign")
	flags.StringVar(priority, "priority", "", "Priority (-1 low,0 normal,1 high,2 force)")
	flags.StringVar(script, "script", "", "Post-processing script")
	flags.StringVar(password, "password", "", "Archive password")
	flags.StringVar(name, "name", "", "Override queue title")
}

func buildAddOptions(priorityStr, category, script, password, name string) (sabapi.AddOptions, error) {
	opts := sabapi.AddOptions{Category: category, Script: script, Password: password, Name: name}
	if strings.TrimSpace(priorityStr) != "" {
		p, err := strconv.Atoi(priorityStr)
		if err != nil {
			return opts, fmt.Errorf("invalid priority: %w", err)
		}
		opts.Priority = &p
	}
	return opts, nil
}

func queuePauseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pause",
		Short: jsonShort("Pause the entire queue"),
		Long:  appendJSONLong("Pauses all active downloads via SABnzbd's queue API."),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()
			return app.Client.QueuePause(ctx, "")
		},
	}
	return cmd
}

func queueResumeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resume",
		Short: jsonShort("Resume the entire queue"),
		Long:  appendJSONLong("Resumes paused downloads via SABnzbd's queue API."),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()
			return app.Client.QueueResume(ctx, "")
		},
	}
	return cmd
}

func queuePurgeCmd() *cobra.Command {
	var purgeAll bool
	var search string
	var deleteData bool
	cmd := &cobra.Command{
		Use:   "purge",
		Short: jsonShort("Purge queue entries"),
		Long:  appendJSONLong("Deletes queue items by filter or entirely. Use --delete-data to remove downloaded files."),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !purgeAll && strings.TrimSpace(search) == "" {
				return errors.New("provide --all to purge everything or --search to filter items")
			}
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()
			params := url.Values{}
			// Note: when purgeAll is true, no additional params required;
			// SAB interprets empty purge as full purge
			if search != "" {
				params.Set("search", search)
			}
			if deleteData {
				params.Set("del_files", "1")
			}
			return app.Client.QueueAction(ctx, "purge", params)
		},
	}
	cmd.Flags().BoolVar(&purgeAll, "all", false, "Purge every queue entry")
	cmd.Flags().StringVar(&search, "search", "", "Purge items whose name matches this substring")
	cmd.Flags().BoolVar(&deleteData, "with-data", false, "Also delete downloaded data")
	return cmd
}

func queueCompleteActionCmd() *cobra.Command {
	actions := map[string]string{
		"none":             "",
		"":                 "",
		"shutdown":         "shutdown_pc",
		"shutdown_pc":      "shutdown_pc",
		"shutdown-program": "shutdown_program",
		"shutdown_program": "shutdown_program",
		"hibernate":        "hibernate_pc",
		"hibernate_pc":     "hibernate_pc",
		"standby":          "standby_pc",
		"standby_pc":       "standby_pc",
	}

	cmd := &cobra.Command{
		Use:   "complete-action <action>",
		Short: jsonShort("Set the queue completion action"),
		Long:  appendJSONLong("Configure what SABnzbd should do when the queue completes (e.g. shutdown or standby)."),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := strings.ToLower(strings.TrimSpace(args[0]))
			action, ok := actions[input]
			if !ok {
				return fmt.Errorf("unknown action %q (use shutdown|shutdown-program|hibernate|standby|none)", input)
			}

			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			if err := app.Client.QueueSetCompleteAction(ctx, action); err != nil {
				return err
			}

			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"action": action})
			}
			if action == "" {
				return app.Printer.Print("Cleared completion action")
			}
			return app.Printer.Print(fmt.Sprintf("Set completion action to %s", action))
		},
	}
	return cmd
}

func queueItemCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "item",
		Short: jsonShort("Operate on individual queue items"),
		Long:  appendJSONLong("Inspect or modify specific SABnzbd queue entries."),
	}

	cmd.AddCommand(queueItemShowCmd())
	cmd.AddCommand(queueItemPauseCmd())
	cmd.AddCommand(queueItemResumeCmd())
	cmd.AddCommand(queueItemDeleteCmd())
	cmd.AddCommand(queueItemPriorityCmd())
	cmd.AddCommand(queueItemMoveCmd())
	cmd.AddCommand(queueItemSetCmd())
	cmd.AddCommand(queueItemOptsCmd())
	cmd.AddCommand(queueItemFilesCmd())

	return cmd
}

func queueItemShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <nzo-id>",
		Short: jsonShort("Show detailed information for an item"),
		Long:  appendJSONLong("Displays full queue slot metadata, including stage logs."),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			slot, err := findQueueSlot(ctx, app.Client, id)
			if err != nil {
				return err
			}

			if app.Printer.JSON {
				return app.Printer.Print(slot)
			}

			var b strings.Builder
			fmt.Fprintf(&b, "%s\nCategory: %s\nPriority: %s\nStatus: %s\nMB: %s\nMB Left: %s\nETA: %s", slot.Filename, slot.Category, priorityLabel(slot.Priority), slot.Status, slot.MB, slot.MBLeft, slot.Eta)
			if len(slot.StageLog) > 0 {
				b.WriteString("\nStages:")
				for _, entry := range slot.StageLog {
					fmt.Fprintf(&b, "\n- %s: %s", entry.Stage, entry.Log)
				}
			}
			return app.Printer.Print(b.String())
		},
	}
	return cmd
}

func queueItemPauseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pause <nzo-id>",
		Short: jsonShort("Pause an item"),
		Long:  appendJSONLong("Pauses a specific queue item in SABnzbd."),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()
			return app.Client.QueuePause(ctx, args[0])
		},
	}
	return cmd
}

func queueItemResumeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resume <nzo-id>",
		Short: jsonShort("Resume an item"),
		Long:  appendJSONLong("Resumes a paused queue item."),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()
			return app.Client.QueueResume(ctx, args[0])
		},
	}
	return cmd
}

func queueItemDeleteCmd() *cobra.Command {
	var deleteData bool
	cmd := &cobra.Command{
		Use:   "delete <nzo-id>",
		Short: jsonShort("Delete an item"),
		Long:  appendJSONLong("Deletes a queue item. Use --with-data to also remove downloaded files when supported."),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()
			return app.Client.QueueDelete(ctx, []string{args[0]}, deleteData)
		},
	}
	cmd.Flags().BoolVar(&deleteData, "with-data", false, "Also delete already downloaded data")
	return cmd
}

func queueItemPriorityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "priority <nzo-id> <value>",
		Short: jsonShort("Change item priority"),
		Long:  appendJSONLong("Sets the SABnzbd priority for an item (-1..2)."),
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			priority, err := strconv.Atoi(args[1])
			if err != nil {
				return err
			}
			if priority < -1 || priority > 2 {
				return errors.New("priority must be -1,0,1,2")
			}
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()
			return app.Client.QueueSetPriority(ctx, id, priority)
		},
	}
	return cmd
}

func queueItemMoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "move <nzo-id> <top|up|down|bottom|to> [position]",
		Short: jsonShort("Reorder queue items"),
		Long:  appendJSONLong("Moves a queue item relative to others or to an absolute position."),
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return errors.New("requires nzo-id and action")
			}
			if args[1] == "to" && len(args) < 3 {
				return errors.New("action 'to' requires a position")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			action := args[1]

			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			switch action {
			case "top", "bottom", "up", "down":
				params := url.Values{}
				params.Set("value", action)
				params.Set("value2", id)
				return app.Client.QueueAction(ctx, "move", params)
			case "to":
				pos, err := strconv.Atoi(args[2])
				if err != nil {
					return err
				}
				if pos < 0 {
					return errors.New("position must be zero or positive")
				}
				return app.Client.QueueSwitchPosition(ctx, id, pos)
			default:
				return fmt.Errorf("unknown move action %s", action)
			}
		},
	}
	return cmd
}

func queueItemSetCmd() *cobra.Command {
	var category string
	var script string
	var password string
	var name string

	cmd := &cobra.Command{
		Use:   "set <nzo-id>",
		Short: jsonShort("Update item metadata"),
		Long:  appendJSONLong("Adjust queue item category, script, display name, or password."),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if category == "" && script == "" && name == "" && password == "" {
				return errors.New("provide at least one field to update")
			}
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			if category != "" {
				if err := app.Client.QueueSetCategory(ctx, id, category); err != nil {
					return err
				}
			}
			if script != "" {
				if err := app.Client.QueueSetScript(ctx, id, script); err != nil {
					return err
				}
			}
			var renameName string
			if name != "" {
				renameName = name
			} else if password != "" {
				slot, err := findQueueSlot(ctx, app.Client, id)
				if err != nil {
					return err
				}
				renameName = slot.Filename
				if renameName == "" {
					return fmt.Errorf("cannot determine current name for %s; provide --name explicitly", id)
				}
			}
			if renameName != "" || password != "" {
				if err := app.Client.QueueRename(ctx, id, renameName, password); err != nil {
					return err
				}
			}

			effectiveName := renameName
			if effectiveName == "" {
				effectiveName = name
			}

			if app.Printer.JSON {
				payload := map[string]any{
					"nzo_id":       id,
					"category":     category,
					"script":       script,
					"password_set": password != "",
					"name":         effectiveName,
				}
				return app.Printer.Print(payload)
			}
			return app.Printer.Print("Updated item")
		},
	}

	cmd.Flags().StringVar(&category, "cat", "", "Category name")
	cmd.Flags().StringVar(&script, "script", "", "Post-processing script")
	cmd.Flags().StringVar(&password, "password", "", "Archive password")
	cmd.Flags().StringVar(&name, "name", "", "Rename the item")

	return cmd
}

func queueItemOptsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "opts <pp-level> <nzo-id> [nzo-id...]",
		Short: jsonShort("Update the post-processing level for specific items"),
		Long:  appendJSONLong("Sets the post-processing level for one or more queue items."),
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return errors.New("provide pp-level and at least one nzo-id")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			pp, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pp-level: %w", err)
			}
			ids := args[1:]

			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			if err := app.Client.QueueChangeOptions(ctx, ids, pp); err != nil {
				return err
			}

			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"pp_level": pp, "nzo_ids": ids})
			}
			return app.Printer.Print(fmt.Sprintf("Updated post-processing level for %s", strings.Join(ids, ",")))
		},
	}
	return cmd
}

func queueItemFilesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "files <nzo-id>",
		Short: jsonShort("List files for an item"),
		Long:  appendJSONLong("Lists NZF files belonging to a queue item."),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			files, err := app.Client.GetFiles(ctx, id)
			if err != nil {
				return err
			}

			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{
					"nzo_id": id,
					"files":  files,
					"count":  len(files),
				})
			}

			if len(files) == 0 {
				return app.Printer.Print(fmt.Sprintf("No files for %s", id))
			}

			headers := []string{"NZF ID", "Filename", "Status", "MB", "MB Left", "Age"}
			rows := make([][]string, 0, len(files))
			for _, file := range files {
				rows = append(rows, []string{
					file.NZFID,
					file.Filename,
					file.Status,
					file.MB,
					file.MBLeft,
					file.Age,
				})
			}
			if err := app.Printer.Table(headers, rows); err != nil {
				return err
			}
			return app.Printer.Print(fmt.Sprintf("%d files", len(files)))
		},
	}
	cmd.AddCommand(queueItemFilesDeleteCmd())
	cmd.AddCommand(queueItemFilesMoveCmd())
	return cmd
}

func queueItemFilesDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <nzo-id> <nzf-id>",
		Short: jsonShort("Delete a specific file from an item"),
		Long:  appendJSONLong("Deletes a single NZF file from a queue item."),
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			nzoID := args[0]
			nzfID := args[1]

			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			if err := app.Client.QueueDeleteFile(ctx, nzoID, nzfID); err != nil {
				return err
			}

			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{
					"nzo_id":  nzoID,
					"nzf_id":  nzfID,
					"deleted": true,
				})
			}
			return app.Printer.Print(fmt.Sprintf("Deleted %s from %s", nzfID, nzoID))
		},
	}
	return cmd
}

func queueItemFilesMoveCmd() *cobra.Command {
	var action string
	var ids []string
	var size int

	cmd := &cobra.Command{
		Use:   "move <nzo-id>",
		Short: jsonShort("Move files within an item's NZF list"),
		Long:  appendJSONLong("Bulk reorder NZF files within a queue item."),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nzoID := args[0]
			actionKey := strings.ToLower(strings.TrimSpace(action))
			if actionKey == "" {
				return errors.New("provide --action top|bottom|up|down")
			}
			if len(ids) == 0 {
				return errors.New("provide at least one NZF id via --id")
			}

			var sizePtr *int
			if actionKey == "up" || actionKey == "down" {
				if size <= 0 {
					return errors.New("--size must be specified and greater than zero for up/down moves")
				}
				sizePtr = &size
			}

			validActions := map[string]string{"top": "top", "bottom": "bottom", "up": "up", "down": "down"}
			actionValue, ok := validActions[actionKey]
			if !ok {
				return fmt.Errorf("unsupported action %q", actionKey)
			}

			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			if err := app.Client.QueueMoveFiles(ctx, actionValue, nzoID, ids, sizePtr); err != nil {
				return err
			}

			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{
					"nzo_id":  nzoID,
					"action":  actionValue,
					"nzf_ids": ids,
					"size":    size,
				})
			}
			return app.Printer.Print(fmt.Sprintf("Moved %s for %s", strings.Join(ids, ","), nzoID))
		},
	}

	cmd.Flags().StringVar(&action, "action", "", "Move direction (top, bottom, up, down)")
	cmd.Flags().StringSliceVar(&ids, "id", nil, "NZF ids to move (repeat for multiple)")
	cmd.Flags().IntVar(&size, "size", 0, "Number of positions to move when using up/down")
	return cmd
}

func queueSortCmd() *cobra.Command {
	var desc bool
	cmd := &cobra.Command{
		Use:   "sort <name|age|size|eta>",
		Short: jsonShort("Sort the queue"),
		Long:  appendJSONLong("Sorts SABnzbd's queue by the requested column."),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			criteria := args[0]
			sortKey, ok := map[string]string{
				"name": "name",
				"age":  "avg_age",
				"size": "size",
				"eta":  "eta",
			}[criteria]
			if !ok {
				return errors.New("unsupported sort criteria")
			}
			dir := "asc"
			if desc {
				dir = "desc"
			}
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()
			return app.Client.QueueSort(ctx, sortKey, dir)
		},
	}
	cmd.Flags().BoolVar(&desc, "desc", false, "Sort descending")
	return cmd
}

func findQueueSlot(ctx context.Context, client *sabapi.Client, id string) (*sabapi.QueueSlot, error) {
	queue, err := client.Queue(ctx, 0, 0, "")
	if err != nil {
		return nil, err
	}
	for _, slot := range queue.Slots {
		if slot.NZOID == id {
			return &slot, nil
		}
	}
	return nil, fmt.Errorf("item %s not found", id)
}
