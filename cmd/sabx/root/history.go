package root

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sabx/sabx/internal/sabapi"
)

func historyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history",
		Short: jsonShort("Inspect past downloads"),
	}

	cmd.AddCommand(historyListCmd())
	cmd.AddCommand(historyDeleteCmd())
	cmd.AddCommand(historyRetryCmd())
	cmd.AddCommand(historyMarkCompletedCmd())

	return cmd
}

func historyListCmd() *cobra.Command {
	var limit int
	var failedOnly bool
	var completedOnly bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: jsonShort("List history entries"),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			history, err := app.Client.History(ctx, failedOnly, limit)
			if err != nil {
				return err
			}

			slots := history.Slots
			if completedOnly {
				filtered := make([]sabapi.HistorySlot, 0, len(slots))
				for _, slot := range slots {
					if strings.EqualFold(slot.Status, "Completed") {
						filtered = append(filtered, slot)
					}
				}
				slots = filtered
			}
			if app.Printer.JSON {
				return app.Printer.Print(slots)
			}

			headers := []string{"ID", "Name", "Status", "Category"}
			rows := make([][]string, 0, len(slots))
			for _, slot := range slots {
				rows = append(rows, []string{slot.NZOID, slot.Name, slot.Status, slot.Category})
			}
			if err := app.Printer.Table(headers, rows); err != nil {
				return err
			}
			return app.Printer.Print(fmt.Sprintf("%d history entries", len(slots)))
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 0, "Limit number of rows")
	cmd.Flags().BoolVar(&failedOnly, "failed", false, "Only show failed items")
	cmd.Flags().BoolVar(&completedOnly, "completed", false, "Only show completed items")
	return cmd
}

func historyDeleteCmd() *cobra.Command {
	var deleteAll bool
	var deleteFailed bool

	cmd := &cobra.Command{
		Use:   "delete [nzo-id ...]",
		Short: jsonShort("Delete history entries"),
		Args: func(cmd *cobra.Command, args []string) error {
			if deleteAll || deleteFailed {
				return nil
			}
			if len(args) == 0 {
				return errors.New("provide at least one nzo-id or use --all/--failed")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			if err := app.Client.DeleteHistory(ctx, args, deleteFailed, deleteAll); err != nil {
				return err
			}

			target := "selected entries"
			if deleteAll {
				target = "all entries"
			} else if deleteFailed {
				target = "failed entries"
			} else {
				target = strings.Join(args, ",")
			}

			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"deleted": target})
			}
			return app.Printer.Print(fmt.Sprintf("Deleted %s", target))
		},
	}

	cmd.Flags().BoolVar(&deleteAll, "all", false, "Delete entire history")
	cmd.Flags().BoolVar(&deleteFailed, "failed", false, "Delete only failed items")
	return cmd
}

func historyRetryCmd() *cobra.Command {
	var retryAll bool
	cmd := &cobra.Command{
		Use:   "retry [nzo-id]",
		Short: jsonShort("Re-queue history entries"),
		Args: func(cmd *cobra.Command, args []string) error {
			if retryAll {
				if len(args) > 0 {
					return errors.New("do not provide IDs when using --all")
				}
				return nil
			}
			if len(args) != 1 {
				return errors.New("provide an nzo-id or use --all")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()
			if retryAll {
				if err := app.Client.HistoryRetryAll(ctx); err != nil {
					return err
				}
				return app.Printer.Print("Re-queued all failed history entries")
			}
			if err := app.Client.HistoryRetry(ctx, args[0]); err != nil {
				return err
			}
			return app.Printer.Print(fmt.Sprintf("Re-queued %s", args[0]))
		},
	}
	cmd.Flags().BoolVar(&retryAll, "all", false, "Retry all failed history entries")
	return cmd
}

func historyMarkCompletedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mark-completed <nzo-id> [nzo-id...]",
		Short: jsonShort("Mark history entries as completed"),
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("provide at least one nzo-id")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			if err := app.Client.HistoryMarkCompleted(ctx, args); err != nil {
				return err
			}

			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"marked": args})
			}
			return app.Printer.Print(fmt.Sprintf("Marked %s as completed", strings.Join(args, ",")))
		},
	}
	return cmd
}
