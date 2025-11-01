package root

import "github.com/spf13/cobra"

func watchedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watched",
		Short: jsonShort("Manage SABnzbd watched-folder scanning"),
		Long:  appendJSONLong("Trigger SABnzbd's watched folder scan or automate it. API errors bubble up if the request fails."),
	}

	cmd.AddCommand(watchedScanCmd())
	return cmd
}

func watchedScanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: jsonShort("Trigger an immediate watched-folder scan"),
		Long:  appendJSONLong("Requests SABnzbd to process new files in the watched directory immediately."),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			if err := app.Client.WatchedNow(ctx); err != nil {
				return err
			}

			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"triggered": true})
			}
			return app.Printer.Print("Watched folder scan triggered")
		},
	}
	return cmd
}
