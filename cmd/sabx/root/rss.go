package root

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sabx/sabx/internal/cobraext"
)

func rssCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rss",
		Short: "Manage SABnzbd RSS feeds",
	}

	cmd.AddCommand(rssListCmd())
	cmd.AddCommand(rssAddCmd())
	cmd.AddCommand(rssSetCmd())
	cmd.AddCommand(rssDeleteCmd())
	cmd.AddCommand(rssRunCmd())
	return cmd
}

func rssListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configured RSS feeds",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			payload, err := app.Client.RSSList(ctx)
			if err != nil {
				return err
			}
			feeds := parseRSSFeeds(payload)
			if app.Printer.JSON {
				return app.Printer.Print(feeds)
			}
			headers := []string{"Name", "URL", "Category", "Priority", "Enabled"}
			rows := make([][]string, 0, len(feeds))
			for _, feed := range feeds {
				rows = append(rows, []string{feed.Name, feed.URL, feed.Category, feed.Priority, fmt.Sprintf("%v", feed.Enabled)})
			}
			if err := app.Printer.Table(headers, rows); err != nil {
				return err
			}
			return app.Printer.Print(fmt.Sprintf("%d feeds", len(feeds)))
		},
	}
	return cmd
}

func rssAddCmd() *cobra.Command {
	var urlStr string
	var category string
	var priority string
	var enabled bool

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Create a new RSS feed",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if urlStr == "" {
				return errors.New("--url is required")
			}
			name := args[0]
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			props := map[string]string{
				"uri": urlStr,
			}
			if category != "" {
				props["cat"] = category
			}
			if priority != "" {
				props["priority"] = priority
			}
			props["enabled"] = boolToFlag(enabled)

			if err := applyRSSProperties(ctx, app, name, props); err != nil {
				return err
			}
			return app.Printer.Print(fmt.Sprintf("RSS feed %s created", name))
		},
	}

	cmd.Flags().StringVar(&urlStr, "url", "", "Feed URL")
	cmd.Flags().StringVar(&category, "cat", "", "Category to assign")
	cmd.Flags().StringVar(&priority, "priority", "", "Priority override")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "Enable the feed")
	return cmd
}

func rssSetCmd() *cobra.Command {
	var entries []string
	cmd := &cobra.Command{
		Use:   "set <name>",
		Short: "Update feed properties",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(entries) == 0 {
				return errors.New("provide at least one --set key=value pair")
			}
			name := args[0]
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			props := make(map[string]string)
			for _, entry := range entries {
				parts := strings.SplitN(entry, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid pair %q", entry)
				}
				props[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}

			if err := applyRSSProperties(ctx, app, name, props); err != nil {
				return err
			}

			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"name": name, "updated": props})
			}
			return app.Printer.Print("RSS feed updated")
		},
	}

	cmd.Flags().StringArrayVar(&entries, "set", nil, "Key=value pairs to set (uri, cat, priority, enabled, etc.)")
	return cmd
}

func rssDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete an RSS feed",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()
			if err := app.Client.ConfigDelete(ctx, "rss", name); err != nil {
				return err
			}
			return app.Printer.Print("RSS feed deleted")
		},
	}
	return cmd
}

func rssRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [name]",
		Short: "Run RSS fetch now",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var name string
			if len(args) == 1 {
				name = args[0]
			}
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()
			if err := app.Client.RSSNow(ctx, name); err != nil {
				return err
			}
			if name == "" {
				return app.Printer.Print("Triggered RSS for all feeds")
			}
			return app.Printer.Print(fmt.Sprintf("Triggered RSS for %s", name))
		},
	}
	return cmd
}

// parseRSSFeeds attempts to normalise SABnzbd rss config payloads.
type rssFeed struct {
	Name     string            `json:"name"`
	URL      string            `json:"url"`
	Category string            `json:"category"`
	Priority string            `json:"priority"`
	Enabled  bool              `json:"enabled"`
	Raw      map[string]string `json:"raw"`
}

func parseRSSFeeds(m map[string]any) []rssFeed {
	feeds := []rssFeed{}
	raw := extractValueMap(m)
	list, ok := raw["feeds"].([]any)
	if !ok {
		// Some SAB builds return map keyed by feed name.
		if keyed, ok := raw["feeds"].(map[string]any); ok {
			for name, payload := range keyed {
				feeds = append(feeds, rssFeedFrom(name, payload))
			}
			return feeds
		}
		// Fallback: iterate entire map
		for name, payload := range raw {
			if name == "feeds" {
				continue
			}
			feeds = append(feeds, rssFeedFrom(name, payload))
		}
		return feeds
	}
	for _, item := range list {
		feed := rssFeedFrom("", item)
		feeds = append(feeds, feed)
	}
	return feeds
}

func rssFeedFrom(defaultName string, payload any) rssFeed {
	feed := rssFeed{Raw: map[string]string{}}
	if defaultName != "" {
		feed.Name = defaultName
	}
	switch v := payload.(type) {
	case map[string]any:
		for key, value := range v {
			strVal := fmt.Sprintf("%v", value)
			feed.Raw[key] = strVal
			switch strings.ToLower(key) {
			case "name":
				feed.Name = strVal
			case "uri", "url":
				feed.URL = strVal
			case "cat", "category":
				feed.Category = strVal
			case "priority", "prio":
				feed.Priority = strVal
			case "enabled":
				feed.Enabled = isTruthy(strVal)
			}
		}
	}
	return feed
}

func extractValueMap(m map[string]any) map[string]any {
	if value, ok := m["value"]; ok {
		if asMap, ok := value.(map[string]any); ok {
			return asMap
		}
	}
	return m
}

func applyRSSProperties(ctx context.Context, app *cobraext.App, name string, props map[string]string) error {
	values := url.Values{}
	for key, val := range props {
		if val == "" {
			continue
		}
		values.Set(key, val)
	}
	if len(values) == 0 {
		return nil
	}
	if err := app.Client.ConfigSet(ctx, "rss", name, values); err != nil {
		return err
	}
	return nil
}

func boolToFlag(v bool) string {
	if v {
		return "1"
	}
	return "0"
}

func isTruthy(v string) bool {
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}
