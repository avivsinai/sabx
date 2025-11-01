package root

import (
	"fmt"

	"github.com/spf13/cobra"
)

func whoamiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "whoami",
		Short: jsonShort("Show the connected SABnzbd instance"),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			if app.Client == nil {
				return fmt.Errorf("no active session; run 'sabx login'")
			}

			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			version, err := app.Client.Version(ctx)
			if err != nil {
				return err
			}
			status, err := app.Client.Status(ctx)
			if err != nil {
				return err
			}

			if app.Printer.JSON {
				payload := map[string]any{
					"profile":     app.ProfileName,
					"base_url":    app.BaseURL,
					"version":     version.Version,
					"paused":      status.Paused,
					"speed_kbps":  status.Speed,
					"speed_limit": status.SpeedLimit,
				}
				return app.Printer.Print(payload)
			}

			return app.Printer.Print(fmt.Sprintf("%s (%s) paused=%v speed=%sKB/s limit=%sKB/s", app.BaseURL, version.Version, status.Paused, status.Speed, status.SpeedLimit))
		},
	}

	return cmd
}
