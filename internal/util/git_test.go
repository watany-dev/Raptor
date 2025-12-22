package util

import (
	"context"
	"os"
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
