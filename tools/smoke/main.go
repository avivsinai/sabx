package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type smokeCommand struct {
	Name        string
	Args        []string
	Description string
	ExpectJSON  bool
	Fixture     string
}

type commandReport struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Args        []string       `json:"args"`
	ExitCode    int            `json:"exit_code"`
	DurationMS  int64          `json:"duration_ms"`
	Stdout      string         `json:"stdout"`
	Stderr      string         `json:"stderr,omitempty"`
	ParsedJSON  map[string]any `json:"parsed_json,omitempty"`
	Err         string         `json:"error,omitempty"`
}

type runReport struct {
	RanAt      time.Time       `json:"ran_at"`
	BaseURL    string          `json:"base_url"`
	Commands   []commandReport `json:"commands"`
	Failures   int             `json:"failures"`
	OutputDir  string          `json:"output_dir,omitempty"`
	BinaryPath string          `json:"binary_path"`
}

func main() {
	var (
		baseURL   = flag.String("base-url", "", "SABnzbd base URL (falls back to SABX_BASE_URL)")
		apiKey    = flag.String("api-key", "", "SABnzbd API key (falls back to SABX_API_KEY)")
		binary    = flag.String("binary", "", "Path to sabx binary (defaults to temporary build)")
		outputDir = flag.String("output", "testdata/smoke/latest", "Directory for recorded fixtures")
		failFast  = flag.Bool("fail-fast", true, "Stop after the first failing command")
		record    = flag.Bool("record", true, "Persist stdout to fixture files")
		timeout   = flag.Duration("timeout", 30*time.Second, "Per-command timeout")
	)
	flag.Parse()

	if *baseURL == "" {
		*baseURL = strings.TrimSpace(os.Getenv("SABX_BASE_URL"))
	}
	if *apiKey == "" {
		*apiKey = strings.TrimSpace(os.Getenv("SABX_API_KEY"))
	}

	if *baseURL == "" || *apiKey == "" {
		fmt.Fprintln(os.Stderr, "error: base URL and API key are required (flags or SABX_BASE_URL/SABX_API_KEY)")
		os.Exit(2)
	}

	binPath, cleanup, err := resolveBinary(*binary)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if cleanup != nil {
		defer cleanup()
	}

	commands := []smokeCommand{
		{
			Name:        "notifications-email",
			Args:        []string{"notifications", "test", "email"},
			Description: "Exercise test_email notification endpoint",
			ExpectJSON:  true,
			Fixture:     "notifications-email.json",
		},
		{
			Name:        "notifications-pushover",
			Args:        []string{"notifications", "test", "pushover"},
			Description: "Exercise test_pushover notification endpoint",
			ExpectJSON:  true,
			Fixture:     "notifications-pushover.json",
		},
		{
			Name:        "browse-root",
			Args:        []string{"browse"},
			Description: "Browse SAB host root directory",
			ExpectJSON:  true,
			Fixture:     "browse-root.json",
		},
		{
			Name:        "browse-compact",
			Args:        []string{"browse", "--compact"},
			Description: "Browse SAB host root in compact mode",
			ExpectJSON:  true,
			Fixture:     "browse-compact.json",
		},
		{
			Name:        "debug-eval-sort",
			Args:        []string{"debug", "eval-sort", "title"},
			Description: "Ensure eval_sort endpoint responds",
			ExpectJSON:  true,
			Fixture:     "debug-eval-sort.json",
		},
		{
			Name:        "watched-scan",
			Args:        []string{"watched", "scan"},
			Description: "Trigger watched_now scan",
			ExpectJSON:  true,
			Fixture:     "watched-scan.json",
		},
		{
			Name:        "orphans-list",
			Args:        []string{"status", "orphans", "list"},
			Description: "List orphaned jobs (fullstatus folders)",
			ExpectJSON:  true,
			Fixture:     "orphans-list.json",
		},
		{
			Name:        "orphans-delete-all",
			Args:        []string{"status", "orphans", "delete-all"},
			Description: "Delete all orphaned jobs",
			ExpectJSON:  true,
			Fixture:     "orphans-delete-all.json",
		},
		{
			Name:        "orphans-add-all",
			Args:        []string{"status", "orphans", "add-all"},
			Description: "Re-add all orphans to queue",
			ExpectJSON:  true,
			Fixture:     "orphans-add-all.json",
		},
	}

	report := runReport{
		RanAt:      time.Now().UTC(),
		BaseURL:    *baseURL,
		BinaryPath: binPath,
	}

	if *record {
		if err := os.MkdirAll(*outputDir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "error: unable to create output directory: %v\n", err)
			os.Exit(1)
		}
		report.OutputDir = *outputDir
	}

	for _, cmd := range commands {
		res := runSmokeCommand(binPath, *baseURL, *apiKey, *timeout, cmd)
		report.Commands = append(report.Commands, res)
		if res.ExitCode != 0 {
			report.Failures++
			if *failFast {
				break
			}
		}

		if *record {
			if err := writeFixture(*outputDir, cmd, res); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to write fixture for %s: %v\n", cmd.Name, err)
			}
		}
	}

	if err := emitReport(*record, *outputDir, report); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to write report: %v\n", err)
	}

	for _, cmd := range report.Commands {
		fmt.Printf("[%s] exit=%d duration=%dms\n", cmd.Name, cmd.ExitCode, cmd.DurationMS)
		if cmd.Err != "" {
			fmt.Printf("  error: %s\n", cmd.Err)
		}
		if strings.TrimSpace(cmd.Stderr) != "" {
			fmt.Printf("  stderr: %s\n", strings.TrimSpace(cmd.Stderr))
		}
	}

	if report.Failures > 0 {
		fmt.Fprintf(os.Stderr, "%d smoke command(s) failed\n", report.Failures)
		os.Exit(1)
	}
}

