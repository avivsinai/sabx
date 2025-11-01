package root

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/sabx/sabx/internal/auth"
	"github.com/sabx/sabx/internal/config"
	"github.com/sabx/sabx/internal/output"
)

func logoutCmd() *cobra.Command {
	var profileOverride string
	var removeProfile bool

	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove stored SABnzbd credentials",
		Annotations: map[string]string{
			"skipPersistent": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			profileName := firstNonEmpty(profileOverride, profileFlag)
			profileName = profileOrDefault(profileName)

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			prof, ok := cfg.GetProfile(profileName)
			if !ok {
				return fmt.Errorf("profile %q not found", profileName)
			}

			if err := auth.DeleteAPIKey(profileName, prof.BaseURL); err != nil && !errors.Is(err, auth.ErrNotFound) {
				return fmt.Errorf("failed to delete keyring entry: %w", err)
			}

			if removeProfile {
				delete(cfg.Profiles, profileName)
				if cfg.DefaultProfile == profileName {
					cfg.DefaultProfile = ""
				}
			} else {
				prof.APIKey = ""
				cfg.SetProfile(profileName, prof)
			}

			if err := cfg.Save(); err != nil {
				return err
			}

			printer := output.New()
			printer.JSON = jsonFlag
			printer.Quiet = quietFlag

			if printer.JSON {
				return printer.Print(map[string]any{"profile": profileName, "removed": removeProfile})
			}
			if removeProfile {
				return printer.Print(fmt.Sprintf("Removed profile %q", profileName))
			}
			return printer.Print(fmt.Sprintf("Removed credentials for profile %q", profileName))
		},
	}

	cmd.Flags().StringVar(&profileOverride, "profile", "", "Profile to logout (defaults to --profile global flag or 'default')")
	cmd.Flags().BoolVar(&removeProfile, "remove-profile", false, "Remove the profile entry from config")
	return cmd
}
