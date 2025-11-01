package root

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func scheduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedule",
		Short: jsonShort("Manage SABnzbd scheduler"),
	}
	cmd.AddCommand(scheduleListCmd())
	cmd.AddCommand(scheduleAddCmd())
	cmd.AddCommand(scheduleSetCmd())
	cmd.AddCommand(scheduleDeleteCmd())
	return cmd
}

func scheduleListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: jsonShort("List scheduled tasks"),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()
			payload, err := app.Client.SchedulerList(ctx)
			if err != nil {
				return err
			}
			tasks := parseNamedConfig(payload)
			if app.Printer.JSON {
				return app.Printer.Print(tasks)
			}
			headers := []string{"Name", "Command", "When", "Parameters"}
			rows := make([][]string, 0, len(tasks))
			for _, task := range tasks {
				command := task.Values["command"]
				when := fmt.Sprintf("%s %s:%s", task.Values["day"], task.Values["hour"], task.Values["min"])
				rows = append(rows, []string{task.Name, command, when, task.Values["value"]})
			}
			if err := app.Printer.Table(headers, rows); err != nil {
				return err
			}
			return app.Printer.Print(fmt.Sprintf("%d tasks", len(tasks)))
		},
	}
	return cmd
}

func scheduleAddCmd() *cobra.Command {
	var entries []string
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: jsonShort("Add a scheduled task"),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(entries) == 0 {
				return errors.New("provide at least one --set key=value pair describing the task")
			}
			props := pairsToMap(entries)
			name := args[0]
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()
			if err := applyNamedProperties(ctx, app, "scheduler", name, props); err != nil {
				return err
			}
			return app.Printer.Print("Task added")
		},
	}
	cmd.Flags().StringArrayVar(&entries, "set", nil, "Key=value pairs (command, day, hour, min, value, etc.)")
	return cmd
}

func scheduleSetCmd() *cobra.Command {
	var entries []string
	cmd := &cobra.Command{
		Use:   "set <name>",
		Short: jsonShort("Update a scheduled task"),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(entries) == 0 {
				return errors.New("provide at least one --set key=value pair")
			}
			props := pairsToMap(entries)
			name := args[0]
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()
			if err := applyNamedProperties(ctx, app, "scheduler", name, props); err != nil {
				return err
			}
			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"task": name, "updated": props})
			}
			return app.Printer.Print("Task updated")
		},
	}
	cmd.Flags().StringArrayVar(&entries, "set", nil, "Key=value pairs to update")
	return cmd
}

func scheduleDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: jsonShort("Delete a scheduled task"),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()
			if err := app.Client.ConfigDelete(ctx, "scheduler", args[0]); err != nil {
				return err
			}
			return app.Printer.Print("Task deleted")
		},
	}
	return cmd
}

func pairsToMap(entries []string) map[string]string {
	props := make(map[string]string)
	for _, entry := range entries {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}
		props[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return props
}
