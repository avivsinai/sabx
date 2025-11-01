package root

import "github.com/spf13/cobra"

func serverCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Control SABnzbd server lifecycle",
	}
	cmd.AddCommand(serverRestartCmd())
	cmd.AddCommand(serverShutdownCmd())
	return cmd
}

func serverRestartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart SABnzbd",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()
			return app.Client.ServerControl(ctx, "restart")
		},
	}
	return cmd
}

func serverShutdownCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shutdown",
		Short: "Shutdown SABnzbd",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()
			return app.Client.ServerControl(ctx, "shutdown")
		},
	}
	return cmd
}
