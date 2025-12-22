package security

import "testing"

func TestValidateWorkingDirectory(t *testing.T) {
	tests := []struct {
		name      string
		workDir   string
		basePath  string
		wantError bool
	}{
		{"empty path", "", "/repo", false},
		{"relative path", "subdir", "/repo", false},
		{"nested relative", "a/b/c", "/repo", false},
		{"absolute path", "/etc", "/repo", true},
		{"traverse up", "../outside", "/repo", true},
		{"traverse multiple", "../../etc", "/repo", true},
		{"clean relative", "./subdir", "/repo", false},
		{"traverse then back", "../repo/subdir", "/repo", true},
		{"hidden directory", ".hidden", "/repo", false},
		{"double slash", "a//b", "/repo", false},
		{"windows absolute", "C:\\Windows", "/repo", false}, // Not absolute on Unix
		{"dot only", ".", "/repo", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWorkingDirectory(tt.workDir, tt.basePath)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateWorkingDirectory(%q, %q) error = %v, wantError %v", tt.workDir, tt.basePath, err, tt.wantError)
			}
		})
	}
}

func TestValidateWorkingDirectory_ErrorMessages(t *testing.T) {
	tests := []struct {
		name       string
		workDir    string
		basePath   string
		wantSubstr string
	}{
		{
			name:       "absolute path error",
			workDir:    "/etc/passwd",
			basePath:   "/repo",
			wantSubstr: "absolute paths are not allowed",
		},
		{
			name:       "traverse error",
			workDir:    "../etc",
			basePath:   "/repo",
			wantSubstr: "cannot traverse outside",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWorkingDirectory(tt.workDir, tt.basePath)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !contains(err.Error(), tt.wantSubstr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantSubstr)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
