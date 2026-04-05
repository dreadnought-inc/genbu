package config

// Config represents the top-level genbu configuration.
type Config struct {
	Version    string     `yaml:"version"`
	DumpFormat string     `yaml:"dump_format,omitempty"`
	Defaults   *Defaults  `yaml:"defaults,omitempty"`
	Variables  []Variable `yaml:"variables,omitempty"`
	Groups     []Group    `yaml:"groups,omitempty"`
}

// Defaults defines default settings applied to all variables.
type Defaults struct {
	Required *bool `yaml:"required,omitempty"`
}

// Variable represents a single environment variable definition.
type Variable struct {
	Name     string          `yaml:"name"`
	Value    string          `yaml:"value,omitempty"`
	Default  string          `yaml:"default,omitempty"`
	Source   *SourceConfig   `yaml:"source,omitempty"`
	Validate *ValidateConfig `yaml:"validate,omitempty"`
}

// SourceConfig defines where to fetch a variable's value.
type SourceConfig struct {
	Type     string `yaml:"type"`
	Path     string `yaml:"path,omitempty"`
	SecretID string `yaml:"secret_id,omitempty"`
	JSONKey  string `yaml:"json_key,omitempty"`
	Region   string `yaml:"region,omitempty"`
}

// ValidateConfig defines validation rules for a variable.
type ValidateConfig struct {
	Required  *bool    `yaml:"required,omitempty"`
	Pattern   string   `yaml:"pattern,omitempty"`
	Enum      []string `yaml:"enum,omitempty"`
	MinLength *int     `yaml:"min_length,omitempty"`
	MaxLength *int     `yaml:"max_length,omitempty"`
}

// Group defines a set of variables sharing common source configuration.
type Group struct {
	Name      string        `yaml:"name"`
	Source    *SourceConfig `yaml:"source,omitempty"`
	Variables []Variable    `yaml:"variables,omitempty"`
}

// Flatten expands groups into the top-level variable list, merging group-level
// source config into each variable. Group variables inherit the group's source
// fields unless the variable overrides them.
func (c *Config) Flatten() []Variable {
	vars := make([]Variable, 0, len(c.Variables))
	vars = append(vars, c.Variables...)

	for _, g := range c.Groups {
		for _, v := range g.Variables {
			if g.Source != nil {
				v.Source = mergeSource(g.Source, v.Source)
			}
			vars = append(vars, v)
		}
	}

	return vars
}

// mergeSource merges group-level source into variable-level source.
// Variable-level fields take precedence over group-level fields.
func mergeSource(group, variable *SourceConfig) *SourceConfig {
	if variable == nil {
		copied := *group
		return &copied
	}

	merged := *group
	if variable.Type != "" {
		merged.Type = variable.Type
	}
	if variable.Path != "" {
		merged.Path = variable.Path
	}
	if variable.SecretID != "" {
		merged.SecretID = variable.SecretID
	}
	if variable.JSONKey != "" {
		merged.JSONKey = variable.JSONKey
	}
	if variable.Region != "" {
		merged.Region = variable.Region
	}

	return &merged
}
