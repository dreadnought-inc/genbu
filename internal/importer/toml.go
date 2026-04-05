package importer

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// TOMLImporter parses TOML files.
// Nested keys are flattened with underscore separators and uppercased.
// [database] host = "x" → DATABASE_HOST=x
type TOMLImporter struct{}

// Import reads a TOML file and returns env vars.
func (t *TOMLImporter) Import(r io.Reader) ([]EnvVar, error) {
	var data map[string]interface{}
	if _, err := toml.NewDecoder(r).Decode(&data); err != nil {
		return nil, fmt.Errorf("parsing toml: %w", err)
	}

	var vars []EnvVar
	flatten("", data, &vars)

	// Sort for deterministic output
	sort.Slice(vars, func(i, j int) bool {
		return vars[i].Name < vars[j].Name
	})

	return vars, nil
}

func flatten(prefix string, data map[string]interface{}, vars *[]EnvVar) {
	for key, value := range data {
		envKey := key
		if prefix != "" {
			envKey = prefix + "_" + key
		}

		switch v := value.(type) {
		case map[string]interface{}:
			flatten(envKey, v, vars)
		default:
			name := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(envKey, "-", "_"), ".", "_"))
			*vars = append(*vars, EnvVar{Name: name, Value: fmt.Sprintf("%v", v)})
		}
	}
}
