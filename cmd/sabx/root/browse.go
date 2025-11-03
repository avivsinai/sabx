package root

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/avivsinai/sabx/internal/sabapi"
)

func browseCmd() *cobra.Command {
	var showFiles bool
	var showHidden bool
	var compact bool

	cmd := &cobra.Command{
		Use:   "browse [path]",
		Short: jsonShort("Browse filesystem paths on the SABnzbd host"),
		Long:  appendJSONLong("Inspect directories exposed by SABnzbd. Combine flags like --files or --compact to tailor the response. Errors surface if SABnzbd refuses a path or the API call fails."),
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := ""
			if len(args) == 1 {
				path = args[0]
			}

			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			opts := sabapi.BrowseOptions{
				Compact:           compact,
				ShowFiles:         showFiles,
				ShowHiddenFolders: showHidden,
			}

			entries, err := app.Client.Browse(ctx, path, opts)
			if err != nil {
				return err
			}

			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{
					"path":    path,
					"entries": entries,
				})
			}

			if len(entries) == 0 {
				if path == "" {
					return app.Printer.Print("No entries")
				}
				return app.Printer.Print(fmt.Sprintf("No entries for %s", path))
			}

			var current string
			rows := make([][]string, 0, len(entries))
			for _, entry := range entries {
				if entry.CurrentPath != "" {
					current = entry.CurrentPath
					continue
				}
				kind := "File"
				if entry.Dir {
					kind = "Dir"
				}
				rows = append(rows, []string{
					entry.Name,
					entry.Path,
					kind,
				})
			}

			if current != "" {
				if errs := app.Printer.Print(fmt.Sprintf("Current: %s", current)); errs != nil {
					return errs
				}
			}

			if len(rows) == 0 {
				return app.Printer.Print("No entries")
			}

			return app.Printer.Table([]string{"Name", "Path", "Type"}, rows)
		},
	}

	cmd.Flags().BoolVar(&showFiles, "files", false, "Include files in results")
	cmd.Flags().BoolVar(&showHidden, "hidden", false, "Include hidden folders")
	cmd.Flags().BoolVar(&compact, "compact", false, "Return compact results (path strings only)")

	return cmd
}
