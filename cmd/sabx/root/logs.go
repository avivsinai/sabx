package root

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sabx/sabx/internal/sabapi"
)

const defaultTailLines = 50

func logsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: jsonShort("Inspect SABnzbd application logs"),
		Long:  appendJSONLong("Display recent SABnzbd log lines for troubleshooting."),
	}
	cmd.AddCommand(logsListCmd())
	cmd.AddCommand(logsTailCmd())
	return cmd
}

func logsListCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"show"},
		Short:   jsonShort("List log lines (optionally limited)"),
		Long:    appendJSONLong("Fetches SABnzbd's sanitized log output. Combine with --lines to constrain results."),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			lines, _, err := fetchLogTail(cmd.Context(), app.Client, limit)
			if err != nil {
				return err
			}

			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{
					"lines": lines,
					"count": len(lines),
				})
			}
			return app.Printer.Print(strings.Join(lines, "\n"))
		},
	}
	cmd.Flags().IntVar(&limit, "lines", 0, "Only show the last N lines")
	return cmd
}

func logsTailCmd() *cobra.Command {
	var limit int
	var follow bool
	var interval time.Duration
	cmd := &cobra.Command{
		Use:   "tail",
		Short: jsonShort("Tail the end of the log"),
		Long:  appendJSONLong("Streams the most recent SABnzbd log lines. When --follow is enabled, disable --json to avoid incompatible streaming output."),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			if limit <= 0 {
				limit = defaultTailLines
			}
			if follow && app.Printer.JSON {
				return errors.New("follow mode is not compatible with --json")
			}

			ctx := cmd.Context()

			lines, total, err := fetchLogTail(ctx, app.Client, limit)
			if err != nil {
				return err
			}

			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{
					"lines": lines,
					"count": len(lines),
				})
			}

			if err := app.Printer.Print(strings.Join(lines, "\n")); err != nil {
				return err
			}

			if !follow {
				return nil
			}

			lastTotal := total
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return nil
				case <-ticker.C:
					lines, currentTotal, err := fetchLogTail(ctx, app.Client, limit)
					if err != nil {
						return err
					}

					if currentTotal < lastTotal {
						// Log rotated or truncated; print fresh tail.
						if len(lines) > 0 {
							if err := app.Printer.Print(strings.Join(lines, "\n")); err != nil {
								return err
							}
						}
						lastTotal = currentTotal
						continue
					}

					if currentTotal > lastTotal {
						delta := currentTotal - lastTotal
						toPrint := lines
						if delta < len(lines) {
							toPrint = lines[len(lines)-delta:]
						}
						if len(toPrint) > 0 {
							if err := app.Printer.Print(strings.Join(toPrint, "\n")); err != nil {
								return err
							}
						}
						lastTotal = currentTotal
					}
				}
			}
		},
	}
	cmd.Flags().IntVar(&limit, "lines", defaultTailLines, "Number of lines to display")
	cmd.Flags().BoolVar(&follow, "follow", false, "Poll for new log lines")
	cmd.Flags().DurationVar(&interval, "interval", 2*time.Second, "Polling interval for follow mode")
	return cmd
}

func fetchLogTail(ctx context.Context, client *sabapi.Client, limit int) ([]string, int, error) {
	reqCtx, cancel := timeoutContext(ctx)
	defer cancel()

	data, err := client.ShowLog(reqCtx)
	if err != nil {
		return nil, 0, err
	}

	fullLines := splitLogLines(data)
	total := len(fullLines)

	out := fullLines
	if limit > 0 && len(fullLines) > limit {
		out = fullLines[len(fullLines)-limit:]
	}
	return out, total, nil
}

func splitLogLines(data string) []string {
	data = strings.ReplaceAll(data, "\r\n", "\n")
	lines := strings.Split(data, "\n")
	// Drop trailing empty line if the log ended with newline.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
