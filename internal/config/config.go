package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config models sabx persistent settings stored on disk. Profiles reference SABnzbd instances.
type Config struct {
	DefaultProfile string             `yaml:"default_profile"`
	Profiles       map[string]Profile `yaml:"profiles"`
	path           string             `yaml:"-"`
	mu             sync.RWMutex       `yaml:"-"`
}

// Profile stores base URL for a SABnzbd instance.
type Profile struct {
	BaseURL            string `yaml:"base_url"`
	APIKey             string `yaml:"api_key,omitempty"`
	AllowInsecureStore bool   `yaml:"allow_insecure_store,omitempty"`
}

// Load reads configuration from disk, returning an initialized Config.
func Load() (*Config, error) {
	dir, err := resolveConfigDir()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		DefaultProfile: "default",
		Profiles:       map[string]Profile{},
	}

	for _, name := range []string{"config.yml", "config.yaml"} {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}

		if len(data) == 0 {
			cfg.path = path
			return cfg, nil
		}

		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}

		cfg.path = path
		if cfg.Profiles == nil {
			cfg.Profiles = map[string]Profile{}
		}
		if cfg.DefaultProfile == "" {
			cfg.DefaultProfile = "default"
		}
		return cfg, nil
	}

	cfg.path = filepath.Join(dir, "config.yml")
	return cfg, nil
}

// Save persists the configuration to disk.
func (c *Config) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.path == "" {
		dir, err := resolveConfigDir()
		if err != nil {
			return err
		}
		c.path = filepath.Join(dir, "config.yml")
	}

	if c.Profiles == nil {
		c.Profiles = map[string]Profile{}
	}
	if c.DefaultProfile == "" {
		c.DefaultProfile = "default"
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	tmpFile, err := os.CreateTemp(dir, ".config-*.yml")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write temp config: %w", err)
	}

	// Sync to ensure data is written to disk before rename
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("sync temp config: %w", err)
	}

	if err := tmpFile.Chmod(0o600); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("chmod temp config: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp config: %w", err)
	}

	if err := os.Rename(tmpFile.Name(), c.path); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// Path returns the config file location.
func (c *Config) Path() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.path
}

// SetProfile updates or creates a profile entry.
func (c *Config) SetProfile(name string, profile Profile) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Profiles == nil {
		c.Profiles = map[string]Profile{}
	}
	c.Profiles[name] = profile
}

// GetProfile retrieves a profile, returning bool indicating existence.
func (c *Config) GetProfile(name string) (Profile, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	p, ok := c.Profiles[name]
	return p, ok
}

// ActiveProfile resolves the profile to use, considering overrides.
func (c *Config) ActiveProfile(override string) (string, Profile, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	name := c.DefaultProfile
	if override != "" {
		name = override
	}

	if name == "" {
		return "", Profile{}, errors.New("no profile configured")
	}

	profile, ok := c.Profiles[name]
	if !ok {
		return "", Profile{}, fmt.Errorf("profile %q not found", name)
	}

	return name, profile, nil
}

func resolveConfigDir() (string, error) {
	if base := strings.TrimSpace(os.Getenv("SABX_CONFIG_DIR")); base != "" {
		return base, nil
	}

	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "sabx"), nil
}
