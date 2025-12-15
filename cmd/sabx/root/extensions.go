package root

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/avivsinai/sabx/internal/extensions"
)

func extensionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "extension",
		Short: jsonShort("Manage sabx extensions (sabx-<name>)"),
		Annotations: map[string]string{
			"skipPersistent": "true",
		},
	}

	cmd.AddCommand(extensionListCmd())
	cmd.AddCommand(extensionInstallCmd())
	cmd.AddCommand(extensionRemoveCmd())
	return cmd
}

func extensionListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: jsonShort("List installed extensions"),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			exts, err := extensions.List()
			if err != nil {
				return err
			}

			if app.Printer.JSON {
				return app.Printer.Print(exts)
			}

			if len(exts) == 0 {
				return app.Printer.Print("No extensions installed")
			}

			headers := []string{"Name", "Binary", "Kind", "Source"}
			rows := make([][]string, 0, len(exts))
			for _, ext := range exts {
				rows = append(rows, []string{
					ext.Name,
					ext.Binary,
					ext.Kind,
					ext.Source,
				})
			}
			if err := app.Printer.Table(headers, rows); err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}

func extensionInstallCmd() *cobra.Command {
	var overwrite bool
	cmd := &cobra.Command{
		Use:   "install <source>",
		Short: jsonShort("Install an extension from GitHub (owner/repo) or local path"),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			ext, err := extensions.Install(args[0], overwrite)
			if err != nil {
				return err
			}

			if app.Printer.JSON {
				return app.Printer.Print(ext)
			}
			return app.Printer.Print(fmt.Sprintf("Installed extension %s (%s)", ext.Name, ext.Source))
		},
	}
	cmd.Flags().BoolVar(&overwrite, "force", false, "Overwrite if the extension already exists")
	return cmd
}

func extensionRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: jsonShort("Remove an installed extension"),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			if err := extensions.Remove(args[0]); err != nil {
				return err
			}
			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"removed": args[0]})
			}
			return app.Printer.Print(fmt.Sprintf("Removed extension %s", args[0]))
		},
	}
	return cmd
}

func extensionExecFallback(name string, args []string) error {
	if err := extensions.Exec(name, args); err != nil {
		return fmt.Errorf("extension %s: %w", name, err)
	}
	return nil
}
