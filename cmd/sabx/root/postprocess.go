package root

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func postprocessCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "postprocess",
		Short: jsonShort("Control SABnzbd post-processing"),
	}
	cmd.AddCommand(postprocessPauseCmd())
	cmd.AddCommand(postprocessResumeCmd())
	cmd.AddCommand(postprocessCancelCmd())
	return cmd
}

func postprocessPauseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pause",
		Short: jsonShort("Pause post-processing globally"),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()
			if err := app.Client.PausePostProcessing(ctx); err != nil {
				return err
			}
			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"paused": true})
			}
			return app.Printer.Print("Post-processing paused")
		},
	}
	return cmd
}

func postprocessResumeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resume",
		Short: jsonShort("Resume post-processing globally"),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()
			if err := app.Client.ResumePostProcessing(ctx); err != nil {
				return err
			}
			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"paused": false})
			}
			return app.Printer.Print("Post-processing resumed")
		},
	}
	return cmd
}

func postprocessCancelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel <nzo-id> [nzo-id...]",
		Short: jsonShort("Cancel post-processing for specific queue items"),
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("provide at least one nzo-id")
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
			if err := app.Client.CancelPostProcessing(ctx, args); err != nil {
				return err
			}
			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"cancelled": args})
			}
			return app.Printer.Print(fmt.Sprintf("Cancelled post-processing for %s", strings.Join(args, ", ")))
		},
	}
	return cmd
}
