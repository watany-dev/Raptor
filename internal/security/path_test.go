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

func TestValidateGitHubPath(t *testing.T) {
	tests := []struct {
		name          string
		pathEntry     string
		workspacePath string
		wantError     bool
	}{
		// Safe paths
		{"relative path in workspace", "node_modules/.bin", "/workspace", false},
		{"absolute path in workspace", "/workspace/bin", "/workspace", false},
		{"nested workspace path", "/workspace/tools/bin", "/workspace", false},

		// Dangerous system paths (command shadowing attacks)
		{"tmp directory", "/tmp/malicious", "/workspace", true},
		{"var tmp", "/var/tmp/attack", "/workspace", true},
		{"dev shm", "/dev/shm/exploit", "/workspace", true},
		{"run directory", "/run/user/attack", "/workspace", true},
		{"proc directory", "/proc/self", "/workspace", true},
		{"sys directory", "/sys/attack", "/workspace", true},
		{"just tmp", "/tmp", "/workspace", true},

		// Outside workspace
		{"outside workspace", "/etc/evil", "/workspace", true},
		{"home directory", "/home/user/.local/bin", "/workspace", true},
		{"usr local", "/usr/local/bin", "/workspace", true},

		// Invalid entries
		{"empty path", "", "/workspace", true},
		{"whitespace only", "   ", "/workspace", true},
		{"null byte", "/workspace/bin\x00/evil", "/workspace", true},

		// Edge cases
		{"dot path", ".", "/workspace", false},
		{"relative no workspace", "relative/path", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGitHubPath(tt.pathEntry, tt.workspacePath)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateGitHubPath(%q, %q) error = %v, wantError %v",
					tt.pathEntry, tt.workspacePath, err, tt.wantError)
			}
		})
	}
}

func TestValidateGitHubPath_ErrorMessages(t *testing.T) {
	tests := []struct {
		name       string
		pathEntry  string
		workspace  string
		wantSubstr string
	}{
		{
			name:       "tmp blocked",
			pathEntry:  "/tmp/evil",
			workspace:  "/workspace",
			wantSubstr: "command shadowing attacks",
		},
		{
			name:       "outside workspace",
			pathEntry:  "/etc/passwd",
			workspace:  "/workspace",
			wantSubstr: "outside the workspace",
		},
		{
			name:       "empty entry",
			pathEntry:  "",
			workspace:  "/workspace",
			wantSubstr: "cannot be empty",
		},
		{
			name:       "null byte",
			pathEntry:  "/workspace/\x00/bin",
			workspace:  "/workspace",
			wantSubstr: "null bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGitHubPath(tt.pathEntry, tt.workspace)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !contains(err.Error(), tt.wantSubstr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantSubstr)
			}
		})
	}
}
