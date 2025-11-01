package root

import (
	"fmt"

	"github.com/spf13/cobra"
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
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show global SABnzbd status",
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
			return app.Printer.Print(summary)
		},
	}

	return cmd
}
