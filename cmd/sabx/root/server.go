package root

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/avivsinai/sabx/internal/sabapi"
)

func serverCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: jsonShort("Manage SABnzbd server connectivity"),
	}

	cmd.AddCommand(serverListCmd())
	cmd.AddCommand(serverStatsCmd())
	cmd.AddCommand(serverTestCmd())
	cmd.AddCommand(serverDisconnectCmd())
	cmd.AddCommand(serverUnblockCmd())
	cmd.AddCommand(serverRestartCmd())
	cmd.AddCommand(serverShutdownCmd())
	cmd.AddCommand(serverRepairCmd())

	return cmd
}

func serverListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: jsonShort("List configured news servers"),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			servers, err := app.Client.ServerConfigs(ctx)
			if err != nil {
				return err
			}

			sort.Slice(servers, func(i, j int) bool {
				return servers[i].DisplayName < servers[j].DisplayName
			})

			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"servers": servers})
			}

			if len(servers) == 0 {
				return app.Printer.Print("No servers configured")
			}

			rows := make([][]string, 0, len(servers))
			for _, srv := range servers {
				rows = append(rows, []string{
					srv.DisplayName,
					srv.Host,
					strconv.Itoa(srv.Port),
					boolToStr(srv.SSL),
					strconv.Itoa(srv.Connections),
					boolToStr(srv.Enable),
					strconv.Itoa(srv.Priority),
				})
			}
			headers := []string{"Name", "Host", "Port", "SSL", "Connections", "Enabled", "Priority"}
			return app.Printer.Table(headers, rows)
		},
	}
	return cmd
}

func serverStatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: jsonShort("Show aggregate server throughput statistics"),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			stats, err := app.Client.ServerStats(ctx)
			if err != nil {
				return err
			}

			if app.Printer.JSON {
				return app.Printer.Print(stats)
			}

			summary := [][]string{
				{"Total", humanBytes(stats.Total)},
				{"This Month", humanBytes(stats.Month)},
				{"This Week", humanBytes(stats.Week)},
				{"Today", humanBytes(stats.Day)},
			}
			if err := app.Printer.Table([]string{"Period", "Usage"}, summary); err != nil {
				return err
			}

			if len(stats.Servers) == 0 {
				return nil
			}

			configs, _ := app.Client.ServerConfigs(ctx) // best effort for friendly names
			nameMap := map[string]string{}
			for _, cfg := range configs {
				nameMap[cfg.Name] = cfg.DisplayName
			}

			headers := []string{"Server", "Total", "Month", "Week", "Day", "Articles Tried", "Articles Success"}
			rows := make([][]string, 0, len(stats.Servers))
			for key, value := range stats.Servers {
				label := nameMap[key]
				if label == "" {
					label = key
				}
				rows = append(rows, []string{
					label,
					humanBytes(value.Total),
					humanBytes(value.Month),
					humanBytes(value.Week),
					humanBytes(value.Day),
					formatFloat(value.ArticlesTried),
					formatFloat(value.ArticlesSuccess),
				})
			}
			sort.Slice(rows, func(i, j int) bool {
				return rows[i][0] < rows[j][0]
			})

			return app.Printer.Table(headers, rows)
		},
	}
	return cmd
}

