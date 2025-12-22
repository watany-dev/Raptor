package security

import (
	"strings"
	"testing"
)

func TestValidateEnvVarName(t *testing.T) {
	tests := []struct {
		name      string
		varName   string
		wantError bool
	}{
		{"valid uppercase", "MY_VAR", false},
		{"valid mixed case", "MyVar", false},
		{"valid with numbers", "VAR_123", false},
		{"starts with underscore", "_VAR", false},
		{"single letter", "X", false},
		{"underscore only", "_", false},

		// Blocked variables
		{"blocked LD_PRELOAD", "LD_PRELOAD", true},
		{"blocked ld_preload lowercase", "ld_preload", true},
		{"blocked BASH_ENV", "BASH_ENV", true},
		{"blocked bash_env lowercase", "bash_env", true},
		{"blocked LD_LIBRARY_PATH", "LD_LIBRARY_PATH", true},
		{"blocked ENV", "ENV", true},
		{"blocked GIT_DIR", "GIT_DIR", true},
		{"blocked IFS", "IFS", true},

		// Invalid format
		{"invalid starts with number", "123VAR", true},
		{"invalid contains dash", "MY-VAR", true},
		{"invalid contains dot", "MY.VAR", true},
		{"invalid contains space", "MY VAR", true},
		{"empty", "", true},
		{"whitespace only", "   ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEnvVarName(tt.varName)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateEnvVarName(%q) error = %v, wantError %v", tt.varName, err, tt.wantError)
			}
		})
	}
}

func TestValidateEnvVarName_ErrorMessages(t *testing.T) {
	tests := []struct {
		name       string
		varName    string
		wantSubstr string
	}{
		{
			name:       "blocked variable has reason",
			varName:    "LD_PRELOAD",
			wantSubstr: "blocked for security",
		},
		{
			name:       "blocked variable mentions injection",
			varName:    "LD_PRELOAD",
			wantSubstr: "inject malicious",
		},
		{
			name:       "invalid format mentions rules",
			varName:    "123VAR",
			wantSubstr: "must start with letter or underscore",
		},
		{
			name:       "empty name error",
			varName:    "",
			wantSubstr: "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEnvVarName(tt.varName)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantSubstr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantSubstr)
			}
		})
	}
}

func TestValidateEnvVarValue(t *testing.T) {
	tests := []struct {
		name      string
		varName   string
		value     string
		wantError bool
	}{
		{"normal value", "VAR", "hello", false},
		{"empty value", "VAR", "", false},
		{"multiline value", "VAR", "line1\nline2\nline3", false},
		{"unicode value", "VAR", "日本語", false},
		{"special chars", "VAR", "!@#$%^&*()", false},

		// Invalid values
		{"null byte", "VAR", "before\x00after", true},
		{"too long", "VAR", strings.Repeat("a", 1024*100+1), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEnvVarValue(tt.varName, tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateEnvVarValue(%q, value) error = %v, wantError %v", tt.varName, err, tt.wantError)
			}
		})
	}
}

func TestValidateEnvVarValue_ErrorMessages(t *testing.T) {
	tests := []struct {
		name       string
		varName    string
		value      string
		wantSubstr string
	}{
		{
			name:       "null byte error",
			varName:    "VAR",
			value:      "test\x00test",
			wantSubstr: "null bytes",
		},
		{
			name:       "too long error",
			varName:    "VAR",
			value:      strings.Repeat("a", 1024*100+1),
			wantSubstr: "too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEnvVarValue(tt.varName, tt.value)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantSubstr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantSubstr)
			}
		})
	}
}

func TestBlockedEnvVars_AllHaveReasons(t *testing.T) {
	for name, reason := range BlockedEnvVars {
		if reason == "" {
			t.Errorf("Blocked variable %q has empty reason", name)
		}
		// Verify name is uppercase (convention)
		if name != strings.ToUpper(name) {
			t.Errorf("Blocked variable %q should be uppercase", name)
		}
	}
}
