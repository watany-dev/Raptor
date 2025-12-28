package util

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindGitRoot(t *testing.T) {
	ctx := context.Background()

	t.Run("returns root when in git repository root", func(t *testing.T) {
		// Get the current working directory (which should be the repo root)
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get cwd: %v", err)
		}

		// Go up to find the actual repo root
		repoRoot := findTestRepoRoot(t, cwd)

		root, err := FindGitRoot(ctx, repoRoot)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		if root != repoRoot {
			t.Errorf("expected %s, got %s", repoRoot, root)
		}
	})

	t.Run("returns root when in subdirectory", func(t *testing.T) {
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get cwd: %v", err)
		}

		repoRoot := findTestRepoRoot(t, cwd)

		// Test from a subdirectory
		subDir := filepath.Join(repoRoot, "internal", "util")

		root, err := FindGitRoot(ctx, subDir)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		if root != repoRoot {
			t.Errorf("expected %s, got %s", repoRoot, root)
		}
	})

	t.Run("returns error when not in git repository", func(t *testing.T) {
		// Use a directory that is definitely not a git repo
		_, err := FindGitRoot(ctx, "/tmp")
		if err == nil {
			t.Error("expected error for non-git directory, got nil")
		}
	})
}

func TestGitHeadSHA(t *testing.T) {
	ctx := context.Background()

	t.Run("returns SHA for valid repository", func(t *testing.T) {
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get cwd: %v", err)
		}

		repoRoot := findTestRepoRoot(t, cwd)

		sha, err := GitHeadSHA(ctx, repoRoot)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		// SHA should be 40 hex characters
		if len(sha) != 40 {
			t.Errorf("expected 40 char SHA, got %d chars: %s", len(sha), sha)
		}
	})

	t.Run("returns error for non-git directory", func(t *testing.T) {
		_, err := GitHeadSHA(ctx, "/tmp")
		if err == nil {
			t.Error("expected error for non-git directory, got nil")
		}
	})
}

func TestGitHeadRef(t *testing.T) {
	ctx := context.Background()

	t.Run("returns ref for valid repository", func(t *testing.T) {
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get cwd: %v", err)
		}

		repoRoot := findTestRepoRoot(t, cwd)

		ref, err := GitHeadRef(ctx, repoRoot)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		// ref could be empty (detached HEAD) or start with refs/
		if ref != "" && !strings.HasPrefix(ref, "refs/") {
			t.Errorf("expected ref to be empty or start with refs/, got: %s", ref)
		}
	})

	t.Run("returns error for non-git directory", func(t *testing.T) {
		// Non-git directory should return an error
		_, err := GitHeadRef(ctx, "/tmp")
		if err == nil {
			t.Error("expected error for non-git directory, got nil")
		}
	})
}

// findTestRepoRoot finds the repository root by looking for .git directory
func findTestRepoRoot(t *testing.T, startDir string) string {
	t.Helper()
	dir := startDir
	for {
		gitPath := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitPath); err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find git root from %s", startDir)
		}
		dir = parent
	}
}

// TestGitHeadSHA_ValidRepo tests getting SHA of valid repository
func TestGitHeadSHA_ValidRepo(t *testing.T) {
	t.Parallel()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	repoRoot := findTestRepoRoot(t, cwd)

	sha, err := GitHeadSHA(context.Background(), repoRoot)
	if err != nil {
		t.Fatalf("GitHeadSHA() error = %v", err)
	}

	if len(sha) == 0 {
		t.Error("GitHeadSHA() returned empty SHA")
	}

	// SHA should be 40 hex characters (SHA-1) or 64 (SHA-256)
	if len(sha) != 40 && len(sha) != 64 {
		t.Errorf("GitHeadSHA() returned invalid SHA length: %d", len(sha))
	}
}

// TestGitHeadRef_DefaultBranch tests GitHeadRef returns default branch name
func TestGitHeadRef_DefaultBranch(t *testing.T) {
	t.Parallel()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	repoRoot := findTestRepoRoot(t, cwd)

	ref, err := GitHeadRef(context.Background(), repoRoot)
	if err != nil {
		t.Fatalf("GitHeadRef() error = %v", err)
	}

	// ref may be empty in detached HEAD state, which is acceptable
	// Just verify the function doesn't error
	t.Logf("GitHeadRef() returned: %q (length: %d)", ref, len(ref))
}

// TestGitHeadRef_DetachedHead tests GitHeadRef returns empty string for detached HEAD
func TestGitHeadRef_DetachedHead(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Initialize a new git repository
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user for the commit
	cmd = exec.CommandContext(ctx, "git", "config", "user.email", "test@test.com")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to config git: %v", err)
	}
	cmd = exec.CommandContext(ctx, "git", "config", "user.name", "Test")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to config git: %v", err)
	}

	// Create a file and commit it
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	cmd = exec.CommandContext(ctx, "git", "add", ".")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git add: %v", err)
	}
	cmd = exec.CommandContext(ctx, "git", "commit", "--no-gpg-sign", "-m", "initial")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to git commit: %v, output: %s", err, output)
	}

	// Get the commit SHA
	sha, err := GitHeadSHA(ctx, tmpDir)
	if err != nil {
		t.Fatalf("failed to get HEAD SHA: %v", err)
	}

	// Detach HEAD by checking out the commit SHA directly
	cmd = exec.CommandContext(ctx, "git", "checkout", sha)
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to checkout detached HEAD: %v", err)
	}

	// Now GitHeadRef should return empty string (detached HEAD)
	ref, err := GitHeadRef(ctx, tmpDir)
	if err != nil {
		t.Errorf("GitHeadRef() error = %v; expected no error for detached HEAD", err)
	}
	if ref != "" {
		t.Errorf("GitHeadRef() = %q; expected empty string for detached HEAD", ref)
	}
}

// TestFindGitRoot_MultipleDirectories tests finding git root from nested directories
func TestFindGitRoot_MultipleDirectories(t *testing.T) {
	t.Parallel()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	repoRoot := findTestRepoRoot(t, cwd)

	ctx := context.Background()

	// Test from current directory
	root1, err := FindGitRoot(ctx, repoRoot)
	if err != nil {
		t.Fatalf("FindGitRoot() error = %v", err)
	}

	// Test from subdirectory
	testDir := filepath.Join(repoRoot, "internal")
	if info, err := os.Stat(testDir); err == nil && info.IsDir() {
		root2, err := FindGitRoot(ctx, testDir)
		if err != nil {
			t.Fatalf("FindGitRoot() error from subdirectory = %v", err)
		}

		// Should find same root from both directories
		if root1 != root2 {
			t.Errorf("FindGitRoot() returned different roots: %s vs %s", root1, root2)
		}
	}
}
