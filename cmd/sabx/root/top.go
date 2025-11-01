package root

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/sabx/sabx/internal/ui/top"
)

func topCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "top",
		Short: jsonShort("Interactive dashboard for SABnzbd queues"),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			if app.Client == nil {
				return fmt.Errorf("not logged in; run 'sabx login'")
			}
			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()
			return top.Run(ctx, app.Client)
		},
	}
	return cmd
}
