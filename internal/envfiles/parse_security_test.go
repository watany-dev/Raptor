package envfiles

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseEnvFile_BlockedVariables(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "blocked LD_PRELOAD",
			content:   "LD_PRELOAD=/tmp/evil.so\n",
			wantError: true,
			errorMsg:  "LD_PRELOAD",
		},
		{
			name:      "blocked BASH_ENV",
			content:   "BASH_ENV=/tmp/evil.sh\n",
			wantError: true,
			errorMsg:  "BASH_ENV",
		},
		{
			name:      "blocked LD_LIBRARY_PATH",
			content:   "LD_LIBRARY_PATH=/tmp/evil\n",
			wantError: true,
			errorMsg:  "LD_LIBRARY_PATH",
		},
		{
			name:      "blocked ENV",
			content:   "ENV=/tmp/evil.sh\n",
			wantError: true,
			errorMsg:  "ENV",
		},
		{
			name:      "blocked GIT_DIR",
			content:   "GIT_DIR=/tmp/evil\n",
			wantError: true,
			errorMsg:  "GIT_DIR",
		},
		{
			name:      "blocked IFS",
			content:   "IFS=;\n",
			wantError: true,
			errorMsg:  "IFS",
		},
		{
			name:      "allowed normal var",
			content:   "MY_VAR=value\n",
			wantError: false,
		},
		{
			name:      "allowed PATH (not blocked)",
			content:   "PATH=/usr/bin\n",
			wantError: false,
		},
		{
			name:      "invalid name with dash",
			content:   "MY-VAR=value\n",
			wantError: true,
			errorMsg:  "invalid",
		},
		{
			name:      "invalid name starts with number",
			content:   "123VAR=value\n",
			wantError: true,
			errorMsg:  "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			envFile := filepath.Join(tmpDir, "env")
			if err := os.WriteFile(envFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write env file: %v", err)
			}

			// Parse file
			_, err := ParseEnvFile(envFile)

			if (err != nil) != tt.wantError {
				t.Errorf("ParseEnvFile() error = %v, wantError %v", err, tt.wantError)
			}

			if tt.wantError && tt.errorMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
				}
			}
		})
	}
}

func TestParseEnvFile_BlockedMultiline(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantError bool
		errorMsg  string
	}{
		{
			name: "blocked LD_PRELOAD multiline",
			content: `LD_PRELOAD<<EOF
/tmp/evil.so
EOF
`,
			wantError: true,
			errorMsg:  "LD_PRELOAD",
		},
		{
			name: "blocked BASH_ENV multiline",
			content: `BASH_ENV<<EOF
/tmp/evil.sh
EOF
`,
			wantError: true,
			errorMsg:  "BASH_ENV",
		},
		{
			name: "valid multiline",
			content: `MY_VAR<<EOF
line1
line2
EOF
`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			envFile := filepath.Join(tmpDir, "env")
			if err := os.WriteFile(envFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write env file: %v", err)
			}

			_, err := ParseEnvFile(envFile)

			if (err != nil) != tt.wantError {
				t.Errorf("ParseEnvFile() error = %v, wantError %v", err, tt.wantError)
			}

			if tt.wantError && tt.errorMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
				}
			}
		})
	}
}

func TestParseEnvFile_ValueValidation(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "value with null byte",
			content:   "MY_VAR=before\x00after\n",
			wantError: true,
			errorMsg:  "null bytes",
		},
		{
			name:      "normal value",
			content:   "MY_VAR=normal value\n",
			wantError: false,
		},
		{
			name:      "value with special chars",
			content:   "MY_VAR=!@#$%^&*()\n",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			envFile := filepath.Join(tmpDir, "env")
			if err := os.WriteFile(envFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write env file: %v", err)
			}

			_, err := ParseEnvFile(envFile)

			if (err != nil) != tt.wantError {
				t.Errorf("ParseEnvFile() error = %v, wantError %v", err, tt.wantError)
			}

			if tt.wantError && tt.errorMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
				}
			}
		})
	}
}

func TestParseEnvFile_CaseInsensitiveBlocking(t *testing.T) {
	// Test that blocking works case-insensitively
	blockedVars := []string{
		"ld_preload", "LD_PRELOAD", "Ld_Preload",
		"bash_env", "BASH_ENV", "Bash_Env",
	}

	for _, varName := range blockedVars {
		t.Run(varName, func(t *testing.T) {
			tmpDir := t.TempDir()
			envFile := filepath.Join(tmpDir, "env")
			content := varName + "=/tmp/evil\n"
			if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to write env file: %v", err)
			}

			_, err := ParseEnvFile(envFile)

			if err == nil {
				t.Errorf("Expected %s to be blocked, but it was allowed", varName)
			}
		})
	}
}
