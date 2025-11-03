package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/99designs/keyring"
)

const serviceName = "sabx"

const (
	envAllowInsecure = "SABX_ALLOW_INSECURE_STORE"
	envPassphrase    = "SABX_KEYRING_PASSPHRASE"
	envBackend       = "SABX_KEYRING_BACKEND"
	envFileDir       = "SABX_KEYRING_FILE_DIR"
)

// ErrNotFound is returned when the requested credential cannot be located.
var ErrNotFound = os.ErrNotExist

// Store manages credential persistence backed by the OS keyring or an
// encrypted file fallback.
type Store struct {
	kr keyring.Keyring
}

type openOptions struct {
	allowFile       bool
	passphrase      string
	allowedBackends []keyring.BackendType
	fileDir         string
}

// Option tweaks how the secret store is opened.
type Option func(*openOptions)

// WithAllowFileFallback enables the encrypted file backend when native keyrings
// are unavailable. The fallback is encrypted but still less secure than the OS
// keychain and should be opt-in.
func WithAllowFileFallback(enable bool) Option {
	return func(o *openOptions) {
		o.allowFile = enable
	}
}

// WithPassphrase supplies a passphrase for the encrypted file backend, allowing
// unattended unlocks.
func WithPassphrase(pass string) Option {
	return func(o *openOptions) {
		if pass != "" {
			o.passphrase = pass
		}
	}
}

// WithFileDir overrides the encrypted file backend directory.
func WithFileDir(dir string) Option {
	return func(o *openOptions) {
		if dir != "" {
			o.fileDir = dir
		}
	}
}

// Open initialises a Store using the preferred OS keyring. Optional arguments
// can enable the encrypted file backend.
func Open(opts ...Option) (*Store, error) {
	cfg := keyring.Config{
		ServiceName: serviceName,
	}

	settings := openOptions{}

	if envEnabled(os.Getenv(envAllowInsecure)) {
		settings.allowFile = true
	}
	if pass := strings.TrimSpace(os.Getenv(envPassphrase)); pass != "" {
		settings.passphrase = pass
	}
	if dir := strings.TrimSpace(os.Getenv(envFileDir)); dir != "" {
		settings.fileDir = dir
	}

	for _, opt := range opts {
		opt(&settings)
	}

	cfg.AllowedBackends = resolveAllowedBackends(settings)

	if usesFileBackend(cfg.AllowedBackends) {
		if err := configureFileBackend(&cfg, settings); err != nil {
			return nil, err
		}
	}

	kr, err := keyring.Open(cfg)
	if err != nil {
		if errors.Is(err, keyring.ErrNoAvailImpl) && !usesFileBackend(cfg.AllowedBackends) {
			return nil, fmt.Errorf("open keyring: %w (set %s=1 or rerun with --allow-insecure-store to permit encrypted file fallback)", err, envAllowInsecure)
		}
		return nil, fmt.Errorf("open keyring: %w", err)
	}

	return &Store{kr: kr}, nil
}

// Save persists a SABnzbd API key.
func (s *Store) Save(profile, baseURL, apiKey string) error {
	if s == nil || s.kr == nil {
		return errors.New("secret store not initialized")
	}

	return s.kr.Set(keyring.Item{
		Key:   keyFor(profile, baseURL),
		Data:  []byte(apiKey),
		Label: fmt.Sprintf("sabx profile %s API key", sanitize(profile)),
	})
}

// Load retrieves a SABnzbd API key.
func (s *Store) Load(profile, baseURL string) (string, error) {
	if s == nil || s.kr == nil {
		return "", errors.New("secret store not initialized")
	}

	item, err := s.kr.Get(keyFor(profile, baseURL))
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return "", ErrNotFound
		}
		return "", err
	}
	return string(item.Data), nil
}

// Delete removes a stored credential.
func (s *Store) Delete(profile, baseURL string) error {
	if s == nil || s.kr == nil {
		return errors.New("secret store not initialized")
	}

	err := s.kr.Remove(keyFor(profile, baseURL))
	if errors.Is(err, keyring.ErrKeyNotFound) {
		return nil
	}
	return err
}

// IsNoKeyringError reports whether the provided error indicates that no native
// keyring backend is available on the host.
func IsNoKeyringError(err error) bool {
	return errors.Is(err, keyring.ErrNoAvailImpl)
}

// SaveAPIKey is a convenience helper that opens a Store, writes the key, and
// closes it immediately.
func SaveAPIKey(profile, baseURL, apiKey string, opts ...Option) error {
	store, err := Open(opts...)
	if err != nil {
		return err
	}
	return store.Save(profile, baseURL, apiKey)
}

