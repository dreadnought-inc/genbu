package validator

import (
	"testing"

	"github.com/dreadnought-inc/genbu/internal/config"
)

func boolPtr(b bool) *bool { return &b }
func intPtr(i int) *int    { return &i }

func TestValidate_allPass(t *testing.T) {
	inputs := []Input{
		{
			Name:  "APP_ENV",
			Value: "production",
			Validate: &config.ValidateConfig{
				Enum: []string{"development", "staging", "production"},
			},
		},
		{
			Name:  "PORT",
			Value: "8080",
			Validate: &config.ValidateConfig{
				Required: boolPtr(true),
				Pattern:  "^[0-9]+$",
			},
		},
	}

	err := Validate(inputs, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_requiredFails(t *testing.T) {
	inputs := []Input{
		{
			Name:  "REQUIRED_VAR",
			Value: "",
			Validate: &config.ValidateConfig{
				Required: boolPtr(true),
			},
		},
	}

	err := Validate(inputs, nil)
	if err == nil {
		t.Fatal("expected validation error")
	}

	errs, ok := err.(ValidationErrors)
	if !ok {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if errs[0].Rule != "required" {
		t.Errorf("rule = %q, want %q", errs[0].Rule, "required")
	}
}

func TestValidate_defaultRequired(t *testing.T) {
	defaults := &config.Defaults{Required: boolPtr(true)}
	inputs := []Input{
		{
			Name:  "EMPTY_VAR",
			Value: "",
		},
	}

	err := Validate(inputs, defaults)
	if err == nil {
		t.Fatal("expected validation error from default required")
	}
}

func TestValidate_variableOverridesDefault(t *testing.T) {
	defaults := &config.Defaults{Required: boolPtr(true)}
	inputs := []Input{
		{
			Name:  "OPTIONAL_VAR",
			Value: "",
			Validate: &config.ValidateConfig{
				Required: boolPtr(false),
			},
		},
	}

	err := Validate(inputs, defaults)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_patternFails(t *testing.T) {
	inputs := []Input{
		{
			Name:  "PORT",
			Value: "not-a-number",
			Validate: &config.ValidateConfig{
				Pattern: "^[0-9]+$",
			},
		},
	}

	err := Validate(inputs, nil)
	if err == nil {
		t.Fatal("expected validation error")
	}

	errs := err.(ValidationErrors)
	if errs[0].Rule != "pattern" {
		t.Errorf("rule = %q, want %q", errs[0].Rule, "pattern")
	}
}

func TestValidate_enumFails(t *testing.T) {
	inputs := []Input{
		{
			Name:  "APP_ENV",
			Value: "unknown",
			Validate: &config.ValidateConfig{
				Enum: []string{"dev", "staging", "prod"},
			},
		},
	}

	err := Validate(inputs, nil)
	if err == nil {
		t.Fatal("expected validation error")
	}

	errs := err.(ValidationErrors)
	if errs[0].Rule != "enum" {
		t.Errorf("rule = %q, want %q", errs[0].Rule, "enum")
	}
}

func TestValidate_minLengthFails(t *testing.T) {
	inputs := []Input{
		{
			Name:  "API_KEY",
			Value: "short",
			Validate: &config.ValidateConfig{
				MinLength: intPtr(32),
			},
		},
	}

	err := Validate(inputs, nil)
	if err == nil {
		t.Fatal("expected validation error")
	}

	errs := err.(ValidationErrors)
	if errs[0].Rule != "min_length" {
		t.Errorf("rule = %q, want %q", errs[0].Rule, "min_length")
	}
}

func TestValidate_maxLengthFails(t *testing.T) {
	inputs := []Input{
		{
			Name:  "SHORT_VAR",
			Value: "this is way too long",
			Validate: &config.ValidateConfig{
				MaxLength: intPtr(5),
			},
		},
	}

	err := Validate(inputs, nil)
	if err == nil {
		t.Fatal("expected validation error")
	}

	errs := err.(ValidationErrors)
	if errs[0].Rule != "max_length" {
		t.Errorf("rule = %q, want %q", errs[0].Rule, "max_length")
	}
}

func TestValidate_multipleErrors(t *testing.T) {
	inputs := []Input{
		{
			Name:  "BAD_PORT",
			Value: "abc",
			Validate: &config.ValidateConfig{
				Pattern:   "^[0-9]+$",
				MinLength: intPtr(5),
			},
		},
	}

	err := Validate(inputs, nil)
	if err == nil {
		t.Fatal("expected validation errors")
	}

	errs := err.(ValidationErrors)
	if len(errs) != 2 {
		t.Fatalf("expected 2 errors, got %d: %v", len(errs), errs)
	}
}

func TestValidate_emptyValueSkipsRulesWhenNotRequired(t *testing.T) {
	inputs := []Input{
		{
			Name:  "OPTIONAL",
			Value: "",
			Validate: &config.ValidateConfig{
				Pattern:   "^[0-9]+$",
				MinLength: intPtr(1),
			},
		},
	}

	err := Validate(inputs, nil)
	if err != nil {
		t.Fatalf("expected no error for empty optional value, got: %v", err)
	}
}

func TestValidate_noValidateConfig(t *testing.T) {
	inputs := []Input{
		{
			Name:  "PLAIN_VAR",
			Value: "anything",
		},
	}

	err := Validate(inputs, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
