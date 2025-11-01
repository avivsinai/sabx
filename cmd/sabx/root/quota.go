package root

import "github.com/spf13/cobra"

func quotaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quota",
		Short: jsonShort("Manage SABnzbd download quota"),
	}
	cmd.AddCommand(quotaResetCmd())
	return cmd
}

func quotaResetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset",
		Short: jsonShort("Reset the download quota counters"),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			if err := app.Client.ResetQuota(ctx); err != nil {
				return err
			}
			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"quota_reset": true})
			}
			return app.Printer.Print("Quota reset")
		},
	}
	return cmd
}
