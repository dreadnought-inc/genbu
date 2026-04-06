package config

import (
	"testing"
)

func TestParse_basic(t *testing.T) {
	data := []byte(`
version: "1"
variables:
  - name: APP_ENV
    value: "production"
  - name: PORT
    value: "8080"
`)
	cfg, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Version != "1" {
		t.Errorf("version = %q, want %q", cfg.Version, "1")
	}
	if len(cfg.Variables) != 2 {
		t.Fatalf("variables count = %d, want 2", len(cfg.Variables))
	}
	if cfg.Variables[0].Name != "APP_ENV" {
		t.Errorf("variables[0].name = %q, want %q", cfg.Variables[0].Name, "APP_ENV")
	}
}

func TestParse_withProvider(t *testing.T) {
	data := []byte(`
version: "1"
provider: aws
variables:
  - name: DB_HOST
    source:
      type: parameter
      key: "/myapp/db-host"
`)
	cfg, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Provider != "aws" {
		t.Errorf("provider = %q, want %q", cfg.Provider, "aws")
	}
	if cfg.Variables[0].Source.Type != "parameter" {
		t.Errorf("source.type = %q, want %q", cfg.Variables[0].Source.Type, "parameter")
	}
	if cfg.Variables[0].Source.Key != "/myapp/db-host" {
		t.Errorf("source.key = %q, want %q", cfg.Variables[0].Source.Key, "/myapp/db-host")
	}
}

func TestParse_invalidProvider(t *testing.T) {
	data := []byte(`
version: "1"
provider: invalid
variables:
  - name: X
    value: "y"
`)
	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for invalid provider")
	}
}

func TestParse_withValidation(t *testing.T) {
	data := []byte(`
version: "1"
defaults:
  required: true
variables:
  - name: APP_ENV
    value: "production"
    validate:
      enum: ["development", "staging", "production"]
  - name: PORT
    value: "8080"
    validate:
      pattern: "^[0-9]+$"
      min_length: 1
      max_length: 5
`)
	cfg, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Defaults == nil || cfg.Defaults.Required == nil || !*cfg.Defaults.Required {
		t.Error("defaults.required should be true")
	}

	v := cfg.Variables[0]
	if v.Validate == nil {
		t.Fatal("variables[0].validate should not be nil")
	}
	if len(v.Validate.Enum) != 3 {
		t.Errorf("enum count = %d, want 3", len(v.Validate.Enum))
	}

	v1 := cfg.Variables[1]
	if v1.Validate.Pattern != "^[0-9]+$" {
		t.Errorf("pattern = %q, want %q", v1.Validate.Pattern, "^[0-9]+$")
	}
	if v1.Validate.MinLength == nil || *v1.Validate.MinLength != 1 {
		t.Error("min_length should be 1")
	}
	if v1.Validate.MaxLength == nil || *v1.Validate.MaxLength != 5 {
		t.Error("max_length should be 5")
	}
}

func TestParse_withGroups(t *testing.T) {
	data := []byte(`
version: "1"
provider: aws
groups:
  - name: db
    source:
      type: parameter
      region: ap-northeast-1
    variables:
      - name: DB_HOST
        source:
          key: "/myapp/prod/db-host"
      - name: DB_PORT
        source:
          key: "/myapp/prod/db-port"
`)
	cfg, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	flat := cfg.Flatten()
	if len(flat) != 2 {
		t.Fatalf("flattened count = %d, want 2", len(flat))
	}

	dbHost := flat[0]
	if dbHost.Source.Type != "parameter" {
		t.Errorf("source.type = %q, want %q", dbHost.Source.Type, "parameter")
	}
	if dbHost.Source.Region != "ap-northeast-1" {
		t.Errorf("source.region = %q, want %q", dbHost.Source.Region, "ap-northeast-1")
	}
	if dbHost.Source.Key != "/myapp/prod/db-host" {
		t.Errorf("source.key = %q, want %q", dbHost.Source.Key, "/myapp/prod/db-host")
	}
}

