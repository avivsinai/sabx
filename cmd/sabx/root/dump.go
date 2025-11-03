package root

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/sabx/internal/cobraext"
)

func dumpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dump",
		Short: jsonShort("Export SABnzbd configuration or runtime state"),
	}

	cmd.AddCommand(dumpConfigCmd())
	cmd.AddCommand(dumpStateCmd())
	return cmd
}

func dumpConfigCmd() *cobra.Command {
	var sections []string
	cmd := &cobra.Command{
		Use:   "config",
		Short: jsonShort("Dump configuration sections (sanitised)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			if app.Client == nil {
				return errors.New("not logged in; run 'sabx login'")
			}

			if len(sections) == 0 {
				sections = []string{"misc", "servers", "rss", "categories", "scheduler"}
			}

			result := map[string]any{}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			for _, section := range sections {
				raw, err := app.Client.ConfigGet(ctx, section, "")
				if err != nil {
					return err
				}
				result[section] = sanitiseConfig(raw)
			}

			return printJSONorText(app, result)
		},
	}
	cmd.Flags().StringSliceVar(&sections, "section", nil, "Specific config sections to dump")
	return cmd
}

func dumpStateCmd() *cobra.Command {
	var historyLimit int
	cmd := &cobra.Command{
		Use:   "state",
		Short: jsonShort("Dump current queue/status/history snapshot"),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			if app.Client == nil {
				return errors.New("not logged in; run 'sabx login'")
			}
			if historyLimit <= 0 {
				historyLimit = 20
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			queue, err := app.Client.Queue(ctx, 0, 0, "")
			if err != nil {
				return err
			}
			status, err := app.Client.Status(ctx)
			if err != nil {
				return err
			}
			history, err := app.Client.History(ctx, false, historyLimit)
			if err != nil {
				return err
			}

			payload := map[string]any{
				"timestamp": time.Now().UTC().Format(time.RFC3339),
				"status":    status,
				"queue":     queue,
				"history":   history.Slots,
			}

			return printJSONorText(app, payload)
		},
	}
	cmd.Flags().IntVar(&historyLimit, "history", 20, "Number of history items to include")
	return cmd
}

func sanitiseConfig(raw map[string]any) map[string]any {
	masked := map[string]any{}
	for key, value := range raw {
		masked[key] = maskValue(key, value)
	}
	return masked
}

func maskValue(key string, value any) any {
	lower := strings.ToLower(key)
	if strings.Contains(lower, "key") || strings.Contains(lower, "secret") || strings.Contains(lower, "password") {
		switch value.(type) {
		case string:
			return "***"
		}
	}
	switch typed := value.(type) {
	case map[string]any:
		nested := map[string]any{}
		for k, v := range typed {
			nested[k] = maskValue(k, v)
		}
		return nested
	case []any:
		arr := make([]any, 0, len(typed))
		for _, item := range typed {
			arr = append(arr, maskValue(key, item))
		}
		return arr
	default:
		return value
	}
}

func printJSONorText(app *cobraext.App, payload any) error {
	if app.Printer.JSON {
		return app.Printer.Print(payload)
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return app.Printer.Print(string(data))
}
