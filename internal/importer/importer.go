package importer

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/dreadnought-inc/genbu/internal/config"
)

// EnvVar represents a discovered environment variable.
type EnvVar struct {
	Name  string
	Value string
}

// Importer parses a config file format into a list of env vars.
type Importer interface {
	Import(r io.Reader) ([]EnvVar, error)
}

// FormatFromExtension returns the importer format name from a file extension.
func FormatFromExtension(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".toml":
		return "toml"
	case ".ini", ".cfg", ".conf":
		return "ini"
	case ".env":
		return "dotenv"
	default:
		return ""
	}
}

// Get returns an Importer for the given format name.
func Get(format string) (Importer, error) {
	switch strings.ToLower(format) {
	case "dotenv", "env":
		return &DotenvImporter{}, nil
	case "ini":
		return &INIImporter{}, nil
	case "toml":
		return &TOMLImporter{}, nil
	default:
		return nil, fmt.Errorf("unsupported format: %q (supported: dotenv, ini, toml)", format)
	}
}

// GenerateYAML generates a genbu.yaml template from imported env vars.
func GenerateYAML(vars []EnvVar) ([]byte, error) {
	cfg := config.Config{
		Version: "1",
	}

	for _, v := range vars {
		cfg.Variables = append(cfg.Variables, config.Variable{
			Name:  v.Name,
			Value: v.Value,
		})
	}

	return yaml.Marshal(&cfg)
}
