package root

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sabx/sabx/internal/auth"
	"github.com/sabx/sabx/internal/config"
)

func loginCmd() *cobra.Command {
	var (
		baseURLFlagLocal string
		apiKeyFlagLocal  string
		profileLocal     string
		setDefault       bool
		insecureStore    bool
	)

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate sabx with a SABnzbd instance",
		Long:  "Stores SABnzbd connection details and API key securely in the system keychain.",
		Annotations: map[string]string{
			"skipPersistent": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			baseURL := firstNonEmpty(baseURLFlagLocal, baseURLFlag)
			baseURL = strings.TrimSpace(baseURL)
			if baseURL == "" {
				return errors.New("--base-url is required")
			}

			if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
				baseURL = "http://" + baseURL
			}

			apiKey := firstNonEmpty(apiKeyFlagLocal, apiKeyFlag)
			apiKey = strings.TrimSpace(apiKey)
			if apiKey == "" {
				return errors.New("--api-key is required")
			}

			profile := firstNonEmpty(profileLocal, profileFlag)
			profile = profileOrDefault(profile)

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			prof := config.Profile{BaseURL: baseURL}
			if insecureStore {
				prof.APIKey = apiKey
			}
			cfg.SetProfile(profile, prof)
			if setDefault {
				cfg.DefaultProfile = profile
			} else if cfg.DefaultProfile == "" {
				cfg.DefaultProfile = profile
			}

			if err := cfg.Save(); err != nil {
				return err
			}

			if !insecureStore {
				if err := auth.SaveAPIKey(profile, baseURL, apiKey); err != nil {
					return fmt.Errorf("failed to store api key in keyring: %w", err)
				}
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Saved profile %q (base URL: %s)\n", profile, baseURL)
			if insecureStore {
				fmt.Fprintln(cmd.OutOrStdout(), "Warning: API key stored insecurely in config file.")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&baseURLFlagLocal, "base-url", "", "SABnzbd base URL (e.g., http://localhost:8080)")
	cmd.Flags().StringVar(&apiKeyFlagLocal, "api-key", "", "SABnzbd API key")
	cmd.Flags().StringVar(&profileLocal, "profile", "", "Profile name to associate with these credentials")
	cmd.Flags().BoolVar(&setDefault, "set-default", false, "Set this profile as the default")
	cmd.Flags().BoolVar(&insecureStore, "insecure-store", false, "Store API key in config file instead of keychain")

	return cmd
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
