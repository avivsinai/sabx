package root

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/sabx/sabx/internal/auth"
	"github.com/sabx/sabx/internal/cobraext"
	"github.com/sabx/sabx/internal/config"
	"github.com/sabx/sabx/internal/extensions"
	"github.com/sabx/sabx/internal/output"
	"github.com/sabx/sabx/internal/sabapi"
)

var (
	profileFlag string
	baseURLFlag string
	apiKeyFlag  string
	jsonFlag    bool
	quietFlag   bool
	envConfig   = viper.New()
)

var rootCmd = &cobra.Command{
	Use:   "sabx",
	Short: "Full-fidelity SABnzbd CLI",
	Long:  "sabx is a fast, scriptable CLI that mirrors the SABnzbd web UI and API.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		printer := output.New()
		printer.JSON = jsonFlag
		printer.Quiet = quietFlag

		app := &cobraext.App{
			Config:  cfg,
			Printer: printer,
		}

		if cmd.Annotations["skipPersistent"] != "true" {
			profileName, baseURL, apiKey, err := resolveConnection(cfg)
			if err != nil {
				return err
			}
			app.ProfileName = profileName

			if baseURL != "" && apiKey != "" {
				client, err := sabapi.NewClient(baseURL, apiKey)
				if err != nil {
					return err
				}
				app.Client = client
				app.BaseURL = baseURL
			}
		}

		ctx := cobraext.WithApp(cmd.Context(), app)
		cmd.SetContext(ctx)
		return nil
	},
}

func init() {
	cobra.EnableCommandSorting = true
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true

	envConfig.SetEnvPrefix("SABX")
	envConfig.AutomaticEnv()

	rootCmd.PersistentFlags().StringVar(&profileFlag, "profile", "", "Profile name (defaults to config default)")
	rootCmd.PersistentFlags().StringVar(&baseURLFlag, "base-url", "", "Override SABnzbd base URL")
	rootCmd.PersistentFlags().StringVar(&apiKeyFlag, "api-key", "", "Override SABnzbd API key")
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Emit JSON output")
	rootCmd.PersistentFlags().BoolVar(&quietFlag, "quiet", false, "Only print errors")

	rootCmd.AddCommand(loginCmd())
	rootCmd.AddCommand(whoamiCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(queueCmd())
	rootCmd.AddCommand(historyCmd())
	rootCmd.AddCommand(configCmd())
	rootCmd.AddCommand(rssCmd())
	rootCmd.AddCommand(categoriesCmd())
	rootCmd.AddCommand(scheduleCmd())
	rootCmd.AddCommand(serverCmd())
	rootCmd.AddCommand(speedCmd())
	rootCmd.AddCommand(dumpCmd())
	rootCmd.AddCommand(topCmd())
	rootCmd.AddCommand(extensionsCmd())
	rootCmd.AddCommand(completionCmd())
	rootCmd.AddCommand(doctorCmd())
	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(logoutCmd())
}

// Execute runs the CLI.
func Execute() error {
	return ExecuteWithArgs(os.Args[1:])
}

// ExecuteWithArgs exposes execution for testing and extension fallback.
func ExecuteWithArgs(args []string) error {
	rootCmd.SetArgs(args)
	_, err := rootCmd.ExecuteC()
	if err == nil {
		return nil
	}

	if isUnknownCommandError(err) {
		name, extArgs, ok := extensions.ExtractExtensionCommand(args)
		if ok && name != "" {
			if execErr := extensionExecFallback(name, extArgs); execErr == nil {
				return nil
			} else {
				if !quietFlag {
					fmt.Fprintln(os.Stderr, execErr)
				}
				return execErr
			}
		}
	}

	if !quietFlag {
		fmt.Fprintln(os.Stderr, err)
	}
	return err
}

func isUnknownCommandError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "unknown command")
}

func resolveConnection(cfg *config.Config) (profile, baseURL, apiKey string, err error) {
	baseURL = strings.TrimSpace(baseURLFlag)
	apiKey = strings.TrimSpace(apiKeyFlag)

	if env := strings.TrimSpace(envConfig.GetString("BASE_URL")); baseURL == "" && env != "" {
		baseURL = env
	}
	if env := strings.TrimSpace(envConfig.GetString("API_KEY")); apiKey == "" && env != "" {
		apiKey = env
	}

	profile = strings.TrimSpace(profileFlag)

	var profileCfg config.Profile
	if cfg != nil {
		resolvedProfile, cfgProfile, cfgErr := cfg.ActiveProfile(profile)
		if cfgErr == nil {
			if baseURL == "" {
				baseURL = cfgProfile.BaseURL
			}
			profile = resolvedProfile
			profileCfg = cfgProfile
		} else if profile == "" {
			return "", "", "", cfgErr
		}
	}

	if baseURL == "" {
		return profile, baseURL, apiKey, errors.New("no SABnzbd base URL configured; run 'sabx login'")
	}

	if apiKey == "" {
		key, keyErr := auth.LoadAPIKey(profileOrDefault(profile), baseURL)
		if keyErr != nil {
			if profileCfg.APIKey != "" {
				apiKey = profileCfg.APIKey
			} else {
				return profile, baseURL, apiKey, fmt.Errorf("api key not found for profile %q (%v)", profileOrDefault(profile), keyErr)
			}
		} else {
			apiKey = key
		}
	}

	return profileOrDefault(profile), baseURL, apiKey, nil
}

func profileOrDefault(profile string) string {
	if strings.TrimSpace(profile) == "" {
		return "default"
	}
	return profile
}

func getApp(cmd *cobra.Command) (*cobraext.App, error) {
	app, ok := cobraext.From(cmd.Context())
	if !ok {
		return nil, errors.New("internal: app context missing")
	}
	return app, nil
}
