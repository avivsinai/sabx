package root

import (
	"fmt"

	"github.com/spf13/cobra"
)

func doctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose connectivity issues",
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

			checks := map[string]string{}

			if version, err := app.Client.Version(ctx); err == nil {
				checks["version"] = version.Version
			} else {
				checks["version_error"] = err.Error()
			}

			if status, err := app.Client.Status(ctx); err == nil {
				checks["status"] = fmt.Sprintf("paused=%v speed=%s", status.Paused, status.Speed)
			} else {
				checks["status_error"] = err.Error()
			}

			if queue, err := app.Client.Queue(ctx, 0, 1, ""); err == nil {
				checks["queue_accessible"] = fmt.Sprintf("slots=%d", len(queue.Slots))
			} else {
				checks["queue_error"] = err.Error()
			}

			return app.Printer.Print(checks)
		},
	}
	return cmd
}
