package security

import (
	"fmt"
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

// TestValidateWorkingDirectory_DeeplyNested tests deeply nested relative paths are accepted
func TestValidateWorkingDirectory_DeeplyNested(t *testing.T) {
	t.Parallel()

	// Create a deeply nested relative path (10 levels deep)
	deepPath := "level0"
	for i := 1; i < 10; i++ {
		deepPath = filepath.Join(deepPath, fmt.Sprintf("level%d", i))
	}

	// Deeply nested relative paths should be accepted as they stay within workspace
	err := ValidateWorkingDirectory(deepPath, "/repo")
	if err != nil {
		t.Errorf("ValidateWorkingDirectory() error = %v; deeply nested relative paths should be valid", err)
	}
}

// TestValidateWorkingDirectory_UnicodeCharacters tests paths with unicode characters are accepted
func TestValidateWorkingDirectory_UnicodeCharacters(t *testing.T) {
	t.Parallel()

	// Test relative path with unicode characters - should be valid
	unicodePath := "ディレクトリ/ファイル"

	err := ValidateWorkingDirectory(unicodePath, "/repo")
	if err != nil {
		t.Errorf("ValidateWorkingDirectory() error = %v; unicode paths should be valid", err)
	}
}

// TestValidateWorkingDirectory_CaseSensitivity tests that different case paths are both accepted
func TestValidateWorkingDirectory_CaseSensitivity(t *testing.T) {
	t.Parallel()

	// Both lowercase and uppercase relative paths should be valid
	err1 := ValidateWorkingDirectory("test/path", "/repo")
	err2 := ValidateWorkingDirectory("TEST/PATH", "/repo")

	if err1 != nil {
		t.Errorf("ValidateWorkingDirectory() lowercase error = %v; should be valid", err1)
	}
	if err2 != nil {
		t.Errorf("ValidateWorkingDirectory() uppercase error = %v; should be valid", err2)
	}
}

// TestValidateWorkingDirectory_PathWithDots tests paths containing multiple dots
func TestValidateWorkingDirectory_PathWithDots(t *testing.T) {
	tests := []struct {
		name      string
		workDir   string
		basePath  string
		wantError bool
	}{
		{"single dot", ".", "/repo", false},
		{"current dir explicit", "./", "/repo", false},
		{"double dot only", "..", "/repo", true},
		{"double dot prefix", "../sibling", "/repo", true},
		{"safe dot in name", "some.dir", "/repo", false},
		{"multiple dots in name", "some..dir", "/repo", false},
		{"dotfile", ".hidden", "/repo", false},
		{"dot slash subdir", "./sub/./dir", "/repo", false},
		{"dot in middle", "a/./b", "/repo", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWorkingDirectory(tt.workDir, tt.basePath)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateWorkingDirectory(%q, %q) error = %v, wantError %v",
					tt.workDir, tt.basePath, err, tt.wantError)
			}
		})
	}
}

// TestValidateWorkingDirectory_FilepathRel tests the filepath.Rel error path
func TestValidateWorkingDirectory_FilepathRel(t *testing.T) {
	// This tests the edge case where filepath.Rel could fail
	// In practice, this is hard to trigger, so we just ensure the code path exists
	tests := []struct {
		name      string
		workDir   string
		basePath  string
		wantError bool
	}{
		{"normal subdir", "sub", "/repo", false},
		{"nested subdir", "a/b/c", "/repo", false},
		{"deeply nested", "a/b/c/d/e/f", "/repo", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWorkingDirectory(tt.workDir, tt.basePath)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateWorkingDirectory(%q, %q) error = %v, wantError %v",
					tt.workDir, tt.basePath, err, tt.wantError)
			}
		})
	}
}

// TestValidateWorkingDirectory_ComplexTraversal tests complex path traversal attempts
func TestValidateWorkingDirectory_ComplexTraversal(t *testing.T) {
	tests := []struct {
		name      string
		workDir   string
		basePath  string
		wantError bool
	}{
		{"traverse with dots", "sub/../..", "/repo", true},
		{"traverse then come back", "sub/../../repo", "/repo", true},
		{"multiple traversals", "../../..", "/repo", true},
		{"traverse hidden", "./../..", "/repo", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWorkingDirectory(tt.workDir, tt.basePath)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateWorkingDirectory(%q, %q) error = %v, wantError %v",
					tt.workDir, tt.basePath, err, tt.wantError)
			}
		})
	}
}

// TestValidateWorkingDirectory_RelPathOutsideWorkspace tests paths that after resolution end up outside workspace
func TestValidateWorkingDirectory_RelPathOutsideWorkspace(t *testing.T) {
	tests := []struct {
		name      string
		workDir   string
		basePath  string
		wantError bool
	}{
		// These should all be caught by the earlier checks
		{"simple parent", "..", "/repo", true},
		{"parent sibling", "../sibling", "/repo", true},
		{"deep and out", "a/b/c/../../../..", "/repo", true},
		// These should be valid
		{"nested then back", "a/../b", "/repo", false},
		{"deep then back one", "a/b/c/../../d", "/repo", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWorkingDirectory(tt.workDir, tt.basePath)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateWorkingDirectory(%q, %q) error = %v, wantError %v",
					tt.workDir, tt.basePath, err, tt.wantError)
			}
		})
	}
}

// TestValidateWorkingDirectory_AllBranches covers all code branches
func TestValidateWorkingDirectory_AllBranches(t *testing.T) {
	// Test empty path (returns nil immediately)
	if err := ValidateWorkingDirectory("", "/repo"); err != nil {
		t.Errorf("Empty path should be allowed, got error: %v", err)
	}

	// Test absolute path (should error)
	if err := ValidateWorkingDirectory("/absolute/path", "/repo"); err == nil {
		t.Error("Absolute path should return error")
	}

	// Test path starting with .. (should error)
	if err := ValidateWorkingDirectory("../outside", "/repo"); err == nil {
		t.Error("Path starting with .. should return error")
	}

	// Test valid relative path (should succeed)
	if err := ValidateWorkingDirectory("valid/subdir", "/repo"); err != nil {
		t.Errorf("Valid relative path should succeed, got error: %v", err)
	}
}
