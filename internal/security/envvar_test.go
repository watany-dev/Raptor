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

func TestFilterSensitiveEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty input",
			input:    []string{},
			expected: []string{},
		},
		{
			name: "no sensitive vars",
			input: []string{
				"PATH=/usr/bin",
				"HOME=/home/user",
				"SHELL=/bin/bash",
			},
			expected: []string{
				"PATH=/usr/bin",
				"HOME=/home/user",
				"SHELL=/bin/bash",
			},
		},
		{
			name: "filter AWS credentials",
			input: []string{
				"PATH=/usr/bin",
				"AWS_ACCESS_KEY_ID=AKIA...",
				"AWS_SECRET_ACCESS_KEY=secret",
				"HOME=/home/user",
			},
			expected: []string{
				"PATH=/usr/bin",
				"HOME=/home/user",
			},
		},
		{
			name: "filter GitHub tokens",
			input: []string{
				"PATH=/usr/bin",
				"GITHUB_TOKEN=ghp_xxxx",
				"GH_TOKEN=ghp_yyyy",
			},
			expected: []string{
				"PATH=/usr/bin",
			},
		},
		{
			name: "filter SSH agent",
			input: []string{
				"SSH_AUTH_SOCK=/tmp/ssh-xxx/agent.123",
				"SSH_AGENT_PID=12345",
				"TERM=xterm",
			},
			expected: []string{
				"TERM=xterm",
			},
		},
		{
			name: "filter suffix patterns",
			input: []string{
				"MY_SECRET=hidden",
				"DB_PASSWORD=pass123",
				"AUTH_TOKEN=abc",
				"NORMAL_VAR=value",
			},
			expected: []string{
				"NORMAL_VAR=value",
			},
		},
		{
			name: "filter prefix patterns",
			input: []string{
				"GIT_CREDENTIAL_HELPER=store",
				"GIT_CREDENTIAL_CACHE=cache",
				"GIT_AUTHOR_NAME=user",
			},
			expected: []string{
				"GIT_AUTHOR_NAME=user",
			},
		},
		{
			name: "case insensitive filtering",
			input: []string{
				"aws_access_key_id=key",
				"Aws_Secret_Access_Key=secret",
				"github_token=token",
			},
			expected: []string{},
		},
		{
			name: "skip invalid format",
			input: []string{
				"VALID=value",
				"INVALID_NO_EQUALS",
				"ANOTHER=ok",
			},
			expected: []string{
				"VALID=value",
				"ANOTHER=ok",
			},
		},
		{
			name: "filter database URLs",
			input: []string{
				"DATABASE_URL=postgres://user:pass@host/db",
				"MONGODB_URI=mongodb://localhost",
				"REDIS_URL=redis://localhost",
			},
			expected: []string{
				"REDIS_URL=redis://localhost",
			},
		},
		{
			name: "filter API keys",
			input: []string{
				"API_KEY=xxx",
				"API_SECRET=yyy",
				"STRIPE_API_KEY=sk_test",
				"SERVICE_NAME=myapp",
			},
			expected: []string{
				"SERVICE_NAME=myapp",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterSensitiveEnvVars(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("FilterSensitiveEnvVars() got %d items, want %d", len(result), len(tt.expected))
				t.Errorf("got: %v", result)
				t.Errorf("want: %v", tt.expected)
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("FilterSensitiveEnvVars()[%d] = %q, want %q", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestIsSensitiveEnvVar(t *testing.T) {
	tests := []struct {
		name     string
		varName  string
		expected bool
	}{
		// Exact matches
		{"AWS_ACCESS_KEY_ID", "AWS_ACCESS_KEY_ID", true},
		{"AWS_SECRET_ACCESS_KEY", "AWS_SECRET_ACCESS_KEY", true},
		{"GITHUB_TOKEN", "GITHUB_TOKEN", true},
		{"SSH_AUTH_SOCK", "SSH_AUTH_SOCK", true},
		{"DATABASE_URL", "DATABASE_URL", true},

		// Case insensitive
		{"lowercase aws key", "aws_access_key_id", true},
		{"mixed case github token", "GitHub_Token", true},

		// Suffix patterns
		{"custom secret", "MY_CUSTOM_SECRET", true},
		{"custom password", "DB_PASSWORD", true},
		{"custom token", "OAUTH_TOKEN", true},
		{"custom credentials", "APP_CREDENTIALS", true},
		{"custom api key", "STRIPE_API_KEY", true},

		// Prefix patterns
		{"git credential helper", "GIT_CREDENTIAL_HELPER", true},
		{"git credential cache", "GIT_CREDENTIAL_STORE", true},

		// Non-sensitive
		{"PATH", "PATH", false},
		{"HOME", "HOME", false},
		{"USER", "USER", false},
		{"SHELL", "SHELL", false},
		{"TERM", "TERM", false},
		{"LANG", "LANG", false},
		{"EDITOR", "EDITOR", false},
		{"PWD", "PWD", false},
		{"GOPATH", "GOPATH", false},

		// Edge cases - not sensitive despite similar names
		{"TOKENIZER", "TOKENIZER", false},
		{"PASSWORD_MIN_LENGTH", "PASSWORD_MIN_LENGTH", false},
		{"SECRET_MODE", "SECRET_MODE", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSensitiveEnvVar(tt.varName)
			if result != tt.expected {
				t.Errorf("isSensitiveEnvVar(%q) = %v, want %v", tt.varName, result, tt.expected)
			}
		})
	}
}

func TestSensitiveEnvVarPatterns_NoDuplicates(t *testing.T) {
	seen := make(map[string]bool)
	for _, pattern := range SensitiveEnvVarPatterns {
		if seen[pattern] {
			t.Errorf("Duplicate pattern found: %q", pattern)
		}
		seen[pattern] = true
	}
}
