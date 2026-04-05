package validator

import (
	"testing"
)

func TestCheckRequired(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"non-empty", "hello", false},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkRequired(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkRequired(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestCheckPattern(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		pattern string
		wantErr bool
	}{
		{"matches", "12345", "^[0-9]+$", false},
		{"no match", "abc", "^[0-9]+$", true},
		{"partial match anchored", "abc123", "^[0-9]+$", true},
		{"url pattern", "postgres://localhost", "^postgres://", false},
		{"invalid regex", "test", "[invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkPattern(tt.value, tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkPattern(%q, %q) error = %v, wantErr %v", tt.value, tt.pattern, err, tt.wantErr)
			}
		})
	}
}

func TestCheckEnum(t *testing.T) {
	allowed := []string{"dev", "staging", "prod"}

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"allowed value", "dev", false},
		{"another allowed", "prod", false},
		{"not allowed", "test", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkEnum(tt.value, allowed)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkEnum(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestCheckMinLength(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		minLen  int
		wantErr bool
	}{
		{"exact length", "abc", 3, false},
		{"longer", "abcd", 3, false},
		{"shorter", "ab", 3, true},
		{"empty", "", 1, true},
		{"zero min", "", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkMinLength(tt.value, tt.minLen)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkMinLength(%q, %d) error = %v, wantErr %v", tt.value, tt.minLen, err, tt.wantErr)
			}
		})
	}
}

func TestCheckMaxLength(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		maxLen  int
		wantErr bool
	}{
		{"exact length", "abc", 3, false},
		{"shorter", "ab", 3, false},
		{"longer", "abcd", 3, true},
		{"empty", "", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkMaxLength(tt.value, tt.maxLen)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkMaxLength(%q, %d) error = %v, wantErr %v", tt.value, tt.maxLen, err, tt.wantErr)
			}
		})
	}
}
