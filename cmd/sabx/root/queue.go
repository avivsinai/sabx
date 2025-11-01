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
		Short: "Inspect and control the active queue",
	}

	cmd.AddCommand(queueListCmd())
	cmd.AddCommand(queueAddCmd())
	cmd.AddCommand(queuePauseCmd())
	cmd.AddCommand(queueResumeCmd())
	cmd.AddCommand(queuePurgeCmd())
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
		Short: "Show queue entries",
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
		Short: "Add NZBs to the queue",
	}

	cmd.AddCommand(queueAddURLCmd())
	cmd.AddCommand(queueAddFileCmd())

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
		Short: "Add an NZB by URL",
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
		Short: "Upload an NZB file",
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
		Short: "Pause the entire queue",
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
		Short: "Resume the entire queue",
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
		Short: "Purge queue entries",
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
			if purgeAll {
				// no additional params required; SAB interprets empty purge as full purge
			}
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

func queueItemCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "item",
		Short: "Operate on individual queue items",
	}

	cmd.AddCommand(queueItemShowCmd())
	cmd.AddCommand(queueItemPauseCmd())
	cmd.AddCommand(queueItemResumeCmd())
	cmd.AddCommand(queueItemDeleteCmd())
	cmd.AddCommand(queueItemPriorityCmd())
	cmd.AddCommand(queueItemMoveCmd())
	cmd.AddCommand(queueItemSetCmd())

	return cmd
}

func queueItemShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <nzo-id>",
		Short: "Show detailed information for an item",
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
		Short: "Pause an item",
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
		Short: "Resume an item",
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
		Short: "Delete an item",
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
		Short: "Change item priority",
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
		Short: "Reorder queue items",
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
		Short: "Update item metadata",
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

func queueSortCmd() *cobra.Command {
	var desc bool
	cmd := &cobra.Command{
		Use:   "sort <name|age|size|eta>",
		Short: "Sort the queue",
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
