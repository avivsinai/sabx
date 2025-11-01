package root

import "github.com/spf13/cobra"

func translateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "translate <text>",
		Short: jsonShort("Translate SABnzbd UI strings using the active locale"),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			phrase := args[0]
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			translated, err := app.Client.Translate(ctx, phrase)
			if err != nil {
				return err
			}

			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{
					"input":      phrase,
					"translated": translated,
				})
			}
			return app.Printer.Print(translated)
		},
	}
	return cmd
}
