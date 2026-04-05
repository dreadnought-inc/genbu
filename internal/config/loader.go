package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadFile reads and parses a genbu YAML config file.
func LoadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	return Parse(data)
}

// Parse parses raw YAML bytes into a Config.
func Parse(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validate(cfg *Config) error {
	if cfg.Version == "" {
		return fmt.Errorf("config version is required")
	}
	if cfg.Version != "1" {
		return fmt.Errorf("unsupported config version: %s", cfg.Version)
	}

	seen := make(map[string]bool)
	for _, v := range cfg.Flatten() {
		if v.Name == "" {
			return fmt.Errorf("variable name is required")
		}
		if seen[v.Name] {
			return fmt.Errorf("duplicate variable name: %s", v.Name)
		}
		seen[v.Name] = true
	}

	return nil
}
