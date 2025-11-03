package root

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/avivsinai/sabx/internal/auth"
	"github.com/avivsinai/sabx/internal/config"
)

func loginCmd() *cobra.Command {
	var (
		baseURLFlagLocal   string
		apiKeyFlagLocal    string
		profileLocal       string
		setDefault         bool
		allowInsecureStore bool
		storeInConfig      bool
	)

	cmd := &cobra.Command{
		Use:   "login",
		Short: jsonShort("Authenticate sabx with a SABnzbd instance"),
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

			allowFallback := allowInsecureStore || auth.AllowInsecureStoreFromEnv()

			prof := config.Profile{
				BaseURL:            baseURL,
				AllowInsecureStore: allowFallback,
			}
			if storeInConfig {
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

			storeOpts := []auth.Option{}
			if allowFallback {
				storeOpts = append(storeOpts, auth.WithAllowFileFallback(true))
			}

			if !storeInConfig {
				if err := auth.SaveAPIKey(profile, baseURL, apiKey, storeOpts...); err != nil {
					return fmt.Errorf("failed to store api key securely: %w", err)
				}
			} else {
				// Best-effort cleanup in case a previous login wrote to the keyring.
				if err := auth.DeleteAPIKey(profile, baseURL, storeOpts...); err != nil && !errors.Is(err, auth.ErrNotFound) {
					fmt.Fprintf(cmd.ErrOrStderr(), "Warning: unable to remove keyring entry (%v)\n", err)
				}
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Saved profile %q (base URL: %s)\n", profile, baseURL)
			if allowFallback {
				fmt.Fprintln(cmd.OutOrStdout(), "Note: Encrypted file fallback enabled; consider disabling with --allow-insecure-store=false on trusted hosts.")
			}
			if storeInConfig {
				fmt.Fprintln(cmd.OutOrStdout(), "Warning: API key stored insecurely in config file.")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&baseURLFlagLocal, "base-url", "", "SABnzbd base URL (e.g., http://localhost:8080)")
	cmd.Flags().StringVar(&apiKeyFlagLocal, "api-key", "", "SABnzbd API key")
	cmd.Flags().StringVar(&profileLocal, "profile", "", "Profile name to associate with these credentials")
	cmd.Flags().BoolVar(&setDefault, "set-default", false, "Set this profile as the default")
	cmd.Flags().BoolVar(&allowInsecureStore, "allow-insecure-store", false, "Allow encrypted file-based storage when OS keychain is unavailable")
	cmd.Flags().BoolVar(&storeInConfig, "store-in-config", false, "Store API key in plaintext config file (discouraged)")

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