// LoadAPIKey loads a stored key using a one-off Store.
func LoadAPIKey(profile, baseURL string, opts ...Option) (string, error) {
	store, err := Open(opts...)
	if err != nil {
		return "", err
	}
	return store.Load(profile, baseURL)
}

// DeleteAPIKey removes a stored key using a one-off Store.
func DeleteAPIKey(profile, baseURL string, opts ...Option) error {
	store, err := Open(opts...)
	if err != nil {
		return err
	}
	return store.Delete(profile, baseURL)
}

func keyFor(profile, baseURL string) string {
	hash := sha256.Sum256([]byte(normalizeBaseURL(baseURL)))
	return fmt.Sprintf("profile/%s/%s", sanitize(profile), hex.EncodeToString(hash[:16]))
}

func sanitize(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "default"
	}
	return strings.ToLower(strings.ReplaceAll(value, " ", "-"))
}

func normalizeBaseURL(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	normalized = strings.TrimRight(normalized, "/")
	return normalized
}

func resolveAllowedBackends(opts openOptions) []keyring.BackendType {
	if len(opts.allowedBackends) > 0 {
		return opts.allowedBackends
	}

	if backendEnv := strings.TrimSpace(firstNonEmpty(os.Getenv(envBackend), os.Getenv("KEYRING_BACKEND"))); backendEnv != "" {
		return parseBackendList(backendEnv, opts.allowFile)
	}

	backends := defaultBackends()
	if opts.allowFile {
		backends = append(backends, keyring.FileBackend)
	}
	return backends
}

func defaultBackends() []keyring.BackendType {
	switch runtime.GOOS {
	case "darwin":
		return []keyring.BackendType{keyring.KeychainBackend}
	case "windows":
		return []keyring.BackendType{keyring.WinCredBackend}
	default:
		return []keyring.BackendType{
			keyring.SecretServiceBackend,
			keyring.KWalletBackend,
			keyring.KeyCtlBackend,
			keyring.PassBackend,
		}
	}
}

func parseBackendList(raw string, allowFile bool) []keyring.BackendType {
	parts := strings.Split(raw, ",")
	var backends []keyring.BackendType
	for _, part := range parts {
		switch strings.TrimSpace(strings.ToLower(part)) {
		case "keychain":
			backends = append(backends, keyring.KeychainBackend)
		case "wincred":
			backends = append(backends, keyring.WinCredBackend)
		case "secret-service", "secretservice":
			backends = append(backends, keyring.SecretServiceBackend)
		case "kwallet":
			backends = append(backends, keyring.KWalletBackend)
		case "keyctl":
			backends = append(backends, keyring.KeyCtlBackend)
		case "pass":
			backends = append(backends, keyring.PassBackend)
		case "file":
			backends = append(backends, keyring.FileBackend)
		}
	}
	if !allowFile {
		filtered := backends[:0]
		for _, backend := range backends {
			if backend == keyring.FileBackend {
				continue
			}
			filtered = append(filtered, backend)
		}
		backends = filtered
	}
	return backends
}

func configureFileBackend(cfg *keyring.Config, opts openOptions) error {
	passphrase := opts.passphrase
	if passphrase == "" {
		if pwd := os.Getenv("KEYRING_FILE_PASSWORD"); pwd != "" {
			passphrase = pwd
		} else if pwd := os.Getenv("KEYRING_PASSWORD"); pwd != "" {
			passphrase = pwd
		}
	}

	if passphrase != "" {
		cfg.FilePasswordFunc = keyring.FixedStringPrompt(passphrase)
	} else {
		cfg.FilePasswordFunc = keyring.TerminalPrompt
	}

	dir := opts.fileDir
	if dir == "" {
		if userDir, err := os.UserConfigDir(); err == nil {
			dir = filepath.Join(userDir, serviceName, "secrets")
		}
	}

	if dir != "" {
		cfg.FileDir = dir
	}
	return nil
}

func usesFileBackend(backends []keyring.BackendType) bool {
	for _, backend := range backends {
		if backend == keyring.FileBackend {
			return true
		}
	}
	return false
}

// AllowInsecureStoreFromEnv reports whether SABX_ALLOW_INSECURE_STORE enables
// the encrypted file fallback.
func AllowInsecureStoreFromEnv() bool {
	return envEnabled(os.Getenv(envAllowInsecure))
}

func envEnabled(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
