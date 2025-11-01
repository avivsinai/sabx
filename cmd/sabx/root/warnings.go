package root

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func warningsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "warnings",
		Short: jsonShort("Inspect SABnzbd warning messages"),
	}
	cmd.AddCommand(warningsListCmd())
	cmd.AddCommand(warningsClearCmd())
	return cmd
}

func warningsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: jsonShort("List current warnings"),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			warnings, err := app.Client.Warnings(ctx)
			if err != nil {
				return err
			}

			if app.Printer.JSON {
				payload := map[string]any{
					"warnings": warnings,
					"count":    len(warnings),
				}
				return app.Printer.Print(payload)
			}

			if len(warnings) == 0 {
				return app.Printer.Print("No warnings")
			}

			headers := []string{"Time", "Type", "Message"}
			rows := make([][]string, 0, len(warnings))
			for _, w := range warnings {
				ts := time.Unix(w.Time, 0).Format(time.RFC3339)
				rows = append(rows, []string{
					ts,
					w.Type,
					strings.ReplaceAll(w.Text, "\n", " "),
				})
			}
			if err := app.Printer.Table(headers, rows); err != nil {
				return err
			}
			return app.Printer.Print(fmt.Sprintf("%d warnings", len(warnings)))
		},
	}
	return cmd
}

func warningsClearCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clear",
		Short: jsonShort("Clear stored warnings"),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			if err := app.Client.WarningsClear(ctx); err != nil {
				return err
			}

			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"cleared": true})
			}
			return app.Printer.Print("Cleared warnings")
		},
	}
	return cmd
}
