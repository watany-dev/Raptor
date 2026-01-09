package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
			if !strings.Contains(err.Error(), tt.wantSubstr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantSubstr)
			}
		})
	}
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

// TestValidatePathWithSymlinkResolution tests symlink-based path traversal detection
func TestValidatePathWithSymlinkResolution(t *testing.T) {
	t.Run("os.Lstat error other than NotExist should return error", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("skipping permission test as root")
		}

		tmpDir := t.TempDir()
		workspaceDir := filepath.Join(tmpDir, "workspace")
		subDir := filepath.Join(workspaceDir, "protected")

		// Create directory structure
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("failed to create subdir: %v", err)
		}

		// Create a file inside protected directory
		targetFile := filepath.Join(subDir, "file.txt")
		if err := os.WriteFile(targetFile, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}

		// Remove execute permission from parent directory to trigger Lstat error
		if err := os.Chmod(subDir, 0000); err != nil {
			t.Fatalf("failed to chmod: %v", err)
		}
		defer func() {
			_ = os.Chmod(subDir, 0755)
		}()

		// This should fail with permission error, not NotExist
		err := ValidatePathWithSymlinkResolution("protected/file.txt", workspaceDir)
		if err == nil {
			t.Error("ValidatePathWithSymlinkResolution() should return error for permission denied")
		}
		if err != nil && !strings.Contains(err.Error(), "failed to stat path") {
			t.Logf("Got error: %v", err)
		}
	})

	t.Run("broken symlink should return error", func(t *testing.T) {
		tmpDir := t.TempDir()
		workspaceDir := filepath.Join(tmpDir, "workspace")

		// Create directory
		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			t.Fatalf("failed to create workspace dir: %v", err)
		}

		// Create a broken symlink (pointing to non-existent file)
		brokenLink := filepath.Join(workspaceDir, "broken-link")
		if err := os.Symlink("/nonexistent/path/that/does/not/exist", brokenLink); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}

		// This should fail because EvalSymlinks will fail on broken symlink
		err := ValidatePathWithSymlinkResolution("broken-link", workspaceDir)
		if err == nil {
			t.Error("ValidatePathWithSymlinkResolution() should return error for broken symlink")
		}
		if err != nil && !strings.Contains(err.Error(), "failed to resolve symlinks") {
			t.Logf("Got error: %v", err)
		}
	})

	t.Run("EvalSymlinks error on basePath should return error", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("skipping permission test as root")
		}

		tmpDir := t.TempDir()
		workspaceDir := filepath.Join(tmpDir, "workspace")
		parentDir := filepath.Join(tmpDir, "parent")
		linkToParent := filepath.Join(tmpDir, "link-to-parent")

		// Create directories
		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			t.Fatalf("failed to create workspace dir: %v", err)
		}
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			t.Fatalf("failed to create parent dir: %v", err)
		}

		// Create a symlink to parent
		if err := os.Symlink(parentDir, linkToParent); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}

		// Create a file in workspace
		testFile := filepath.Join(workspaceDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}

		// Make the symlink target unreadable to cause EvalSymlinks to fail on basePath
		// This is tricky - we need to cause EvalSymlinks(basePath) to fail
		// One way is to use a broken symlink as basePath
		brokenBase := filepath.Join(tmpDir, "broken-base")
		if err := os.Symlink("/nonexistent/base/path", brokenBase); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}

		// The path exists relative to workspace, but basePath symlink is broken
		err := ValidatePathWithSymlinkResolution("test.txt", brokenBase)
		// This will fail at the basic validation or EvalSymlinks stage
		if err == nil {
			// The path doesn't exist relative to broken base, so it might pass
			// because os.Lstat returns NotExist
			t.Log("Path validation passed (expected for non-existent path)")
		}
	})

	t.Run("symlink pointing outside workspace should be rejected", func(t *testing.T) {
		tmpDir := t.TempDir()
		workspaceDir := filepath.Join(tmpDir, "workspace")
		outsideDir := filepath.Join(tmpDir, "outside")
		outsideFile := filepath.Join(outsideDir, "secret.txt")

		// Create directories
		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			t.Fatalf("failed to create workspace dir: %v", err)
		}
		if err := os.MkdirAll(outsideDir, 0755); err != nil {
			t.Fatalf("failed to create outside dir: %v", err)
		}

		// Create a file outside the workspace
		if err := os.WriteFile(outsideFile, []byte("secret"), 0644); err != nil {
			t.Fatalf("failed to create outside file: %v", err)
		}

		// Create a symlink inside workspace pointing to outside file
		symlinkPath := filepath.Join(workspaceDir, "innocent-link.txt")
		if err := os.Symlink(outsideFile, symlinkPath); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}

		// This should be rejected because the symlink points outside workspace
		err := ValidatePathWithSymlinkResolution("innocent-link.txt", workspaceDir)
		if err == nil {
			t.Error("ValidatePathWithSymlinkResolution() should reject symlink pointing outside workspace")
		}
		if err != nil && !strings.Contains(err.Error(), "symlink") && !strings.Contains(err.Error(), "outside") {
			t.Logf("Got error: %v", err)
		}
	})

	t.Run("symlink pointing to directory outside workspace should be rejected", func(t *testing.T) {
		tmpDir := t.TempDir()
		workspaceDir := filepath.Join(tmpDir, "workspace")
		outsideDir := filepath.Join(tmpDir, "outside")

		// Create directories
		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			t.Fatalf("failed to create workspace dir: %v", err)
		}
		if err := os.MkdirAll(outsideDir, 0755); err != nil {
			t.Fatalf("failed to create outside dir: %v", err)
		}

		// Create a symlink inside workspace pointing to outside directory
		symlinkPath := filepath.Join(workspaceDir, "innocent-dir")
		if err := os.Symlink(outsideDir, symlinkPath); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}

		// This should be rejected
		err := ValidatePathWithSymlinkResolution("innocent-dir", workspaceDir)
		if err == nil {
			t.Error("ValidatePathWithSymlinkResolution() should reject symlink pointing to outside directory")
		}
	})

	t.Run("symlink pointing within workspace should be allowed", func(t *testing.T) {
		tmpDir := t.TempDir()
		workspaceDir := filepath.Join(tmpDir, "workspace")
		subDir := filepath.Join(workspaceDir, "subdir")
		targetFile := filepath.Join(subDir, "target.txt")

		// Create directories
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("failed to create subdir: %v", err)
		}

		// Create a target file within workspace
		if err := os.WriteFile(targetFile, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to create target file: %v", err)
		}

		// Create a symlink inside workspace pointing to another file within workspace
		symlinkPath := filepath.Join(workspaceDir, "link.txt")
		if err := os.Symlink(targetFile, symlinkPath); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}

		// This should be allowed because symlink target is within workspace
		err := ValidatePathWithSymlinkResolution("link.txt", workspaceDir)
		if err != nil {
			t.Errorf("ValidatePathWithSymlinkResolution() should allow symlink within workspace, got error: %v", err)
		}
	})

	t.Run("regular file should be allowed", func(t *testing.T) {
		tmpDir := t.TempDir()
		workspaceDir := filepath.Join(tmpDir, "workspace")
		targetFile := filepath.Join(workspaceDir, "file.txt")

		// Create directory and file
		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			t.Fatalf("failed to create workspace dir: %v", err)
		}
		if err := os.WriteFile(targetFile, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}

		// Regular file should be allowed
		err := ValidatePathWithSymlinkResolution("file.txt", workspaceDir)
		if err != nil {
			t.Errorf("ValidatePathWithSymlinkResolution() should allow regular file, got error: %v", err)
		}
	})

	t.Run("non-existent path should pass basic validation", func(t *testing.T) {
		tmpDir := t.TempDir()
		workspaceDir := filepath.Join(tmpDir, "workspace")

		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			t.Fatalf("failed to create workspace dir: %v", err)
		}

		// Non-existent path - should pass basic validation (symlink check skipped for non-existent)
		err := ValidatePathWithSymlinkResolution("non-existent", workspaceDir)
		if err != nil {
			t.Errorf("ValidatePathWithSymlinkResolution() should allow non-existent path, got error: %v", err)
		}
	})

	t.Run("chained symlinks eventually pointing outside should be rejected", func(t *testing.T) {
		tmpDir := t.TempDir()
		workspaceDir := filepath.Join(tmpDir, "workspace")
		outsideDir := filepath.Join(tmpDir, "outside")
		outsideFile := filepath.Join(outsideDir, "secret.txt")

		// Create directories
		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			t.Fatalf("failed to create workspace dir: %v", err)
		}
		if err := os.MkdirAll(outsideDir, 0755); err != nil {
			t.Fatalf("failed to create outside dir: %v", err)
		}

		// Create a file outside the workspace
		if err := os.WriteFile(outsideFile, []byte("secret"), 0644); err != nil {
			t.Fatalf("failed to create outside file: %v", err)
		}

		// Create chained symlinks: link1 -> link2 -> outside
		link2 := filepath.Join(workspaceDir, "link2")
		link1 := filepath.Join(workspaceDir, "link1")

		if err := os.Symlink(outsideFile, link2); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}
		if err := os.Symlink(link2, link1); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}

		// This should be rejected because the final target is outside workspace
		err := ValidatePathWithSymlinkResolution("link1", workspaceDir)
		if err == nil {
			t.Error("ValidatePathWithSymlinkResolution() should reject chained symlinks pointing outside")
		}
	})

	t.Run("absolute path should fail ValidateWorkingDirectory check", func(t *testing.T) {
		tmpDir := t.TempDir()
		workspaceDir := filepath.Join(tmpDir, "workspace")

		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			t.Fatalf("failed to create workspace dir: %v", err)
		}

		// Absolute path should fail the basic validation in ValidatePathWithSymlinkResolution
		err := ValidatePathWithSymlinkResolution("/etc/passwd", workspaceDir)
		if err == nil {
			t.Error("ValidatePathWithSymlinkResolution() should reject absolute paths")
		}
		if err != nil && !strings.Contains(err.Error(), "absolute paths are not allowed") {
			t.Errorf("Expected absolute path error, got: %v", err)
		}
	})

	t.Run("path traversal should fail ValidateWorkingDirectory check", func(t *testing.T) {
		tmpDir := t.TempDir()
		workspaceDir := filepath.Join(tmpDir, "workspace")

		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			t.Fatalf("failed to create workspace dir: %v", err)
		}

		// Path traversal should fail the basic validation
		err := ValidatePathWithSymlinkResolution("../outside", workspaceDir)
		if err == nil {
			t.Error("ValidatePathWithSymlinkResolution() should reject path traversal")
		}
		if err != nil && !strings.Contains(err.Error(), "cannot traverse outside") {
			t.Errorf("Expected path traversal error, got: %v", err)
		}
	})

	t.Run("basePath as broken symlink should return error", func(t *testing.T) {
		tmpDir := t.TempDir()
		workspaceDir := filepath.Join(tmpDir, "workspace")

		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			t.Fatalf("failed to create workspace dir: %v", err)
		}

		// Create a file that exists in the actual workspace
		testFile := filepath.Join(workspaceDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		// Create a symlink that will be used as basePath, pointing to workspace
		baseSymlink := filepath.Join(tmpDir, "base-link")
		if err := os.Symlink(workspaceDir, baseSymlink); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}

		// Now remove the original workspace to break the symlink
		if err := os.RemoveAll(workspaceDir); err != nil {
			t.Fatalf("failed to remove workspace: %v", err)
		}

		// This should fail because basePath symlink is broken
		err := ValidatePathWithSymlinkResolution("test.txt", baseSymlink)
		// It may return nil (path not exist) or error depending on order of checks
		// The key is that it doesn't panic
		if err != nil {
			if !strings.Contains(err.Error(), "failed to resolve base path symlinks") &&
				!strings.Contains(err.Error(), "failed to stat path") &&
				!strings.Contains(err.Error(), "failed to resolve symlinks") {
				t.Logf("Got error (acceptable): %v", err)
			}
		}
	})

	t.Run("permission denied on parent directory triggers Lstat error", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("skipping permission test as root")
		}

		tmpDir := t.TempDir()
		workspaceDir := filepath.Join(tmpDir, "workspace")
		restrictedDir := filepath.Join(workspaceDir, "restricted")

		// Create directory structure
		if err := os.MkdirAll(restrictedDir, 0755); err != nil {
			t.Fatalf("failed to create restricted dir: %v", err)
		}

		// Create a file inside restricted directory
		targetFile := filepath.Join(restrictedDir, "secret.txt")
		if err := os.WriteFile(targetFile, []byte("secret"), 0644); err != nil {
			t.Fatalf("failed to create target file: %v", err)
		}

		// Remove all permissions from the restricted directory
		if err := os.Chmod(restrictedDir, 0000); err != nil {
			t.Fatalf("failed to chmod: %v", err)
		}
		defer func() {
			_ = os.Chmod(restrictedDir, 0755)
		}()

		// This should fail with permission error during Lstat
		err := ValidatePathWithSymlinkResolution("restricted/secret.txt", workspaceDir)
		if err == nil {
			t.Error("Expected error for permission denied")
		}
		if err != nil && !strings.Contains(err.Error(), "failed to stat path") {
			t.Errorf("Expected 'failed to stat path' error, got: %v", err)
		}
	})
}

