package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key,omitempty"`
}

// Load reads configuration from disk, returning an initialized Config.
func Load() (*Config, error) {
	path, err := path()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		DefaultProfile: "default",
		Profiles:       map[string]Profile{},
		path:           path,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			if err := cfg.Save(); err != nil {
				return nil, err
			}
			return cfg, nil
		}
		return nil, err
	}

	if len(data) == 0 {
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

// Save persists the configuration to disk.
func (c *Config) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Profiles == nil {
		c.Profiles = map[string]Profile{}
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return err
	}

	return os.WriteFile(c.path, data, 0o600)
}

// Path returns the config file location.
func (c *Config) Path() string {
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

func path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "sabx", "config.yaml"), nil
}
