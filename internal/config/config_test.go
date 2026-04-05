package config

import (
	"testing"
)

func boolPtr(b bool) *bool { return &b }
func intPtr(i int) *int    { return &i }

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
	if cfg.Variables[0].Value != "production" {
		t.Errorf("variables[0].value = %q, want %q", cfg.Variables[0].Value, "production")
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
variables:
  - name: APP_ENV
    value: "production"
groups:
  - name: aws-params
    source:
      type: aws-ssm
      region: ap-northeast-1
    variables:
      - name: DB_HOST
        source:
          path: "/myapp/prod/db-host"
      - name: DB_PORT
        source:
          path: "/myapp/prod/db-port"
`)
	cfg, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	flat := cfg.Flatten()
	if len(flat) != 3 {
		t.Fatalf("flattened count = %d, want 3", len(flat))
	}

	dbHost := flat[1]
	if dbHost.Name != "DB_HOST" {
		t.Errorf("name = %q, want %q", dbHost.Name, "DB_HOST")
	}
	if dbHost.Source == nil {
		t.Fatal("source should not be nil")
	}
	if dbHost.Source.Type != "aws-ssm" {
		t.Errorf("source.type = %q, want %q", dbHost.Source.Type, "aws-ssm")
	}
	if dbHost.Source.Region != "ap-northeast-1" {
		t.Errorf("source.region = %q, want %q", dbHost.Source.Region, "ap-northeast-1")
	}
	if dbHost.Source.Path != "/myapp/prod/db-host" {
		t.Errorf("source.path = %q, want %q", dbHost.Source.Path, "/myapp/prod/db-host")
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
		Type:   "aws-ssm",
		Region: "us-east-1",
	}

	t.Run("nil variable source inherits group", func(t *testing.T) {
		result := mergeSource(group, nil)
		if result.Type != "aws-ssm" {
			t.Errorf("type = %q, want %q", result.Type, "aws-ssm")
		}
		if result.Region != "us-east-1" {
			t.Errorf("region = %q, want %q", result.Region, "us-east-1")
		}
	})

	t.Run("variable overrides group fields", func(t *testing.T) {
		variable := &SourceConfig{
			Path:   "/my/path",
			Region: "ap-northeast-1",
		}
		result := mergeSource(group, variable)
		if result.Type != "aws-ssm" {
			t.Errorf("type = %q, want %q (inherited from group)", result.Type, "aws-ssm")
		}
		if result.Path != "/my/path" {
			t.Errorf("path = %q, want %q", result.Path, "/my/path")
		}
		if result.Region != "ap-northeast-1" {
			t.Errorf("region = %q, want %q (overridden)", result.Region, "ap-northeast-1")
		}
	})
}
