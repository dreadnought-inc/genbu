package validator

import (
	"fmt"
	"regexp"
)

// checkRequired validates that the value is non-empty.
func checkRequired(value string) error {
	if value == "" {
		return fmt.Errorf("value is required but empty")
	}
	return nil
}

// checkPattern validates that the value matches the given regex pattern.
func checkPattern(value, pattern string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
	}
	if !re.MatchString(value) {
		return fmt.Errorf("value %q does not match pattern %q", value, pattern)
	}
	return nil
}

// checkEnum validates that the value is one of the allowed values.
func checkEnum(value string, allowed []string) error {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return fmt.Errorf("value %q is not in allowed values %v", value, allowed)
}

// checkMinLength validates that the value is at least minLen characters long.
func checkMinLength(value string, minLen int) error {
	if len(value) < minLen {
		return fmt.Errorf("value length %d is less than minimum %d", len(value), minLen)
	}
	return nil
}

// checkMaxLength validates that the value is at most maxLen characters long.
func checkMaxLength(value string, maxLen int) error {
	if len(value) > maxLen {
		return fmt.Errorf("value length %d exceeds maximum %d", len(value), maxLen)
	}
	return nil
}
