package root

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

func notificationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "notifications",
		Short: jsonShort("Test SABnzbd notification integrations"),
		Long:  appendJSONLong("Run SABnzbd's built-in notification testers (email, pushover, etc.) to verify configuration."),
	}
	cmd.AddCommand(notificationsTestCmd())
	return cmd
}

func notificationsTestCmd() *cobra.Command {
	var params []string

	cmd := &cobra.Command{
		Use:   "test <type>",
		Short: jsonShort("Run a notification test (email, pushover, apprise, etc.)"),
		Long:  appendJSONLong("Executes SABnzbd's notification test endpoints. The command exits non-zero if SABnzbd reports a failure."),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			typeKey := strings.ToLower(args[0])
			mode, ok := notificationMode(typeKey)
			if !ok {
				return fmt.Errorf("unsupported notification type %q", typeKey)
			}

			vals := url.Values{}
			for _, entry := range params {
				parts := strings.SplitN(entry, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid --param %q", entry)
				}
				vals.Set(parts[0], parts[1])
			}

			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			result, err := app.Client.TestNotification(ctx, mode, vals)
			if err != nil {
				return err
			}

			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{
					"type":    typeKey,
					"success": result.Success,
					"message": result.Message,
				})
			}

			if result.Success {
				return app.Printer.Print(fmt.Sprintf("Notification %s test succeeded", typeKey))
			}
			if strings.TrimSpace(result.Message) == "" {
				return errors.New("notification test failed")
			}
			return errors.New(result.Message)
		},
	}

	cmd.Flags().StringArrayVar(&params, "param", nil, "Additional key=value parameters to pass to the notification test")
	return cmd
}

func notificationMode(kind string) (string, bool) {
	switch kind {
	case "email":
		return "test_email", true
	case "windows":
		return "test_windows", true
	case "desktop", "notif", "notification":
		return "test_notif", true
	case "osd":
		return "test_osd", true
	case "pushover":
		return "test_pushover", true
	case "pushbullet":
		return "test_pushbullet", true
	case "apprise":
		return "test_apprise", true
	case "prowl":
		return "test_prowl", true
	case "script", "nscript":
		return "test_nscript", true
	default:
		return "", false
	}
}
