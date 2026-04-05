package importer

import (
	"bufio"
	"io"
	"strings"
)

// INIImporter parses .ini files.
// Section names are used as prefixes: [database] host=x → DATABASE_HOST=x
type INIImporter struct{}

// Import reads an INI file and returns env vars.
func (i *INIImporter) Import(r io.Reader) ([]EnvVar, error) {
	var vars []EnvVar
	scanner := bufio.NewScanner(r)
	section := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Section header
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = unquote(value)

		envName := toEnvName(section, key)
		if envName != "" {
			vars = append(vars, EnvVar{Name: envName, Value: value})
		}
	}

	return vars, scanner.Err()
}

// toEnvName converts a section + key pair to an environment variable name.
// [database] host → DATABASE_HOST
func toEnvName(section, key string) string {
	name := key
	if section != "" {
		name = section + "_" + key
	}
	return strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(name, "-", "_"), ".", "_"))
}
