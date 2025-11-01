package root

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

func speedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "speed",
		Short: "Inspect and control download speeds",
	}
	cmd.AddCommand(speedShowCmd())
	cmd.AddCommand(speedLimitCmd())
	return cmd
}

func speedShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Display current speed information",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			status, err := app.Client.Status(ctx)
			if err != nil {
				return err
			}
			queue, err := app.Client.Queue(ctx, 0, 0, "")
			if err != nil {
				return err
			}
			if app.Printer.JSON {
				payload := map[string]any{
					"speed_kbps":   status.Speed,
					"limit_kbps":   status.SpeedLimit,
					"paused":       status.Paused,
					"queue_speed":  queue.Speed,
					"queue_limit":  queue.SpeedLimit,
					"queue_paused": queue.Paused,
				}
				return app.Printer.Print(payload)
			}
			summary := fmt.Sprintf("Speed: %s KB/s (limit %s) paused=%v", status.Speed, status.SpeedLimit, status.Paused)
			return app.Printer.Print(summary)
		},
	}
	return cmd
}

func speedLimitCmd() *cobra.Command {
	var mbps float64
	var remove bool
	cmd := &cobra.Command{
		Use:   "limit",
		Short: "Set the global speed limit",
		RunE: func(cmd *cobra.Command, args []string) error {
			if mbps <= 0 && !remove {
				return errors.New("provide --mbps or --none")
			}
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			if remove {
				if err := app.Client.SpeedLimit(ctx, nil); err != nil {
					return err
				}
				if app.Printer.JSON {
					return app.Printer.Print(map[string]any{"limit": nil})
				}
				return app.Printer.Print("Speed limit removed")
			}

			value := mbps
			if err := app.Client.SpeedLimit(ctx, &value); err != nil {
				return err
			}
			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"mbps": mbps})
			}
			return app.Printer.Print(fmt.Sprintf("Speed limit set to %.2f Mbps", mbps))
		},
	}
	cmd.Flags().Float64Var(&mbps, "mbps", 0, "Megabits per second limit")
	cmd.Flags().BoolVar(&remove, "none", false, "Remove the limit")
	return cmd
}
