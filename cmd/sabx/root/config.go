package root

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Read and update SABnzbd configuration",
	}

	cmd.AddCommand(configGetCmd())
	cmd.AddCommand(configSetCmd())
	cmd.AddCommand(configDeleteCmd())
	return cmd
}

func configGetCmd() *cobra.Command {
	var key string
	cmd := &cobra.Command{
		Use:   "get <section>",
		Short: "Fetch configuration values",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			section := args[0]
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			cfg, err := app.Client.ConfigGet(ctx, section, key)
			if err != nil {
				return err
			}

			return app.Printer.Print(cfg)
		},
	}
	cmd.Flags().StringVar(&key, "key", "", "Specific keyword within the section")
	return cmd
}

func configSetCmd() *cobra.Command {
	var name string
	var entries []string
	cmd := &cobra.Command{
		Use:   "set <section>",
		Short: "Set configuration values",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(entries) == 0 {
				return errors.New("provide at least one --set key=value pair")
			}
			section := args[0]
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			for _, entry := range entries {
				parts := strings.SplitN(entry, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid --set entry %q", entry)
				}
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				if key == "" {
					return fmt.Errorf("invalid key in --set entry %q", entry)
				}
				values := url.Values{}
				values.Set("keyword", key)
				values.Add("value", val)
				if err := app.Client.ConfigSet(ctx, section, name, values); err != nil {
					return err
				}
			}

			if app.Printer.JSON {
				payload := map[string]any{"section": section, "name": name, "applied": entries}
				return app.Printer.Print(payload)
			}
			return app.Printer.Print("Config updated")
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Named configuration item (RSS feed, server, etc.)")
	cmd.Flags().StringArrayVar(&entries, "set", nil, "Key=value pairs (repeat for multiple keys)")
	return cmd
}

func configDeleteCmd() *cobra.Command {
	var name string
	var key string
	cmd := &cobra.Command{
		Use:   "delete <section>",
		Short: "Delete a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			section := args[0]
			if name == "" && key == "" {
				return errors.New("provide --key or --name")
			}
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			if name != "" {
				if err := app.Client.ConfigDelete(ctx, section, name); err != nil {
					return err
				}
			} else {
				if err := app.Client.ConfigDelete(ctx, section, key); err != nil {
					return err
				}
			}
			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"section": section, "name": name, "key": key})
			}
			return app.Printer.Print("Config deleted")
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Named configuration entry to delete")
	cmd.Flags().StringVar(&key, "key", "", "Keyword to delete (for un-named sections)")
	return cmd
}
