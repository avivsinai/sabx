package root

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/avivsinai/sabx/internal/sabapi"
)

func debugCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: jsonShort("Developer utilities and diagnostics"),
	}
	cmd.AddCommand(debugGCStatsCmd())
	cmd.AddCommand(debugEvalSortCmd())
	return cmd
}

func debugGCStatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gc-stats",
		Short: jsonShort("Fetch SABnzbd garbage-collector statistics"),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			stats, err := app.Client.GCStats(ctx)
			if err != nil {
				return err
			}

			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"objects": stats})
			}

			count := len(stats)
			preview := stats
			if len(preview) > 5 {
				preview = preview[:5]
			}
			message := fmt.Sprintf("%d GC-tracked entries", count)
			if len(preview) > 0 {
				message = fmt.Sprintf("%s\nSample:%s", message, strings.Join(preview, "\n"))
			}
			if count > len(preview) {
				message += "\n(Use --json for full output)"
			}
			return app.Printer.Print(message)
		},
	}
	return cmd
}

func debugEvalSortCmd() *cobra.Command {
	var job string
	var label string

	cmd := &cobra.Command{
		Use:   "eval-sort <expression>",
		Short: jsonShort("Evaluate a sorting expression"),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			expr := args[0]
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			result, err := app.Client.EvalSort(ctx, expr, sabapi.EvalSortOptions{JobName: job, MultipartLabel: label})
			if err != nil {
				return err
			}

			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{
					"expression": expr,
					"result":     result,
				})
			}
			return app.Printer.Print(result)
		},
	}

	cmd.Flags().StringVar(&job, "job", "", "Sample job name for the evaluation")
	cmd.Flags().StringVar(&label, "label", "", "Multipart label for the evaluation")
	return cmd
}
