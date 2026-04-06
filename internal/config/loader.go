package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadFile reads and parses a genbu YAML config file.
func LoadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path) //#nosec G304 -- path is user-provided CLI argument
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

	normalize(&cfg)

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// normalize migrates deprecated source types and field names to the current format.
func normalize(cfg *Config) {
	for i := range cfg.Variables {
		normalizeVariable(&cfg.Variables[i])
	}
	for i := range cfg.Groups {
		if cfg.Groups[i].Source != nil {
			normalizeSource(cfg.Groups[i].Source)
		}
		for j := range cfg.Groups[i].Variables {
			normalizeVariable(&cfg.Groups[i].Variables[j])
		}
	}
}

func normalizeVariable(v *Variable) {
	if v.Source != nil {
		normalizeSource(v.Source)
	}
}

func normalizeSource(src *SourceConfig) {
	switch src.Type {
	case "aws-ssm":
		src.Type = "parameter"
	case "aws-secretsmanager":
		src.Type = "secret"
	}

	if src.Key == "" {
		if src.Path != "" {
			src.Key = src.Path
		} else if src.SecretID != "" {
			src.Key = src.SecretID
		}
	}
}

var validProviders = map[string]bool{
	"":      true,
	"aws":   true,
	"gcp":   true,
	"azure": true,
}

func validate(cfg *Config) error {
	if cfg.Version == "" {
		return fmt.Errorf("config version is required")
	}
	if cfg.Version != "1" {
		return fmt.Errorf("unsupported config version: %s", cfg.Version)
	}

	if !validProviders[cfg.Provider] {
		return fmt.Errorf("unsupported provider: %s (supported: aws, gcp, azure)", cfg.Provider)
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
