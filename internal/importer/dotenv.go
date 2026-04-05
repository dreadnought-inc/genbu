package importer

import (
	"bufio"
	"io"
	"strings"
)

// DotenvImporter parses .env files (KEY=VALUE format).
type DotenvImporter struct{}

// Import reads a .env file and returns env vars.
func (d *DotenvImporter) Import(r io.Reader) ([]EnvVar, error) {
	var vars []EnvVar
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Strip optional "export " prefix
		line = strings.TrimPrefix(line, "export ")

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = unquote(value)

		if key != "" {
			vars = append(vars, EnvVar{Name: key, Value: value})
		}
	}

	return vars, scanner.Err()
}

// unquote removes surrounding single or double quotes.
func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
