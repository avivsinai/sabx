package root

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: jsonShort("Read and update SABnzbd configuration"),
	}

	cmd.AddCommand(configGetCmd())
	cmd.AddCommand(configSetCmd())
	cmd.AddCommand(configDeleteCmd())
	cmd.AddCommand(configSetPauseCmd())
	cmd.AddCommand(configRotateAPIKeyCmd())
	cmd.AddCommand(configRotateNZBKeyCmd())
	cmd.AddCommand(configRegenerateCertsCmd())
	cmd.AddCommand(configCreateBackupCmd())
	cmd.AddCommand(configPurgeLogsCmd())
	cmd.AddCommand(configResetDefaultCmd())
	return cmd
}

func configGetCmd() *cobra.Command {
	var key string
	cmd := &cobra.Command{
		Use:   "get <section>",
		Short: jsonShort("Fetch configuration values"),
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
		Short: jsonShort("Set configuration values"),
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
		Short: jsonShort("Delete a configuration value"),
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

func configSetPauseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-pause <minutes>",
		Short: jsonShort("Schedule SABnzbd to resume after the given minutes"),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			minutes, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid minutes: %w", err)
			}
			if minutes < 0 {
				return errors.New("minutes must be >= 0")
			}

			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			if err := app.Client.ConfigSetPause(ctx, minutes); err != nil {
				return err
			}

			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"minutes": minutes})
			}
			return app.Printer.Print(fmt.Sprintf("Queue will resume in %d minute(s)", minutes))
		},
	}
	return cmd
}

func configRotateAPIKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rotate-api-key",
		Short: jsonShort("Generate a new SABnzbd API key"),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			key, err := app.Client.ConfigRotateAPIKey(ctx)
			if err != nil {
				return err
			}
			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"api_key": key})
			}
			return app.Printer.Print(fmt.Sprintf("New API key: %s", key))
		},
	}
	return cmd
}

func configRotateNZBKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rotate-nzb-key",
		Short: jsonShort("Generate a new SABnzbd NZB key"),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			nzbKey, err := app.Client.ConfigRotateNZBKey(ctx)
			if err != nil {
				return err
			}
			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"nzb_key": nzbKey})
			}
			return app.Printer.Print(fmt.Sprintf("New NZB key: %s", nzbKey))
		},
	}
	return cmd
}

func configRegenerateCertsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "regenerate-certs",
		Short: jsonShort("Regenerate default HTTPS certificates"),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			regenerated, err := app.Client.ConfigRegenerateCertificates(ctx)
			if err != nil {
				return err
			}
			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"regenerated": regenerated})
			}
			if regenerated {
				return app.Printer.Print("Certificates regenerated; restart required")
			}
			return app.Printer.Print("Certificates unchanged (custom paths in use)")
		},
	}
	return cmd
}

func configCreateBackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: jsonShort("Create a SABnzbd configuration backup"),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			success, path, err := app.Client.ConfigCreateBackup(ctx)
			if err != nil {
				return err
			}
			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"success": success, "path": path})
			}
			if success {
				return app.Printer.Print(fmt.Sprintf("Backup saved to %s", path))
			}
			return app.Printer.Print("No backup created")
		},
	}
	return cmd
}

func configPurgeLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "purge-logs",
		Short: jsonShort("Purge historical SABnzbd log files"),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			if err := app.Client.ConfigPurgeLogFiles(ctx); err != nil {
				return err
			}
			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"purged": true})
			}
			return app.Printer.Print("Purged SABnzbd log files")
		},
	}
	return cmd
}

func configResetDefaultCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset-default <keyword> [keyword...]",
		Short: jsonShort("Reset misc configuration keys back to defaults"),
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("provide one or more keywords")
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

			if err := app.Client.ConfigSetDefault(ctx, args); err != nil {
				return err
			}
			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"reset": args})
			}
			return app.Printer.Print(fmt.Sprintf("Reset %s to defaults", strings.Join(args, ",")))
		},
	}
	return cmd
}
