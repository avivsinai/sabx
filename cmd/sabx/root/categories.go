package root

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/avivsinai/sabx/internal/cobraext"
)

func categoriesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "categories",
		Short: jsonShort("Manage SABnzbd categories"),
	}

	cmd.AddCommand(categoriesListCmd())
	cmd.AddCommand(categoriesAddCmd())
	cmd.AddCommand(categoriesSetCmd())
	cmd.AddCommand(categoriesDeleteCmd())
	return cmd
}

func categoriesListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: jsonShort("List categories"),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()
			payload, err := app.Client.CategoriesList(ctx)
			if err != nil {
				return err
			}
			if app.Printer.JSON {
				return app.Printer.Print(payload)
			}
			cats := parseNamedConfig(payload)
			headers := []string{"Name", "Dir", "Script", "Priority"}
			rows := make([][]string, 0, len(cats))
			for _, cat := range cats {
				rows = append(rows, []string{cat.Name, cat.Values["dir"], cat.Values["script"], cat.Values["priority"]})
			}
			if err := app.Printer.Table(headers, rows); err != nil {
				return err
			}
			return app.Printer.Print(fmt.Sprintf("%d categories", len(cats)))
		},
	}
	return cmd
}

func categoriesAddCmd() *cobra.Command {
	var dir string
	var script string
	var priority string

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: jsonShort("Add a new category"),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()
			props := map[string]string{}
			if dir != "" {
				props["dir"] = dir
			}
			if script != "" {
				props["script"] = script
			}
			if priority != "" {
				props["priority"] = priority
			}
			if err := applyNamedProperties(ctx, app, "categories", name, props); err != nil {
				return err
			}
			return app.Printer.Print("Category added")
		},
	}

	cmd.Flags().StringVar(&dir, "dir", "", "Download directory override")
	cmd.Flags().StringVar(&script, "script", "", "Post-processing script")
	cmd.Flags().StringVar(&priority, "priority", "", "Priority override")
	return cmd
}

func categoriesSetCmd() *cobra.Command {
	var entries []string
	cmd := &cobra.Command{
		Use:   "set <name>",
		Short: jsonShort("Update a category"),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(entries) == 0 {
				return errors.New("provide at least one --set key=value pair")
			}
			props := make(map[string]string)
			for _, entry := range entries {
				parts := strings.SplitN(entry, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid pair %q", entry)
				}
				props[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
			name := args[0]
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()
			if err := applyNamedProperties(ctx, app, "categories", name, props); err != nil {
				return err
			}
			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"category": name, "updated": props})
			}
			return app.Printer.Print("Category updated")
		},
	}
	cmd.Flags().StringArrayVar(&entries, "set", nil, "Key=value pairs to update")
	return cmd
}

func categoriesDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: jsonShort("Delete a category"),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()
			if err := app.Client.ConfigDelete(ctx, "categories", name); err != nil {
				return err
			}
			return app.Printer.Print("Category deleted")
		},
	}
	return cmd
}

type namedConfig struct {
	Name   string
	Values map[string]string
}

func parseNamedConfig(m map[string]any) []namedConfig {
	result := []namedConfig{}
	value := extractValueMap(m)
	if items, ok := value["items"].([]any); ok {
		for _, item := range items {
			if asMap, ok := item.(map[string]any); ok {
				entry := namedConfig{Values: map[string]string{}}
				for key, raw := range asMap {
					str := fmt.Sprintf("%v", raw)
					if key == "name" {
						entry.Name = str
					} else {
						entry.Values[key] = str
					}
				}
				result = append(result, entry)
			}
		}
		return result
	}
	for key, raw := range value {
		entry := namedConfig{Name: key, Values: map[string]string{}}
		if asMap, ok := raw.(map[string]any); ok {
			for k, v := range asMap {
				entry.Values[k] = fmt.Sprintf("%v", v)
			}
		}
		result = append(result, entry)
	}
	return result
}

func applyNamedProperties(ctx context.Context, app *cobraext.App, section, name string, props map[string]string) error {
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
	if err := app.Client.ConfigSet(ctx, section, name, values); err != nil {
		return err
	}
	return nil
}
