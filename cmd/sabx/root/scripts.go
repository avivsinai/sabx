package root

import (
	"fmt"

	"github.com/spf13/cobra"
)

func scriptsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scripts",
		Short: jsonShort("Manage SABnzbd post-processing scripts"),
	}
	cmd.AddCommand(scriptsListCmd())
	return cmd
}

func scriptsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: jsonShort("List available post-processing scripts"),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			scripts, err := app.Client.GetScripts(ctx)
			if err != nil {
				return err
			}

			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"scripts": scripts})
			}

			if len(scripts) == 0 {
				return app.Printer.Print("No scripts configured")
			}

			rows := make([][]string, 0, len(scripts))
			for _, script := range scripts {
				rows = append(rows, []string{script})
			}
			if err := app.Printer.Table([]string{"Script"}, rows); err != nil {
				return err
			}
			return app.Printer.Print(fmt.Sprintf("Total: %d scripts", len(scripts)))
		},
	}
	return cmd
}