func TestParse_backwardCompat_awsSsm(t *testing.T) {
	data := []byte(`
version: "1"
variables:
  - name: DB_HOST
    source:
      type: aws-ssm
      path: "/myapp/db-host"
`)
	cfg, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	src := cfg.Variables[0].Source
	if src.Type != "parameter" {
		t.Errorf("type = %q, want %q (normalized from aws-ssm)", src.Type, "parameter")
	}
	if src.Key != "/myapp/db-host" {
		t.Errorf("key = %q, want %q (normalized from path)", src.Key, "/myapp/db-host")
	}
}

func TestParse_backwardCompat_awsSecretsmanager(t *testing.T) {
	data := []byte(`
version: "1"
variables:
  - name: API_SECRET
    source:
      type: aws-secretsmanager
      secret_id: "myapp/api-secret"
      json_key: "key"
`)
	cfg, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	src := cfg.Variables[0].Source
	if src.Type != "secret" {
		t.Errorf("type = %q, want %q (normalized from aws-secretsmanager)", src.Type, "secret")
	}
	if src.Key != "myapp/api-secret" {
		t.Errorf("key = %q, want %q (normalized from secret_id)", src.Key, "myapp/api-secret")
	}
	if src.JSONKey != "key" {
		t.Errorf("json_key = %q, want %q", src.JSONKey, "key")
	}
}

func TestEffectiveKey(t *testing.T) {
	tests := []struct {
		name string
		src  SourceConfig
		want string
	}{
		{"key set", SourceConfig{Key: "k"}, "k"},
		{"path fallback", SourceConfig{Path: "p"}, "p"},
		{"secret_id fallback", SourceConfig{SecretID: "s"}, "s"},
		{"key overrides path", SourceConfig{Key: "k", Path: "p"}, "k"},
		{"empty", SourceConfig{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.src.EffectiveKey()
			if got != tt.want {
				t.Errorf("EffectiveKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParse_missingVersion(t *testing.T) {
	data := []byte(`
variables:
  - name: APP_ENV
    value: "production"
`)
	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for missing version")
	}
}

func TestParse_unsupportedVersion(t *testing.T) {
	data := []byte(`
version: "2"
variables:
  - name: APP_ENV
    value: "production"
`)
	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for unsupported version")
	}
}

func TestParse_duplicateVariable(t *testing.T) {
	data := []byte(`
version: "1"
variables:
  - name: DUPLICATE
    value: "first"
  - name: DUPLICATE
    value: "second"
`)
	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for duplicate variable")
	}
}

func TestParse_emptyVariableName(t *testing.T) {
	data := []byte(`
version: "1"
variables:
  - name: ""
    value: "test"
`)
	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for empty variable name")
	}
}

func TestMergeSource(t *testing.T) {
	group := &SourceConfig{
		Type:   "parameter",
		Region: "us-east-1",
	}

	t.Run("nil variable source inherits group", func(t *testing.T) {
		result := mergeSource(group, nil)
		if result.Type != "parameter" {
			t.Errorf("type = %q, want %q", result.Type, "parameter")
		}
		if result.Region != "us-east-1" {
			t.Errorf("region = %q, want %q", result.Region, "us-east-1")
		}
	})

	t.Run("variable overrides group fields", func(t *testing.T) {
		variable := &SourceConfig{
			Key:    "/my/path",
			Region: "ap-northeast-1",
		}
		result := mergeSource(group, variable)
		if result.Type != "parameter" {
			t.Errorf("type = %q, want %q (inherited from group)", result.Type, "parameter")
		}
		if result.Key != "/my/path" {
			t.Errorf("key = %q, want %q", result.Key, "/my/path")
		}
		if result.Region != "ap-northeast-1" {
			t.Errorf("region = %q, want %q (overridden)", result.Region, "ap-northeast-1")
		}
	})
}
