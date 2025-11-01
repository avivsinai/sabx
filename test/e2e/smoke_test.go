package e2e

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	containertypes "github.com/docker/docker/api/types/container"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	defaultAPIKey     = "sabx-e2e-apikey"
	defaultConfigTmpl = `__version__ = 19
__encoding__ = utf-8
[misc]
api_key = %s
enable_https = 0
host = 0.0.0.0
port = 8080
username =
password =
wait_for_unpack = 0
`
)

func init() {
	ensureDockerHost()
	if os.Getenv("TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE") != "" {
		return
	}

	if dockerHost := os.Getenv("DOCKER_HOST"); strings.HasPrefix(dockerHost, "unix://") {
		socket := strings.TrimPrefix(dockerHost, "unix://")
		if socket != "" && !strings.HasPrefix(socket, "/var/run/") {
			_ = os.Setenv("TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE", "/var/run/docker.sock")
		}
	}
}

func ensureDockerHost() {
	if os.Getenv("DOCKER_HOST") != "" {
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	candidates := []string{
		filepath.Join(home, ".colima", "default", "docker.sock"),
		filepath.Join(home, ".colima", "docker.sock"),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			_ = os.Setenv("DOCKER_HOST", "unix://"+candidate)
			return
		}
	}
}

// TestSmokeAgainstSABContainer builds the CLI, launches SABnzbd in a container, and
// runs the sabx smoke harness end-to-end. Requires Docker.
func TestSmokeAgainstSABContainer(t *testing.T) {
	if testing.Short() {
		t.Skip("sabx e2e smoke skipped in short mode")
	}
	if os.Getenv("SABX_E2E_DISABLE") == "1" {
		t.Skip("sabx e2e smoke disabled via SABX_E2E_DISABLE")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker binary not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	h, err := newHarness(ctx, t)
	if err != nil {
		t.Fatalf("failed to create sab e2e harness: %v", err)
	}
	defer func() {
		if err := h.Close(context.Background()); err != nil {
			t.Fatalf("failed to tear down harness: %v", err)
		}
	}()

	t.Logf("running smoke tests against %s", h.baseURL)

	if err := runSmoke(ctx, h.repoRoot, h.baseURL, h.apiKey); err != nil {
		t.Fatalf("smoke harness failed: %v", err)
	}
}

type harness struct {
	container tc.Container
	baseURL   string
	apiKey    string
	configDir string
	repoRoot  string
}

func newHarness(ctx context.Context, t *testing.T) (*harness, error) {
	repoRoot, err := repoRoot()
	if err != nil {
		return nil, err
	}

	configDir, apiKey, err := prepareConfigDir()
	if err != nil {
		return nil, err
	}

	req := tc.ContainerRequest{
		Image:        "ghcr.io/sabnzbd/sabnzbd:latest",
		ExposedPorts: []string{"8080/tcp"},
		Env: map[string]string{
			"SAB_CONFIG_DIR": "/config",
		},
		HostConfigModifier: func(hc *containertypes.HostConfig) {
			hc.Binds = append(hc.Binds, fmt.Sprintf("%s:/config", configDir))
		},
		WaitingFor: wait.ForListeningPort("8080/tcp").WithStartupTimeout(4 * time.Minute),
	}

	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("start sabnzbd container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx) // best effort
		return nil, fmt.Errorf("resolve container host: %w", err)
	}
	mappedPort, err := container.MappedPort(ctx, "8080/tcp")
	if err != nil {
		_ = container.Terminate(ctx) // best effort
		return nil, fmt.Errorf("resolve mapped port: %w", err)
	}

	baseURL := fmt.Sprintf("http://%s:%s", host, mappedPort.Port())

	h := &harness{
		container: container,
		baseURL:   baseURL,
		apiKey:    apiKey,
		configDir: configDir,
		repoRoot:  repoRoot,
	}

	if err := h.refreshAPIKey(ctx); err != nil {
		h.Close(ctx)
		return nil, err
	}
	if err := h.ensureReady(ctx); err != nil {
		h.Close(ctx)
		return nil, err
	}

	return h, nil
}

func (h *harness) ensureReady(ctx context.Context) error {
	client := &http.Client{
		Timeout: 3 * time.Second,
	}
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api", h.baseURL), nil)
		if err != nil {
			return err
		}
		q := req.URL.Query()
		q.Set("mode", "version")
		q.Set("output", "json")
		q.Set("apikey", h.apiKey)
		req.URL.RawQuery = q.Encode()

		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(2 * time.Second)
	}
	return errors.New("sabnzbd did not respond to version API before timeout")
}

func (h *harness) refreshAPIKey(ctx context.Context) error {
	timeout := time.After(2 * time.Minute)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return errors.New("api_key not discovered before timeout")
		case <-ticker.C:
			exitCode, reader, err := h.container.Exec(ctx, []string{"cat", "/config/sabnzbd.ini"})
			if err != nil || exitCode != 0 {
				continue
			}
			content, err := io.ReadAll(reader)
			if err != nil {
				continue
			}
			scanner := bufio.NewScanner(strings.NewReader(string(content)))
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if strings.HasPrefix(line, "api_key") {
					parts := strings.SplitN(line, "=", 2)
					if len(parts) == 2 {
						h.apiKey = strings.TrimSpace(parts[1])
						if h.apiKey == "" {
							break
						}
						return nil
					}
				}
			}
			if scanErr := scanner.Err(); scanErr != nil {
				return fmt.Errorf("parse sabnzbd.ini: %w", scanErr)
			}
		}
	}
}

func (h *harness) Close(ctx context.Context) error {
	var closeErr error
	if h.container != nil {
		closeErr = h.container.Terminate(ctx)
	}
	if h.configDir != "" {
		_ = os.RemoveAll(h.configDir)
	}
	return closeErr
}

func prepareConfigDir() (string, string, error) {
	dir, err := os.MkdirTemp("", "sabx-e2e-config-*")
	if err != nil {
		return "", "", fmt.Errorf("create temp config dir: %w", err)
	}
	if err := os.Chmod(dir, 0o777); err != nil {
		return "", "", fmt.Errorf("chmod config dir: %w", err)
	}
	apiKey := defaultAPIKey
	content := fmt.Sprintf(defaultConfigTmpl, apiKey)
	if err := os.WriteFile(filepath.Join(dir, "sabnzbd.ini"), []byte(content), 0o666); err != nil {
		return "", "", fmt.Errorf("write sabnzbd.ini: %w", err)
	}
	return dir, apiKey, nil
}

func repoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("determine repo root: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func runSmoke(ctx context.Context, repoRoot, baseURL, apiKey string) error {
	tmpDir, err := os.MkdirTemp("", "sabx-smoke-output-*")
	if err != nil {
		return fmt.Errorf("create smoke output dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	args := []string{
		"run", "./tools/smoke",
		"--base-url", baseURL,
		"--api-key", apiKey,
		"--output", tmpDir,
		"--record=false",
	}
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("SABX_BASE_URL=%s", baseURL),
		fmt.Sprintf("SABX_API_KEY=%s", apiKey),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go run ./tools/smoke failed: %w\n%s", err, string(out))
	}
	return nil
}
