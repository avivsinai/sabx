package root

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"

	"github.com/avivsinai/sabx/internal/buildinfo"
	"github.com/avivsinai/sabx/internal/output"
)

func versionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: jsonShort("Print sabx version information"),
		Annotations: map[string]string{
			"skipPersistent": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			info := currentBuildInfo()

			printer := output.New()
			printer.JSON = jsonFlag
			printer.Quiet = quietFlag
			if printer.JSON {
				return printer.Print(info)
			}
			fmt.Fprintln(cmd.OutOrStdout(), humanVersion(info))
			return nil
		},
	}
	return cmd
}

func currentBuildInfo() map[string]string {
	info := map[string]string{
		"version": buildinfo.Version,
		"commit":  buildinfo.Commit,
		"date":    buildinfo.Date,
	}
	if info["commit"] == "" || info["date"] == "" {
		if bi, ok := debug.ReadBuildInfo(); ok && bi != nil {
			if info["version"] == "" || info["version"] == "dev" {
				info["version"] = bi.Main.Version
			}
			for _, setting := range bi.Settings {
				switch setting.Key {
				case "vcs.revision":
					if info["commit"] == "" {
						info["commit"] = setting.Value
					}
				case "vcs.time":
					if info["date"] == "" {
						info["date"] = setting.Value
					}
				}
			}
		}
	}
	if info["version"] == "" {
		info["version"] = "dev"
	}
	return info
}

func humanVersion(info map[string]string) string {
	builder := strings.Builder{}
	builder.WriteString("sabx ")
	builder.WriteString(info["version"])
	if commit := info["commit"]; commit != "" {
		if len(commit) > 7 {
			commit = commit[:7]
		}
		builder.WriteString(" (")
		builder.WriteString(commit)
		builder.WriteString(")")
	}
	if info["date"] != "" {
		builder.WriteString(" built ")
		builder.WriteString(info["date"])
	}
	return builder.String()
}
