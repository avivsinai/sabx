package root

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/sabx/sabx/internal/cobraext"
	"github.com/sabx/sabx/internal/sabapi"
)

func priorityLabel(priority string) string {
	switch priority {
	case "2":
		return "Force"
	case "1":
		return "High"
	case "0":
		return "Normal"
	case "-1":
		return "Low"
	default:
		return priority
	}
}

func statusCmd() *cobra.Command {
	var full bool
	var performance bool
	var skipDashboard bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: jsonShort("Show global SABnzbd status"),
		Long:  appendJSONLong("Summarize SABnzbd's queue and daemon status. Use --full for fullstatus payloads and --performance to include calculated metrics."),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			if app.Client == nil {
				return fmt.Errorf("not logged in; run 'sabx login'")
			}

			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			queue, err := app.Client.Queue(ctx, 0, 0, "")
			if err != nil {
				return err
			}
			status, err := app.Client.Status(ctx)
			if err != nil {
				return err
			}

			var fullStatus map[string]any
			if full || performance {
				opts := sabapi.FullStatusOptions{
					CalculatePerformance: performance,
					SkipDashboard:        skipDashboard,
				}
				fullStatus, err = app.Client.FullStatus(ctx, opts)
				if err != nil {
					return err
				}
			}

			if app.Printer.JSON {
				payload := map[string]any{
					"profile":      app.ProfileName,
					"base_url":     app.BaseURL,
					"queue_slots":  queue.Slots,
					"queue_status": queue.Status,
					"paused":       queue.Paused,
					"speed_kbps":   queue.Speed,
					"speed_limit":  queue.SpeedLimit,
					"size_mb":      queue.SizeMB,
					"mbleft":       queue.MBLeft,
					"timeleft":     queue.TimeLeft,
					"status":       status,
				}
				if fullStatus != nil {
					payload["full_status"] = fullStatus
					if servers, err := app.Client.ServerConfigs(ctx); err == nil {
						payload["servers"] = servers
					}
				}
				return app.Printer.Print(payload)
			}

			rows := [][]string{}
			for _, slot := range queue.Slots {
				rows = append(rows, []string{
					slot.NZOID,
					slot.Filename,
					slot.Status,
					fmt.Sprintf("%s/%s", slot.MB, slot.MBLeft),
					priorityLabel(slot.Priority),
				})
			}

			headers := []string{"ID", "Name", "Status", "MB Done/Left", "Prio"}
			if err := app.Printer.Table(headers, rows); err != nil {
				return err
			}

			summary := fmt.Sprintf("Queue: %d items | Speed %s KB/s (limit %s) | Time left %s",
				len(queue.Slots), queue.Speed, queue.SpeedLimit, queue.TimeLeft)
			if err := app.Printer.Print(summary); err != nil {
				return err
			}

			if fullStatus != nil {
				if err := renderFullStatus(cmd, app, fullStatus); err != nil {
					return err
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&full, "full", false, "Include comprehensive status data from SABnzbd")
	cmd.Flags().BoolVar(&performance, "performance", false, "Calculate performance metrics (implies --full)")
	cmd.Flags().BoolVar(&skipDashboard, "skip-dashboard", false, "Skip dashboard network diagnostics (with --full)")

	cmd.PreRun = func(cmd *cobra.Command, args []string) {
		if performance {
			full = true
		}
	}

	cmd.AddCommand(statusOrphansCmd())

	return cmd
}

func renderFullStatus(cmd *cobra.Command, app *cobraext.App, data map[string]any) error {
	infoRows := [][]string{}
	addRow := func(label string, value any) {
		if value == nil {
			return
		}
		infoRows = append(infoRows, []string{label, fmt.Sprint(value)})
	}

	addRow("Log Level", data["loglevel"])
	addRow("Download Dir", data["downloaddir"])
	addRow("Download Dir Speed", data["downloaddirspeed"])
	addRow("Complete Dir", data["completedir"])
	addRow("Complete Dir Speed", data["completedirspeed"])
	addRow("Internet Bandwidth", data["internetbandwidth"])
	addRow("Load Avg", data["loadavg"])
	addRow("Warnings", lenInterface(sliceFrom(data["warnings"])))

	if len(infoRows) > 0 {
		if err := app.Printer.Table([]string{"Metric", "Value"}, infoRows); err != nil {
			return err
		}
	}

	serverEntries, err := serversFromFullStatus(data["servers"])
	if err != nil || len(serverEntries) == 0 {
		return nil
	}

	sort.Slice(serverEntries, func(i, j int) bool {
		return serverEntries[i].Name < serverEntries[j].Name
	})

	headers := []string{"Server", "Active", "Connections", "SSL", "Warning", "Error"}
	rows := make([][]string, 0, len(serverEntries))
	for _, srv := range serverEntries {
		rows = append(rows, []string{
			srv.Name,
			boolToStr(srv.Active),
			fmt.Sprintf("%d/%d", srv.ActiveConn, srv.TotalConn),
			boolToStr(srv.SSL),
			srv.Warning,
			srv.Error,
		})
	}
	return app.Printer.Table(headers, rows)
}

type statusServerEntry struct {
	Name       string
	Active     bool
	ActiveConn int
	TotalConn  int
	SSL        bool
	Warning    string
	Error      string
}

func serversFromFullStatus(val any) ([]statusServerEntry, error) {
	raw := sliceFrom(val)
	results := make([]statusServerEntry, 0, len(raw))
	for _, entry := range raw {
		m, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		results = append(results, statusServerEntry{
			Name:       fmt.Sprint(m["servername"]),
			Active:     boolFrom(m["serveractive"]),
			ActiveConn: intFrom(m["serveractiveconn"]),
			TotalConn:  intFrom(m["servertotalconn"]),
			SSL:        boolFrom(m["serverssl"]),
			Warning:    fmt.Sprint(m["serverwarning"]),
			Error:      fmt.Sprint(m["servererror"]),
		})
	}
	return results, nil
}

func sliceFrom(val any) []any {
	switch v := val.(type) {
	case []any:
		return v
	case []map[string]any:
		out := make([]any, len(v))
		for i := range v {
			out[i] = v[i]
		}
		return out
	default:
		return nil
	}
}

func boolFrom(val any) bool {
	switch v := val.(type) {
	case bool:
		return v
	case string:
		return v == "True" || v == "true" || v == "1"
	case float64:
		return v != 0
	default:
		return false
	}
}

func intFrom(val any) int {
	switch v := val.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		var n int
		fmt.Sscanf(v, "%d", &n)
		return n
	default:
		return 0
	}
}

