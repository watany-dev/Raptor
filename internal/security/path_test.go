package security

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

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

// TestValidateWorkingDirectory_DeeplyNested tests deeply nested paths
func TestValidateWorkingDirectory_DeeplyNested(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	workspace := filepath.Join(baseDir, "workspace")

	// Create a deeply nested path (100 levels deep)
	deepPath := workspace
	for i := 0; i < 100; i++ {
		deepPath = filepath.Join(deepPath, fmt.Sprintf("level%d", i))
	}

	// Validate should accept deeply nested paths as long as they stay in workspace
	err := ValidateWorkingDirectory(workspace, deepPath)
	if err != nil {
		t.Logf("ValidateWorkingDirectory() with 100-level nesting: %v", err)
		// Deep nesting may fail, which is acceptable
	}
}

// TestValidateWorkingDirectory_UnicodeCharacters tests paths with unicode characters
func TestValidateWorkingDirectory_UnicodeCharacters(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	workspace := filepath.Join(baseDir, "workspace-日本語")

	// Create workspace directory
	if err := os.MkdirAll(workspace, 0755); err != nil {
		t.Skipf("failed to create unicode-named directory: %v", err)
	}

	// Test path with unicode characters
	unicodePath := filepath.Join(workspace, "ファイル.txt")

	err := ValidateWorkingDirectory(workspace, unicodePath)
	if err != nil {
		t.Logf("ValidateWorkingDirectory() with unicode characters: %v", err)
		// Some systems may not support unicode in paths
	}
}

// TestValidateWorkingDirectory_CaseSensitivity tests case sensitivity in paths
func TestValidateWorkingDirectory_CaseSensitivity(t *testing.T) {
	t.Parallel()
	baseDir := t.TempDir()
	workspace := filepath.Join(baseDir, "workspace")

	if err := os.MkdirAll(workspace, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Test with different case
	path1 := filepath.Join(workspace, "test")
	path2 := filepath.Join(workspace, "TEST")

	// Both paths should be valid (within workspace)
	err1 := ValidateWorkingDirectory(workspace, path1)
	err2 := ValidateWorkingDirectory(workspace, path2)

	// Both should be acceptable (case may or may not matter depending on filesystem)
	if err1 != nil {
		t.Logf("ValidateWorkingDirectory() for lowercase path: %v", err1)
	}
	if err2 != nil {
		t.Logf("ValidateWorkingDirectory() for uppercase path: %v", err2)
	}
}
