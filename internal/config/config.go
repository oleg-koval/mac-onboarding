package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Source  SourceConfig            `yaml:"source"`
	Modules map[string]ModuleConfig `yaml:"modules"`
}

type SourceConfig struct {
	Host string `yaml:"host"` // Tailscale hostname of source Mac (bridge mode)
}

// ModuleConfig controls per-module behaviour.
// Unknown fields are captured in Extra and trigger a warning.
type ModuleConfig struct {
	Skip    bool              `yaml:"skip"`
	Options map[string]string `yaml:"options"`
}

var searchPaths = []string{
	"./onboard.yaml",
	"~/.config/mac-onboarding/onboard.yaml",
}

func Load(explicit string) (*Config, error) {
	path, err := resolve(explicit)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var cfg Config
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	dec.KnownFields(false) // warn but don't fatal on unknown fields
	if err := dec.Decode(&cfg); err != nil && err.Error() != "EOF" {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	if cfg.Modules == nil {
		cfg.Modules = make(map[string]ModuleConfig)
	}

	return &cfg, nil
}

// IsSkipped returns true if the module is explicitly skipped in config.
func (c *Config) IsSkipped(name string) bool {
	m, ok := c.Modules[name]
	return ok && m.Skip
}

func resolve(explicit string) (string, error) {
	if explicit != "" {
		return expand(explicit), nil
	}
	for _, p := range searchPaths {
		expanded := expand(p)
		if _, err := os.Stat(expanded); err == nil {
			return expanded, nil
		}
	}
	// No config found — return empty path; caller gets defaults.
	return "", fmt.Errorf("no onboard.yaml found; copy onboard.yaml.example and edit it")
}

func expand(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
