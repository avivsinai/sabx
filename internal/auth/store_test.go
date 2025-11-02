package auth

import (
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/99designs/keyring"
)

func TestKeyForDeterministic(t *testing.T) {
	t.Helper()

	want := keyFor("default", "https://example.com/api")
	if want == "" {
		t.Fatalf("keyFor returned empty string")
	}

	if got := keyFor(" default ", "https://example.com/api/"); got != want {
		t.Fatalf("expected same key for normalized inputs, got %q, want %q", got, want)
	}

	other := keyFor("default", "https://example.net")
	if other == want {
		t.Fatalf("expected different keys for different hosts")
	}
}

func TestParseBackendList(t *testing.T) {
	t.Helper()

	tests := []struct {
		name      string
		raw       string
		allowFile bool
		want      []keyring.BackendType
	}{
		{
			name:      "file allowed explicitly",
			raw:       "file",
			allowFile: true,
			want:      []keyring.BackendType{keyring.FileBackend},
		},
		{
			name:      "file stripped when disallowed",
			raw:       "file,keychain",
			allowFile: false,
			want:      []keyring.BackendType{keyring.KeychainBackend},
		},
		{
			name:      "multiple backends",
			raw:       "secret-service,pass",
			allowFile: false,
			want: []keyring.BackendType{
				keyring.SecretServiceBackend,
				keyring.PassBackend,
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := parseBackendList(tc.raw, tc.allowFile); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("parseBackendList(%q, %v) = %#v, want %#v", tc.raw, tc.allowFile, got, tc.want)
			}
		})
	}
}

func TestResolveAllowedBackendsEnvOverride(t *testing.T) {
	t.Helper()

	t.Setenv(envBackend, "file")
	opts := openOptions{allowFile: true}

	if got := resolveAllowedBackends(opts); !reflect.DeepEqual(got, []keyring.BackendType{keyring.FileBackend}) {
		t.Fatalf("resolveAllowedBackends returned %#v, want file backend only", got)
	}
}

func TestEnvEnabled(t *testing.T) {
	t.Helper()

	cases := map[string]bool{
		"1":     true,
		"true":  true,
		"TRUE":  true,
		"yes":   true,
		"on":    true,
		"0":     false,
		"false": false,
		"off":   false,
		"":      false,
	}

	for input, want := range cases {
		if got := envEnabled(input); got != want {
			t.Fatalf("envEnabled(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestConfigureFileBackendUsesEnvWhenUnset(t *testing.T) {
	t.Helper()

	t.Setenv(envPassphrase, "")
	t.Setenv("KEYRING_FILE_PASSWORD", "env-pass")
	t.Setenv("KEYRING_PASSWORD", "")

	cfg := keyring.Config{}
	opts := openOptions{}

	if err := configureFileBackend(&cfg, opts); err != nil {
		t.Fatalf("configureFileBackend returned error: %v", err)
	}

	if cfg.FilePasswordFunc == nil {
		t.Fatalf("FilePasswordFunc should be set")
	}

	got, err := cfg.FilePasswordFunc("prompt")
	if err != nil {
		t.Fatalf("FilePasswordFunc returned error: %v", err)
	}
	if got != "env-pass" {
		t.Fatalf("FilePasswordFunc returned %q, want env-pass", got)
	}
}

func TestConfigureFileBackendUsesCustomDir(t *testing.T) {
	t.Helper()

	tmp := t.TempDir()
	cfg := keyring.Config{}
	opts := openOptions{
		fileDir: tmp,
	}

	if err := configureFileBackend(&cfg, opts); err != nil {
		t.Fatalf("configureFileBackend returned error: %v", err)
	}
	if cfg.FileDir != tmp {
		t.Fatalf("expected FileDir %q, got %q", tmp, cfg.FileDir)
	}
}

func TestOpenHonorsAllowFileEnv(t *testing.T) {
	t.Helper()

	t.Setenv(envAllowInsecure, "1")
	t.Setenv(envBackend, "file")

	opts := openOptions{allowFile: true}
	got := resolveAllowedBackends(opts)

	if len(got) == 0 || got[0] != keyring.FileBackend {
		t.Fatalf("expected file backend when %s=1, got %#v", envAllowInsecure, got)
	}
}

func TestAllowInsecureStoreFromEnv(t *testing.T) {
	t.Helper()

	t.Setenv(envAllowInsecure, "true")
	if !AllowInsecureStoreFromEnv() {
		t.Fatalf("expected AllowInsecureStoreFromEnv to return true when env is set")
	}

	t.Setenv(envAllowInsecure, "0")
	if AllowInsecureStoreFromEnv() {
		t.Fatalf("expected AllowInsecureStoreFromEnv to return false after disable")
	}
}

// Ensure ErrNotFound mirrors os.ErrNotExist for errors.Is compatibility.
func TestErrNotFoundCompatibility(t *testing.T) {
	t.Helper()

	if !errors.Is(ErrNotFound, os.ErrNotExist) {
		t.Fatalf("ErrNotFound should match os.ErrNotExist")
	}
}
