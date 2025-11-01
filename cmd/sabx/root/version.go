package root

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"

	"github.com/sabx/sabx/internal/output"
)

var (
	buildVersion = "dev"
	buildCommit  = ""
	buildDate    = ""
)

func versionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print sabx version information",
		Annotations: map[string]string{
			"skipPersistent": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			info := map[string]string{
				"version": buildVersion,
				"commit":  buildCommit,
				"date":    buildDate,
			}
			if info["commit"] == "" || info["date"] == "" {
				if bi, ok := debug.ReadBuildInfo(); ok && bi != nil {
					info["version"] = bi.Main.Version
					for _, setting := range bi.Settings {
						switch setting.Key {
						case "vcs.revision":
							info["commit"] = setting.Value
						case "vcs.time":
							info["date"] = setting.Value
						}
					}
				}
			}

			printer := output.New()
			printer.JSON = jsonFlag
			printer.Quiet = quietFlag
			if printer.JSON {
				return printer.Print(info)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "sabx %s", info["version"])
			if info["commit"] != "" {
				fmt.Fprintf(cmd.OutOrStdout(), " (%s)", info["commit"])
			}
			if info["date"] != "" {
				fmt.Fprintf(cmd.OutOrStdout(), " built %s", info["date"])
			}
			fmt.Fprintln(cmd.OutOrStdout())
			return nil
		},
	}
	return cmd
}