func lenInterface(val []any) int {
	return len(val)
}

func boolToStr(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

func statusOrphansCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "orphans",
		Short: jsonShort("Manage orphaned SABnzbd jobs"),
		Long:  appendJSONLong("Inspect or reconcile orphaned job folders reported by SABnzbd."),
	}
	cmd.AddCommand(statusOrphansListCmd())
	cmd.AddCommand(statusOrphansDeleteCmd())
	cmd.AddCommand(statusOrphansDeleteAllCmd())
	cmd.AddCommand(statusOrphansAddCmd())
	cmd.AddCommand(statusOrphansAddAllCmd())
	return cmd
}

func statusOrphansListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: jsonShort("List orphaned job folders"),
		Long:  appendJSONLong("Displays orphaned folders returned by SABnzbd fullstatus responses."),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			status, err := app.Client.FullStatus(ctx, sabapi.FullStatusOptions{})
			if err != nil {
				return err
			}

			foldersAny := sliceFrom(status["folders"])
			orphans := make([]string, 0, len(foldersAny))
			for _, entry := range foldersAny {
				if s, ok := entry.(string); ok {
					orphans = append(orphans, s)
				}
			}

			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"orphans": orphans})
			}
			if len(orphans) == 0 {
				return app.Printer.Print("No orphaned jobs")
			}
			rows := make([][]string, len(orphans))
			for i, folder := range orphans {
				rows[i] = []string{folder}
			}
			if err := app.Printer.Table([]string{"Folder"}, rows); err != nil {
				return err
			}
			return app.Printer.Print(fmt.Sprintf("%d orphaned jobs", len(orphans)))
		},
	}
	return cmd
}

func statusOrphansDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <path>",
		Short: jsonShort("Delete a specific orphaned job"),
		Long:  appendJSONLong("Instruct SABnzbd to delete the chosen orphaned folder."),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			if err := app.Client.StatusDeleteOrphan(ctx, args[0]); err != nil {
				return err
			}

			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"deleted": args[0]})
			}
			return app.Printer.Print(fmt.Sprintf("Deleted orphan %s", args[0]))
		},
	}
	return cmd
}

func statusOrphansDeleteAllCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete-all",
		Short: jsonShort("Delete all orphaned jobs"),
		Long:  appendJSONLong("Removes every orphaned folder reported by SABnzbd."),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			if err := app.Client.StatusDeleteAllOrphans(ctx); err != nil {
				return err
			}
			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"deleted_all": true})
			}
			return app.Printer.Print("Deleted all orphaned jobs")
		},
	}
	return cmd
}

func statusOrphansAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <path>",
		Short: jsonShort("Re-add a specific orphaned job"),
		Long:  appendJSONLong("Requests SABnzbd to requeue the provided orphaned folder."),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			if err := app.Client.StatusAddOrphan(ctx, args[0]); err != nil {
				return err
			}
			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"added": args[0]})
			}
			return app.Printer.Print(fmt.Sprintf("Re-added orphan %s", args[0]))
		},
	}
	return cmd
}

func statusOrphansAddAllCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-all",
		Short: jsonShort("Re-add all orphaned jobs"),
		Long:  appendJSONLong("Requeues every orphaned folder reported by SABnzbd."),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			if err := app.Client.StatusAddAllOrphans(ctx); err != nil {
				return err
			}
			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"added_all": true})
			}
			return app.Printer.Print("Re-added all orphaned jobs")
		},
	}
	return cmd
}