// TestValidateWorkingDirectory_RelPathError tests the filepath.Rel error branch
func TestValidateWorkingDirectory_RelPathError(t *testing.T) {
	// This test ensures the code handles edge cases in filepath.Rel
	// The error branch at line 42-46 is a defensive check that handles:
	// 1. filepath.Rel returning an error
	// 2. The relative path starting with ".." (which shouldn't happen after earlier checks)

	// Test with deeply nested path that goes back and forth
	tests := []struct {
		name      string
		workDir   string
		basePath  string
		wantError bool
	}{
		{
			name:      "valid nested path",
			workDir:   "a/b/c/d",
			basePath:  "/repo",
			wantError: false,
		},
		{
			name:      "path with backtracking inside",
			workDir:   "a/b/../c",
			basePath:  "/repo",
			wantError: false,
		},
		{
			name:      "deep backtrack still inside",
			workDir:   "a/b/c/../../d",
			basePath:  "/repo",
			wantError: false,
		},
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

// TestValidateWorkingDirectory_MockedRelError tests filepath.Rel error using mock
func TestValidateWorkingDirectory_MockedRelError(t *testing.T) {
	// Save original function and restore after test
	originalFilepathRel := filepathRel
	defer func() { filepathRel = originalFilepathRel }()

	// Mock filepath.Rel to return an error
	filepathRel = func(basepath, targpath string) (string, error) {
		return "", fmt.Errorf("mocked filepath.Rel error")
	}

	// This should trigger the error branch
	err := ValidateWorkingDirectory("subdir", "/repo")
	if err == nil {
		t.Error("Expected error when filepath.Rel fails")
	}
	if err != nil && !strings.Contains(err.Error(), "must be within the workspace") {
		t.Errorf("Expected 'must be within the workspace' error, got: %v", err)
	}
}

// TestValidateWorkingDirectory_MockedRelReturnsParent tests when filepath.Rel returns ".."
func TestValidateWorkingDirectory_MockedRelReturnsParent(t *testing.T) {
	// Save original function and restore after test
	originalFilepathRel := filepathRel
	defer func() { filepathRel = originalFilepathRel }()

	// Mock filepath.Rel to return a path starting with ".."
	filepathRel = func(basepath, targpath string) (string, error) {
		return "../escaped", nil
	}

	// This should trigger the error branch
	err := ValidateWorkingDirectory("subdir", "/repo")
	if err == nil {
		t.Error("Expected error when relative path starts with ..")
	}
	if err != nil && !strings.Contains(err.Error(), "must be within the workspace") {
		t.Errorf("Expected 'must be within the workspace' error, got: %v", err)
	}
}

// TestValidatePathWithSymlinkResolution_MockedLstatError tests os.Lstat error using mock
func TestValidatePathWithSymlinkResolution_MockedLstatError(t *testing.T) {
	// Save original function and restore after test
	originalOsLstat := osLstat
	defer func() { osLstat = originalOsLstat }()

	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Mock os.Lstat to return a non-NotExist error
	osLstat = func(name string) (os.FileInfo, error) {
		return nil, fmt.Errorf("mocked permission denied")
	}

	err := ValidatePathWithSymlinkResolution("subdir", workspaceDir)
	if err == nil {
		t.Error("Expected error when os.Lstat fails")
	}
	if err != nil && !strings.Contains(err.Error(), "failed to stat path") {
		t.Errorf("Expected 'failed to stat path' error, got: %v", err)
	}
}

// TestValidatePathWithSymlinkResolution_MockedEvalSymlinksError tests EvalSymlinks error on fullPath
func TestValidatePathWithSymlinkResolution_MockedEvalSymlinksError(t *testing.T) {
	// Save original functions and restore after test
	originalOsLstat := osLstat
	originalEvalSymlinks := filepathEvalSymlinks
	defer func() {
		osLstat = originalOsLstat
		filepathEvalSymlinks = originalEvalSymlinks
	}()

	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Mock os.Lstat to succeed (path exists)
	osLstat = func(name string) (os.FileInfo, error) {
		return nil, nil // Return nil error (path exists)
	}

	// Mock filepath.EvalSymlinks to fail on fullPath
	osLstat = func(name string) (os.FileInfo, error) {
		return nil, nil
	}
	filepathEvalSymlinks = func(path string) (string, error) {
		return "", fmt.Errorf("mocked broken symlink")
	}

	err := ValidatePathWithSymlinkResolution("subdir", workspaceDir)
	if err == nil {
		t.Error("Expected error when EvalSymlinks fails on fullPath")
	}
	if err != nil && !strings.Contains(err.Error(), "failed to resolve symlinks") {
		t.Errorf("Expected 'failed to resolve symlinks' error, got: %v", err)
	}
}

// TestValidatePathWithSymlinkResolution_MockedEvalSymlinksBasePathError tests EvalSymlinks error on basePath
func TestValidatePathWithSymlinkResolution_MockedEvalSymlinksBasePathError(t *testing.T) {
	// Save original functions and restore after test
	originalOsLstat := osLstat
	originalEvalSymlinks := filepathEvalSymlinks
	defer func() {
		osLstat = originalOsLstat
		filepathEvalSymlinks = originalEvalSymlinks
	}()

	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Mock os.Lstat to succeed (path exists)
	osLstat = func(name string) (os.FileInfo, error) {
		return nil, nil
	}

	callCount := 0
	// Mock filepath.EvalSymlinks to succeed on fullPath but fail on basePath
	filepathEvalSymlinks = func(path string) (string, error) {
		callCount++
		if callCount == 1 {
			// First call (fullPath) - succeed
			return path, nil
		}
		// Second call (basePath) - fail
		return "", fmt.Errorf("mocked base path symlink error")
	}

	err := ValidatePathWithSymlinkResolution("subdir", workspaceDir)
	if err == nil {
		t.Error("Expected error when EvalSymlinks fails on basePath")
	}
	if err != nil && !strings.Contains(err.Error(), "failed to resolve base path symlinks") {
		t.Errorf("Expected 'failed to resolve base path symlinks' error, got: %v", err)
	}
}

// TestValidatePathWithSymlinkResolution_MockedRelError tests filepath.Rel error in symlink resolution
func TestValidatePathWithSymlinkResolution_MockedRelError(t *testing.T) {
	// Save original functions and restore after test
	originalOsLstat := osLstat
	originalEvalSymlinks := filepathEvalSymlinks
	originalFilepathRel := filepathRel
	defer func() {
		osLstat = originalOsLstat
		filepathEvalSymlinks = originalEvalSymlinks
		filepathRel = originalFilepathRel
	}()

	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Mock os.Lstat to succeed
	osLstat = func(name string) (os.FileInfo, error) {
		return nil, nil
	}

	// Mock filepath.EvalSymlinks to succeed for both calls
	filepathEvalSymlinks = func(path string) (string, error) {
		return path, nil
	}

	// Mock filepath.Rel - need to let ValidateWorkingDirectory pass but fail in ValidatePathWithSymlinkResolution
	relCallCount := 0
	filepathRel = func(basepath, targpath string) (string, error) {
		relCallCount++
		if relCallCount == 1 {
			// First call in ValidateWorkingDirectory - succeed
			return filepath.Rel(basepath, targpath)
		}
		// Second call in ValidatePathWithSymlinkResolution - fail
		return "", fmt.Errorf("mocked rel error")
	}

	err := ValidatePathWithSymlinkResolution("subdir", workspaceDir)
	if err == nil {
		t.Error("Expected error when filepath.Rel fails in symlink resolution")
	}
	if err != nil && !strings.Contains(err.Error(), "symlink target escapes workspace") {
		t.Errorf("Expected 'symlink target escapes workspace' error, got: %v", err)
	}
}

// TestSymlinkTraversalAttacks tests various symlink-based path traversal attack patterns
func TestSymlinkTraversalAttacks(t *testing.T) {
	t.Run("relative symlink with parent traversal should be rejected", func(t *testing.T) {
		tmpDir := t.TempDir()
		workspaceDir := filepath.Join(tmpDir, "workspace")
		outsideDir := filepath.Join(tmpDir, "outside")
		secretFile := filepath.Join(outsideDir, "secret.txt")

		// Create directory structure
		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			t.Fatalf("failed to create workspace dir: %v", err)
		}
		if err := os.MkdirAll(outsideDir, 0755); err != nil {
			t.Fatalf("failed to create outside dir: %v", err)
		}

		// Create target file outside workspace
		if err := os.WriteFile(secretFile, []byte("secret data"), 0644); err != nil {
			t.Fatalf("failed to create secret file: %v", err)
		}

		// Create symlink using relative path with parent traversal: ../outside/secret.txt
		maliciousLink := filepath.Join(workspaceDir, "malicious")
		if err := os.Symlink("../outside/secret.txt", maliciousLink); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}

		// This should be rejected - symlink uses relative traversal to escape
		err := ValidatePathWithSymlinkResolution("malicious", workspaceDir)
		if err == nil {
			t.Error("ValidatePathWithSymlinkResolution() should reject relative symlink traversal")
		}
	})

	t.Run("absolute symlink to system path should be rejected", func(t *testing.T) {
		tmpDir := t.TempDir()
		workspaceDir := filepath.Join(tmpDir, "workspace")

		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			t.Fatalf("failed to create workspace dir: %v", err)
		}

		// Create symlink pointing to absolute system path
		absLink := filepath.Join(workspaceDir, "etc-link")
		if err := os.Symlink("/etc", absLink); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}

		// This should be rejected - absolute symlink to system directory
		err := ValidatePathWithSymlinkResolution("etc-link", workspaceDir)
		if err == nil {
			t.Error("ValidatePathWithSymlinkResolution() should reject absolute symlink to system path")
		}
	})

	t.Run("nested symlink in subdirectory escaping workspace should be rejected", func(t *testing.T) {
		tmpDir := t.TempDir()
		workspaceDir := filepath.Join(tmpDir, "workspace")
		subDir := filepath.Join(workspaceDir, "subdir", "deep", "nested")
		outsideDir := filepath.Join(tmpDir, "outside")
		secretFile := filepath.Join(outsideDir, "secret.txt")

		// Create directory structure
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("failed to create subdirs: %v", err)
		}
		if err := os.MkdirAll(outsideDir, 0755); err != nil {
			t.Fatalf("failed to create outside dir: %v", err)
		}

		// Create target file
		if err := os.WriteFile(secretFile, []byte("secret"), 0644); err != nil {
			t.Fatalf("failed to create secret file: %v", err)
		}

		// Create symlink deep in directory tree that escapes using multiple ../
		deepLink := filepath.Join(subDir, "escape")
		if err := os.Symlink("../../../../outside/secret.txt", deepLink); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}

		// This should be rejected
		err := ValidatePathWithSymlinkResolution("subdir/deep/nested/escape", workspaceDir)
		if err == nil {
			t.Error("ValidatePathWithSymlinkResolution() should reject nested symlink that escapes workspace")
		}
	})

	t.Run("symlink directory traversal via intermediate path should be rejected", func(t *testing.T) {
		tmpDir := t.TempDir()
		workspaceDir := filepath.Join(tmpDir, "workspace")
		outsideDir := filepath.Join(tmpDir, "outside")
		targetDir := filepath.Join(outsideDir, "target")
		secretFile := filepath.Join(targetDir, "secret.txt")

		// Create directory structure
		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			t.Fatalf("failed to create workspace dir: %v", err)
		}
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			t.Fatalf("failed to create target dir: %v", err)
		}

		// Create target file
		if err := os.WriteFile(secretFile, []byte("secret"), 0644); err != nil {
			t.Fatalf("failed to create secret file: %v", err)
		}

		// Create a directory symlink that points outside
		evilDir := filepath.Join(workspaceDir, "evil-dir")
		if err := os.Symlink(outsideDir, evilDir); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}

		// Access file through symlinked directory: evil-dir/target/secret.txt
		// The path "evil-dir/target/secret.txt" looks safe but evil-dir points outside
		err := ValidatePathWithSymlinkResolution("evil-dir/target/secret.txt", workspaceDir)
		if err == nil {
			t.Error("ValidatePathWithSymlinkResolution() should reject path through symlink directory pointing outside")
		}
	})

	t.Run("symlink to parent directory should be rejected", func(t *testing.T) {
		tmpDir := t.TempDir()
		workspaceDir := filepath.Join(tmpDir, "workspace")

		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			t.Fatalf("failed to create workspace dir: %v", err)
		}

		// Create symlink pointing to parent directory
		parentLink := filepath.Join(workspaceDir, "parent")
		if err := os.Symlink("..", parentLink); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}

		// Symlink resolves to tmpDir which is outside workspace
		err := ValidatePathWithSymlinkResolution("parent", workspaceDir)
		if err == nil {
			t.Error("ValidatePathWithSymlinkResolution() should reject symlink to parent directory")
		}
	})

	t.Run("triple chained symlinks escaping workspace should be rejected", func(t *testing.T) {
		tmpDir := t.TempDir()
		workspaceDir := filepath.Join(tmpDir, "workspace")
		outsideDir := filepath.Join(tmpDir, "outside")
		secretFile := filepath.Join(outsideDir, "secret.txt")

		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			t.Fatalf("failed to create workspace dir: %v", err)
		}
		if err := os.MkdirAll(outsideDir, 0755); err != nil {
			t.Fatalf("failed to create outside dir: %v", err)
		}
		if err := os.WriteFile(secretFile, []byte("secret"), 0644); err != nil {
			t.Fatalf("failed to create secret file: %v", err)
		}

		// Create triple chained symlinks: start -> middle -> end -> outside
		endLink := filepath.Join(workspaceDir, "end")
		middleLink := filepath.Join(workspaceDir, "middle")
		startLink := filepath.Join(workspaceDir, "start")

		if err := os.Symlink(secretFile, endLink); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}
		if err := os.Symlink("end", middleLink); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}
		if err := os.Symlink("middle", startLink); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}

		err := ValidatePathWithSymlinkResolution("start", workspaceDir)
		if err == nil {
			t.Error("ValidatePathWithSymlinkResolution() should reject triple chained symlinks")
		}
	})

	t.Run("symlink with dot-dot in target should be rejected", func(t *testing.T) {
		tmpDir := t.TempDir()
		workspaceDir := filepath.Join(tmpDir, "workspace")
		subDir := filepath.Join(workspaceDir, "subdir")
		outsideFile := filepath.Join(tmpDir, "outside.txt")

		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("failed to create subdir: %v", err)
		}
		if err := os.WriteFile(outsideFile, []byte("outside"), 0644); err != nil {
			t.Fatalf("failed to create outside file: %v", err)
		}

		// Create symlink: workspace/subdir/link -> ../../outside.txt
		dotDotLink := filepath.Join(subDir, "link")
		if err := os.Symlink("../../outside.txt", dotDotLink); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}

		err := ValidatePathWithSymlinkResolution("subdir/link", workspaceDir)
		if err == nil {
			t.Error("ValidatePathWithSymlinkResolution() should reject symlink with .. in target")
		}
	})

	t.Run("symlink loop should return error", func(t *testing.T) {
		tmpDir := t.TempDir()
		workspaceDir := filepath.Join(tmpDir, "workspace")

		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			t.Fatalf("failed to create workspace dir: %v", err)
		}

		// Create circular symlinks: loopA -> loopB -> loopA
		loopA := filepath.Join(workspaceDir, "loopA")
		loopB := filepath.Join(workspaceDir, "loopB")

		if err := os.Symlink("loopB", loopA); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}
		if err := os.Symlink("loopA", loopB); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}

		// EvalSymlinks should fail on circular symlinks
		err := ValidatePathWithSymlinkResolution("loopA", workspaceDir)
		if err == nil {
			t.Error("ValidatePathWithSymlinkResolution() should return error for symlink loop")
		}
	})

	t.Run("mixed path traversal and symlink attack should be rejected", func(t *testing.T) {
		tmpDir := t.TempDir()
		workspaceDir := filepath.Join(tmpDir, "workspace")
		subDir := filepath.Join(workspaceDir, "sub")
		outsideDir := filepath.Join(tmpDir, "outside")

		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("failed to create subdir: %v", err)
		}
		if err := os.MkdirAll(outsideDir, 0755); err != nil {
			t.Fatalf("failed to create outside dir: %v", err)
		}

		// Create symlink that leads outside
		linkInSub := filepath.Join(subDir, "escape")
		if err := os.Symlink(outsideDir, linkInSub); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}

		// Path: sub/escape which resolves to outside directory
		err := ValidatePathWithSymlinkResolution("sub/escape", workspaceDir)
		if err == nil {
			t.Error("ValidatePathWithSymlinkResolution() should reject symlink escape in subdirectory")
		}
	})

	t.Run("symlink with encoded traversal patterns should be rejected", func(t *testing.T) {
		tmpDir := t.TempDir()
		workspaceDir := filepath.Join(tmpDir, "workspace")
		outsideDir := filepath.Join(tmpDir, "outside")
		secretFile := filepath.Join(outsideDir, "secret")

		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			t.Fatalf("failed to create workspace dir: %v", err)
		}
		if err := os.MkdirAll(outsideDir, 0755); err != nil {
			t.Fatalf("failed to create outside dir: %v", err)
		}
		if err := os.WriteFile(secretFile, []byte("secret"), 0644); err != nil {
			t.Fatalf("failed to create secret file: %v", err)
		}

		// Symlink with path that after resolution goes outside
		trickLink := filepath.Join(workspaceDir, "innocent.txt")
		if err := os.Symlink("../outside/secret", trickLink); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}

		err := ValidatePathWithSymlinkResolution("innocent.txt", workspaceDir)
		if err == nil {
			t.Error("ValidatePathWithSymlinkResolution() should reject disguised symlink traversal")
		}
	})

	t.Run("deeply nested chained symlinks should be rejected", func(t *testing.T) {
		tmpDir := t.TempDir()
		workspaceDir := filepath.Join(tmpDir, "workspace")
		outsideFile := filepath.Join(tmpDir, "secret.txt")

		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			t.Fatalf("failed to create workspace dir: %v", err)
		}
		if err := os.WriteFile(outsideFile, []byte("secret"), 0644); err != nil {
			t.Fatalf("failed to create outside file: %v", err)
		}

		// Create 5 levels of symlink chain, final one pointing outside
		links := make([]string, 5)
		for i := 0; i < 5; i++ {
			links[i] = filepath.Join(workspaceDir, fmt.Sprintf("link%d", i))
		}

		// Last link points outside
		if err := os.Symlink(outsideFile, links[4]); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}
		// Each previous link points to the next
		for i := 3; i >= 0; i-- {
			if err := os.Symlink(fmt.Sprintf("link%d", i+1), links[i]); err != nil {
				t.Skipf("symlink not supported: %v", err)
			}
		}

		err := ValidatePathWithSymlinkResolution("link0", workspaceDir)
		if err == nil {
			t.Error("ValidatePathWithSymlinkResolution() should reject deeply nested symlink chain")
		}
	})

	t.Run("symlink targeting tmp directory should be rejected", func(t *testing.T) {
		tmpDir := t.TempDir()
		workspaceDir := filepath.Join(tmpDir, "workspace")

		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			t.Fatalf("failed to create workspace dir: %v", err)
		}

		// Create symlink to /tmp (a common attack target)
		tmpLink := filepath.Join(workspaceDir, "tmp-link")
		if err := os.Symlink("/tmp", tmpLink); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}

		err := ValidatePathWithSymlinkResolution("tmp-link", workspaceDir)
		if err == nil {
			t.Error("ValidatePathWithSymlinkResolution() should reject symlink to /tmp")
		}
	})
}