func resolveBinary(userPath string) (string, func(), error) {
	if userPath != "" {
		if _, err := os.Stat(userPath); err != nil {
			return "", nil, err
		}
		return userPath, nil, nil
	}

	tmpDir, err := os.MkdirTemp("", "sabx-smoke-*")
	if err != nil {
		return "", nil, err
	}
	binPath := filepath.Join(tmpDir, "sabx-smoke")

	build := exec.Command("go", "build", "-o", binPath, "./cmd/sabx")
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	build.Env = os.Environ()
	if err := build.Run(); err != nil {
		os.RemoveAll(tmpDir)
		return "", nil, fmt.Errorf("failed to build sabx: %w", err)
	}

	cleanup := func() {
		_ = os.RemoveAll(tmpDir)
	}
	return binPath, cleanup, nil
}

func runSmokeCommand(binary, baseURL, apiKey string, timeout time.Duration, cmd smokeCommand) commandReport {
	args := append([]string{"--base-url", baseURL, "--api-key", apiKey, "--json"}, cmd.Args...)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	execCmd := exec.CommandContext(ctx, binary, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	start := time.Now()
	err := execCmd.Run()
	duration := time.Since(start)

	report := commandReport{
		Name:        cmd.Name,
		Description: cmd.Description,
		Args:        cmd.Args,
		DurationMS:  duration.Milliseconds(),
		Stdout:      stdout.String(),
		Stderr:      stderr.String(),
	}

	if ctx.Err() != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
		report.Err = "command timed out"
		report.ExitCode = -1
		return report
	}

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			report.ExitCode = exitErr.ExitCode()
		} else {
			report.ExitCode = -1
		}
		report.Err = err.Error()
	} else {
		report.ExitCode = 0
	}

	if cmd.ExpectJSON {
		var decoded map[string]any
		if parseErr := json.Unmarshal(stdout.Bytes(), &decoded); parseErr != nil {
			report.Err = strings.TrimSpace(report.Err + "\njson decode error: " + parseErr.Error())
			if report.ExitCode == 0 {
				report.ExitCode = -1
			}
		} else {
			report.ParsedJSON = decoded
		}
	}

	return report
}

func writeFixture(dir string, cmd smokeCommand, report commandReport) error {
	if report.ParsedJSON == nil {
		return os.WriteFile(filepath.Join(dir, cmd.Fixture), []byte(report.Stdout), 0o644)
	}

	normalized := redactDynamicFields(report.ParsedJSON)
	data, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, cmd.Fixture), append(data, '\n'), 0o644)
}

func emitReport(record bool, dir string, report runReport) error {
	if !record {
		return nil
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "report.json"), append(data, '\n'), 0o644)
}

func redactDynamicFields(payload map[string]any) map[string]any {
	clean := make(map[string]any, len(payload))
	for k, v := range payload {
		switch val := v.(type) {
		case string:
			if strings.HasPrefix(strings.ToLower(k), "apikey") || strings.Contains(strings.ToLower(k), "apikey") {
				clean[k] = "***redacted***"
				continue
			}
			clean[k] = val
		case map[string]any:
			clean[k] = redactDynamicFields(val)
		case []any:
			clean[k] = redactSlice(val)
		default:
			clean[k] = val
		}
	}
	return clean
}

func redactSlice(in []any) []any {
	out := make([]any, len(in))
	for i, v := range in {
		switch val := v.(type) {
		case map[string]any:
			out[i] = redactDynamicFields(val)
		case []any:
			out[i] = redactSlice(val)
		default:
			out[i] = val
		}
	}
	return out
}