func serverTestCmd() *cobra.Command {
	var host string
	var port int
	var username string
	var password string
	var connections int
	var timeout int
	var sslFlag bool
	var sslVerify int
	var sslCiphers string

	cmd := &cobra.Command{
		Use:   "test <server-name>",
		Short: jsonShort("Run SABnzbd's built-in server connectivity test"),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := strings.TrimSpace(args[0])
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			configs, err := app.Client.ServerConfigs(ctx)
			if err != nil {
				return err
			}
			server, ok := findServerConfig(configs, target)
			if !ok {
				return fmt.Errorf("server %q not found", target)
			}

			params := sabapi.ServerTestParams{
				Server:      server.Name,
				Host:        server.Host,
				Port:        server.Port,
				Username:    server.Username,
				Password:    server.Password,
				Connections: server.Connections,
				Timeout:     server.Timeout,
				SSL:         server.SSL,
				SSLVerify:   server.SSLVerify,
				SSLCiphers:  server.SSLCiphers,
			}

			if cmd.Flags().Changed("host") {
				params.Host = host
			}
			if cmd.Flags().Changed("port") {
				params.Port = port
			}
			if cmd.Flags().Changed("username") {
				params.Username = username
			}
			if cmd.Flags().Changed("password") {
				params.Password = password
			}
			if cmd.Flags().Changed("connections") {
				params.Connections = connections
			}
			if cmd.Flags().Changed("timeout") {
				params.Timeout = timeout
			}
			if cmd.Flags().Changed("ssl") {
				params.SSL = sslFlag
			}
			if cmd.Flags().Changed("ssl-verify") {
				params.SSLVerify = sslVerify
			}
			if cmd.Flags().Changed("ssl-ciphers") {
				params.SSLCiphers = sslCiphers
			}

			result, err := app.Client.TestServer(ctx, params)
			if err != nil {
				return err
			}

			if app.Printer.JSON {
				return app.Printer.Print(result)
			}

			status := "FAILED"
			if result.Result {
				status = "OK"
			}
			return app.Printer.Print(fmt.Sprintf("[%s] %s", status, result.Message))
		},
	}

	cmd.Flags().StringVar(&host, "host", "", "Override server host")
	cmd.Flags().IntVar(&port, "port", 0, "Override server port")
	cmd.Flags().StringVar(&username, "username", "", "Override username")
	cmd.Flags().StringVar(&password, "password", "", "Override password")
	cmd.Flags().IntVar(&connections, "connections", 0, "Override connection count")
	cmd.Flags().IntVar(&timeout, "timeout", 0, "Override timeout (seconds)")
	cmd.Flags().BoolVar(&sslFlag, "ssl", false, "Override SSL usage for test")
	cmd.Flags().IntVar(&sslVerify, "ssl-verify", -1, "Override SSL verification mode (0-3)")
	cmd.Flags().StringVar(&sslCiphers, "ssl-ciphers", "", "Override custom SSL ciphers")

	return cmd
}

func serverDisconnectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disconnect",
		Short: jsonShort("Force SABnzbd to disconnect from news servers"),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()
			return app.Client.Disconnect(ctx)
		},
	}
	return cmd
}

func serverUnblockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unblock <server-name>",
		Short: jsonShort("Unblock a temporarily disabled news server"),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			if name == "" {
				return errors.New("server name required")
			}
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()
			return app.Client.UnblockServer(ctx, name)
		},
	}
	return cmd
}

func serverRestartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restart",
		Short: jsonShort("Restart SABnzbd"),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()
			return app.Client.ServerControl(ctx, "restart")
		},
	}
	return cmd
}

func serverShutdownCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shutdown",
		Short: jsonShort("Shutdown SABnzbd"),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()
			return app.Client.ServerControl(ctx, "shutdown")
		},
	}
	return cmd
}

func serverRepairCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repair",
		Short: jsonShort("Trigger queue repair and restart SABnzbd"),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()
			return app.Client.RestartRepair(ctx)
		},
	}
	return cmd
}

func findServerConfig(configs []sabapi.ServerConfig, name string) (sabapi.ServerConfig, bool) {
	for _, cfg := range configs {
		if strings.EqualFold(cfg.Name, name) || strings.EqualFold(cfg.DisplayName, name) {
			return cfg, true
		}
	}
	return sabapi.ServerConfig{}, false
}

func humanBytes(value float64) string {
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	i := 0
	for value >= 1024 && i < len(units)-1 {
		value /= 1024
		i++
	}
	return fmt.Sprintf("%.2f %s", value, units[i])
}
