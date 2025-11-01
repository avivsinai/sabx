package root

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func speedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "speed",
		Short: "Inspect and control download speeds",
	}
	cmd.AddCommand(speedShowCmd())
	cmd.AddCommand(speedLimitCmd())
	return cmd
}

func speedShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Display current speed information",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			status, err := app.Client.Status(ctx)
			if err != nil {
				return err
			}
			queue, err := app.Client.Queue(ctx, 0, 0, "")
			if err != nil {
				return err
			}
			if app.Printer.JSON {
				payload := map[string]any{
					"speed_kbps":   status.Speed,
					"limit_kbps":   status.SpeedLimit,
					"paused":       status.Paused,
					"queue_speed":  queue.Speed,
					"queue_limit":  queue.SpeedLimit,
					"queue_paused": queue.Paused,
				}
				return app.Printer.Print(payload)
			}
			summary := fmt.Sprintf("Speed: %s KB/s (limit %s) paused=%v", status.Speed, status.SpeedLimit, status.Paused)
			return app.Printer.Print(summary)
		},
	}
	return cmd
}

func speedLimitCmd() *cobra.Command {
	var rate string
	var remove bool
	cmd := &cobra.Command{
		Use:   "limit",
		Short: "Set the global speed limit",
		RunE: func(cmd *cobra.Command, args []string) error {
			if remove {
				if cmd.Flags().Changed("rate") {
					return errors.New("use --none without --rate")
				}
			} else if !cmd.Flags().Changed("rate") {
				return errors.New("provide --rate or use --none")
			}
			app, err := getApp(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := timeoutContext(cmd.Context())
			defer cancel()

			if remove {
				if err := app.Client.SpeedLimit(ctx, nil); err != nil {
					return err
				}
				if app.Printer.JSON {
					return app.Printer.Print(map[string]any{"limit": nil})
				}
				return app.Printer.Print("Speed limit removed")
			}

			normalized, err := normalizeSpeedLimitInput(rate)
			if err != nil {
				return err
			}

			if err := app.Client.SpeedLimit(ctx, &normalized); err != nil {
				return err
			}
			if app.Printer.JSON {
				return app.Printer.Print(map[string]any{"value": normalized, "input": rate})
			}
			return app.Printer.Print(fmt.Sprintf("Speed limit set to %s", normalized))
		},
	}
	cmd.Flags().StringVar(&rate, "rate", "", "Limit rate (examples: 50%, 800K, 4M, 4MB/s, 10Mbps)")
	cmd.Flags().BoolVar(&remove, "none", false, "Remove the limit")
	return cmd
}

func normalizeSpeedLimitInput(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", errors.New("rate string must not be empty")
	}
	if strings.HasSuffix(value, "%") {
		number := strings.TrimSpace(strings.TrimSuffix(value, "%"))
		if number == "" {
			return "", errors.New("invalid percentage value")
		}
		percent, err := strconv.ParseFloat(number, 64)
		if err != nil {
			return "", fmt.Errorf("invalid percentage %q: %w", raw, err)
		}
		if percent < 0 {
			return "", errors.New("percentage must be non-negative")
		}
		return formatFloat(percent), nil
	}

	compact := strings.ReplaceAll(value, " ", "")
	numPart, unitPart := splitRate(compact)
	if numPart == "" || unitPart == "" {
		return "", fmt.Errorf("invalid rate %q: specify a numeric value and unit (e.g. 800K, 4MB/s, 10Mbps)", raw)
	}

	number, err := strconv.ParseFloat(numPart, 64)
	if err != nil {
		return "", fmt.Errorf("invalid rate %q: %w", raw, err)
	}
	if number <= 0 {
		return "", errors.New("rate must be positive")
	}

	bytesPerSecond, err := resolveBytesPerSecond(number, unitPart)
	if err != nil {
		return "", err
	}
	kiloPerSecond := bytesPerSecond / 1000
	return formatAbsoluteRate(kiloPerSecond), nil
}

func splitRate(input string) (string, string) {
	if input == "" {
		return "", ""
	}
	i := 0
	for i < len(input) {
		c := input[i]
		if (c >= '0' && c <= '9') || c == '.' {
			i++
			continue
		}
		break
	}
	return input[:i], input[i:]
}

func resolveBytesPerSecond(value float64, unit string) (float64, error) {
	lower := strings.ToLower(unit)
	clean := lower
	for _, suffix := range []string{"/sec", "/s", "ps"} {
		if strings.HasSuffix(clean, suffix) {
			clean = strings.TrimSuffix(clean, suffix)
			break
		}
	}
	isBits := strings.Contains(clean, "bit") || strings.HasSuffix(lower, "bps")
	if isBits && strings.Contains(unit, "B") {
		// Uppercase B signals bytes (e.g., MBps)
		isBits = false
	}

	base := clean
	switch {
	case strings.HasPrefix(base, "kib"):
		base = "kib"
	case strings.HasPrefix(base, "kb"):
		base = "kb"
	case strings.HasPrefix(base, "ki"):
		base = "kib"
	case strings.HasPrefix(base, "k"):
		base = "kb"
	case strings.HasPrefix(base, "mib"):
		base = "mib"
	case strings.HasPrefix(base, "mb"):
		base = "mb"
	case strings.HasPrefix(base, "mi"):
		base = "mib"
	case strings.HasPrefix(base, "m"):
		base = "mb"
	case strings.HasPrefix(base, "gib"):
		base = "gib"
	case strings.HasPrefix(base, "gb"):
		base = "gb"
	case strings.HasPrefix(base, "gi"):
		base = "gib"
	case strings.HasPrefix(base, "g"):
		base = "gb"
	case strings.HasPrefix(base, "b"):
		base = "b"
	default:
		return 0, fmt.Errorf("unsupported unit %q", unit)
	}

	if isBits {
		return bitsToBytesPerSecond(value, base), nil
	}
	return bytesPerSecond(value, base), nil
}

func bytesPerSecond(value float64, base string) float64 {
	switch base {
	case "kb":
		return value * 1000
	case "kib":
		return value * 1024
	case "mb":
		return value * 1000 * 1000
	case "mib":
		return value * 1024 * 1024
	case "gb":
		return value * 1000 * 1000 * 1000
	case "gib":
		return value * 1024 * 1024 * 1024
	case "b":
		return value
	default:
		return value
	}
}

func bitsToBytesPerSecond(value float64, base string) float64 {
	var multiplier float64
	switch base {
	case "kb":
		multiplier = 1000
	case "kib":
		multiplier = 1024
	case "mb":
		multiplier = 1000 * 1000
	case "mib":
		multiplier = 1024 * 1024
	case "gb":
		multiplier = 1000 * 1000 * 1000
	case "gib":
		multiplier = 1024 * 1024 * 1024
	case "b":
		multiplier = 1
	default:
		multiplier = 1
	}
	bitsPerSecond := value * multiplier
	return bitsPerSecond / 8
}

func formatAbsoluteRate(kiloPerSecond float64) string {
	if kiloPerSecond >= 1000 {
		return formatFloat(kiloPerSecond/1000) + "M"
	}
	return formatFloat(kiloPerSecond) + "K"
}

func formatFloat(v float64) string {
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return "0"
	}
	if math.Abs(v-math.Round(v)) < 1e-6 {
		return strconv.FormatInt(int64(math.Round(v)), 10)
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.3f", v), "0"), ".")
}
