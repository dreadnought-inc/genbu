package validator

import (
	"fmt"
	"strings"

	"github.com/dreadnought-inc/genbu/internal/config"
)

// ValidationError represents a single validation failure.
type ValidationError struct {
	VarName string
	Rule    string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: [%s] %s", e.VarName, e.Rule, e.Message)
}

// ValidationErrors is a collection of validation errors.
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	msgs := make([]string, len(e))
	for i, err := range e {
		msgs[i] = err.Error()
	}
	return strings.Join(msgs, "\n")
}

// Input represents a resolved variable to be validated.
type Input struct {
	Name     string
	Value    string
	Validate *config.ValidateConfig
}

// Validate checks all inputs against their validation rules and defaults.
// Returns nil if all pass, or ValidationErrors with all failures.
func Validate(inputs []Input, defaults *config.Defaults) error {
	var errs ValidationErrors

	for _, input := range inputs {
		varErrs := validateVar(input, defaults)
		errs = append(errs, varErrs...)
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

func validateVar(input Input, defaults *config.Defaults) []ValidationError {
	var errs []ValidationError

	rules := input.Validate
	if rules == nil {
		rules = &config.ValidateConfig{}
	}

	// Determine if required (variable-level overrides defaults)
	required := false
	if defaults != nil && defaults.Required != nil {
		required = *defaults.Required
	}
	if rules.Required != nil {
		required = *rules.Required
	}

	if required {
		if err := checkRequired(input.Value); err != nil {
			errs = append(errs, ValidationError{
				VarName: input.Name,
				Rule:    "required",
				Message: err.Error(),
			})
			// If required fails and value is empty, skip other rules
			if input.Value == "" {
				return errs
			}
		}
	}

	// Skip remaining rules if value is empty and not required
	if input.Value == "" {
		return errs
	}

	if rules.Pattern != "" {
		if err := checkPattern(input.Value, rules.Pattern); err != nil {
			errs = append(errs, ValidationError{
				VarName: input.Name,
				Rule:    "pattern",
				Message: err.Error(),
			})
		}
	}

	if len(rules.Enum) > 0 {
		if err := checkEnum(input.Value, rules.Enum); err != nil {
			errs = append(errs, ValidationError{
				VarName: input.Name,
				Rule:    "enum",
				Message: err.Error(),
			})
		}
	}

	if rules.MinLength != nil {
		if err := checkMinLength(input.Value, *rules.MinLength); err != nil {
			errs = append(errs, ValidationError{
				VarName: input.Name,
				Rule:    "min_length",
				Message: err.Error(),
			})
		}
	}

	if rules.MaxLength != nil {
		if err := checkMaxLength(input.Value, *rules.MaxLength); err != nil {
			errs = append(errs, ValidationError{
				VarName: input.Name,
				Rule:    "max_length",
				Message: err.Error(),
			})
		}
	}

	return errs
}
