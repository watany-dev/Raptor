package worktree

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateWorkspace(t *testing.T) {
	ctx := context.Background()

	// Find the repository root for testing
	repoRoot := findTestRepoRoot(t)

	t.Run("creates new worktree successfully", func(t *testing.T) {
		ws, err := CreateWorkspace(ctx, repoRoot)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		defer func() {
			// Cleanup: remove the worktree
			if err := RemoveWorkspace(ctx, ws); err != nil {
				t.Logf("failed to cleanup workspace: %v", err)
			}
		}()

		// Verify worktree directory exists
		if _, err := os.Stat(ws.Path); os.IsNotExist(err) {
			t.Errorf("worktree directory does not exist: %s", ws.Path)
		}

		// Verify it's a valid git worktree
		gitDir := filepath.Join(ws.Path, ".git")
		if _, err := os.Stat(gitDir); os.IsNotExist(err) {
			t.Errorf(".git should exist in worktree: %s", gitDir)
		}

		// Verify RepoRoot is set correctly
		if ws.RepoRoot != repoRoot {
			t.Errorf("expected RepoRoot %s, got %s", repoRoot, ws.RepoRoot)
		}
	})

	t.Run("workspace ID is unique", func(t *testing.T) {
		ws1, err := CreateWorkspace(ctx, repoRoot)
		if err != nil {
			t.Fatalf("failed to create first workspace: %v", err)
		}
		defer func() { _ = RemoveWorkspace(ctx, ws1) }()

		ws2, err := CreateWorkspace(ctx, repoRoot)
		if err != nil {
			t.Fatalf("failed to create second workspace: %v", err)
		}
		defer func() { _ = RemoveWorkspace(ctx, ws2) }()

		if ws1.ID == ws2.ID {
			t.Errorf("workspace IDs should be unique, both got: %s", ws1.ID)
		}

		if ws1.Path == ws2.Path {
			t.Errorf("workspace paths should be unique, both got: %s", ws1.Path)
		}
	})

	t.Run("worktree is created in correct location", func(t *testing.T) {
		ws, err := CreateWorkspace(ctx, repoRoot)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		defer func() { _ = RemoveWorkspace(ctx, ws) }()

		// Verify path is under .raptor directory
		expectedPrefix := filepath.Join(repoRoot, ".raptor", "ws-")
		if !strings.HasPrefix(ws.Path, expectedPrefix) {
			t.Errorf("expected path to start with %s, got %s", expectedPrefix, ws.Path)
		}

		// Verify ID is in the path
		if !strings.Contains(ws.Path, ws.ID) {
			t.Errorf("expected path to contain ID %s, got %s", ws.ID, ws.Path)
		}
	})

	t.Run("returns error for non-git directory", func(t *testing.T) {
		_, err := CreateWorkspace(ctx, "/tmp")
		if err == nil {
			t.Error("expected error for non-git directory, got nil")
		}
	})
}

func TestRemoveWorkspace(t *testing.T) {
	ctx := context.Background()
	repoRoot := findTestRepoRoot(t)

	t.Run("removes worktree successfully", func(t *testing.T) {
		ws, err := CreateWorkspace(ctx, repoRoot)
		if err != nil {
			t.Fatalf("failed to create workspace: %v", err)
		}

		// Verify workspace exists before removal
		if _, err := os.Stat(ws.Path); os.IsNotExist(err) {
			t.Fatalf("workspace should exist before removal: %s", ws.Path)
		}

		// Remove the workspace
		err = RemoveWorkspace(ctx, ws)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})

	t.Run("directory does not exist after removal", func(t *testing.T) {
		ws, err := CreateWorkspace(ctx, repoRoot)
		if err != nil {
			t.Fatalf("failed to create workspace: %v", err)
		}

		wsPath := ws.Path

		// Remove the workspace
		err = RemoveWorkspace(ctx, ws)
		if err != nil {
			t.Fatalf("failed to remove workspace: %v", err)
		}

		// Verify directory no longer exists
		if _, err := os.Stat(wsPath); !os.IsNotExist(err) {
			t.Errorf("workspace directory should not exist after removal: %s", wsPath)
		}
	})

	t.Run("handles nil workspace gracefully", func(t *testing.T) {
		err := RemoveWorkspace(ctx, nil)
		if err != nil {
			t.Errorf("expected no error for nil workspace, got: %v", err)
		}
	})
}

// findTestRepoRoot finds the repository root by looking for .git directory
func findTestRepoRoot(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	dir := cwd
	for {
		gitPath := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitPath); err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find git root from %s", cwd)
		}
		dir = parent
	}
}
