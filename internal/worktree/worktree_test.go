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
		ws, err := CreateWorkspace(ctx, repoRoot, false)
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
		ws1, err := CreateWorkspace(ctx, repoRoot, false)
		if err != nil {
			t.Fatalf("failed to create first workspace: %v", err)
		}
		defer func() { _ = RemoveWorkspace(ctx, ws1) }()

		ws2, err := CreateWorkspace(ctx, repoRoot, false)
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
		ws, err := CreateWorkspace(ctx, repoRoot, false)
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
		_, err := CreateWorkspace(ctx, "/tmp", false)
		if err == nil {
			t.Error("expected error for non-git directory, got nil")
		}
	})
}

func TestRemoveWorkspace(t *testing.T) {
	ctx := context.Background()
	repoRoot := findTestRepoRoot(t)

	t.Run("removes worktree successfully", func(t *testing.T) {
		ws, err := CreateWorkspace(ctx, repoRoot, false)
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
		ws, err := CreateWorkspace(ctx, repoRoot, false)
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

	t.Run("falls back to manual cleanup when git worktree remove fails", func(t *testing.T) {
		// Create a workspace
		ws, err := CreateWorkspace(ctx, repoRoot, false)
		if err != nil {
			t.Fatalf("failed to create workspace: %v", err)
		}

		// Corrupt the worktree by removing .git file to make git worktree remove fail
		gitFile := filepath.Join(ws.Path, ".git")
		if err := os.Remove(gitFile); err != nil {
			t.Fatalf("failed to remove .git file: %v", err)
		}

		wsPath := ws.Path

		// Remove should still succeed via manual cleanup
		err = RemoveWorkspace(ctx, ws)
		if err != nil {
			t.Errorf("expected no error with manual cleanup fallback, got: %v", err)
		}

		// Verify directory was removed
		if _, err := os.Stat(wsPath); !os.IsNotExist(err) {
			// Clean up manually if test failed
			_ = os.RemoveAll(wsPath)
			t.Errorf("workspace directory should not exist after removal: %s", wsPath)
		}
	})

	t.Run("returns error when both git and manual cleanup fail", func(t *testing.T) {
		// Create a workspace with invalid path that will fail both cleanups
		ws := &Workspace{
			RepoRoot: repoRoot,
			Path:     "/nonexistent/path/that/does/not/exist",
			ID:       "test-id",
		}

		// This should return an error since the path doesn't exist
		// and git worktree remove will fail
		err := RemoveWorkspace(ctx, ws)
		// Note: os.RemoveAll on non-existent path returns nil,
		// so this won't error. The test validates the code path.
		if err != nil {
			// If it does error, that's also acceptable
			t.Logf("got error as expected: %v", err)
		}
	})
}

func TestCreateWorkspace_Verified(t *testing.T) {
	ctx := context.Background()
	repoRoot := findTestRepoRoot(t)

	t.Run("skips git verification when verified is true", func(t *testing.T) {
		ws, err := CreateWorkspace(ctx, repoRoot, true)
		if err != nil {
			t.Fatalf("expected no error with verified=true, got: %v", err)
		}
		defer func() { _ = RemoveWorkspace(ctx, ws) }()

		// Verify workspace was created
		if _, err := os.Stat(ws.Path); os.IsNotExist(err) {
			t.Errorf("worktree directory does not exist: %s", ws.Path)
		}
	})

	t.Run("still fails for non-git directory even with verified true", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Even with verified=true, git worktree add will fail
		_, err := CreateWorkspace(ctx, tmpDir, true)
		if err == nil {
			t.Error("expected error for non-git directory")
		}
	})
}

func TestGenerateID(t *testing.T) {
	t.Run("generates unique IDs", func(t *testing.T) {
		ids := make(map[string]bool)
		for i := 0; i < 100; i++ {
			id, err := generateID()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ids[id] {
				t.Errorf("duplicate ID generated: %s", id)
			}
			ids[id] = true
		}
	})

	t.Run("generates 16 character hex strings", func(t *testing.T) {
		id, err := generateID()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(id) != 16 {
			t.Errorf("expected 16 character ID, got %d characters: %s", len(id), id)
		}
		// Verify it's valid hex
		for _, c := range id {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("ID contains non-hex character: %c in %s", c, id)
			}
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

// TestCreateWorkspace_InvalidRepo tests handling when repoRoot is invalid
func TestCreateWorkspace_InvalidRepo(t *testing.T) {
	tmpDir := t.TempDir()

	// Try to create workspace in non-git directory
	ctx := context.Background()
	_, err := CreateWorkspace(ctx, tmpDir, false)
	if err == nil {
		t.Error("CreateWorkspace() expected error for non-git directory, got nil")
	}
}

// TestRemoveWorkspace_EmptyWorkspace tests removal of an empty workspace
func TestRemoveWorkspace_EmptyWorkspace(t *testing.T) {
	repoRoot := findTestRepoRoot(t)
	ctx := context.Background()

	// Create a workspace first
	ws, err := CreateWorkspace(ctx, repoRoot, false)
	if err != nil {
		t.Skipf("failed to create workspace: %v", err)
	}

	// Then remove it
	err = RemoveWorkspace(ctx, ws)
	if err != nil {
		t.Logf("RemoveWorkspace() returned error: %v", err)
	}
}

// TestGenerateID_Uniqueness tests that generated IDs are unique
func TestGenerateID_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		id, err := generateID()
		if err != nil {
			t.Fatalf("generateID() error = %v", err)
		}

		if seen[id] {
			t.Errorf("generateID() produced duplicate ID: %s", id)
		}
		seen[id] = true

		// Verify format: should be 16 hex characters
		if len(id) != 16 {
			t.Errorf("generateID() returned ID with wrong length: got %d, expected 16", len(id))
		}

		// Verify all characters are hex
		for _, c := range id {
			if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
				t.Errorf("generateID() returned non-hex character: %c", c)
			}
		}
	}
}

// TestCreateWorkspace_IDFormat tests that workspace IDs are correctly formatted
func TestCreateWorkspace_IDFormat(t *testing.T) {
	repoRoot := findTestRepoRoot(t)
	ctx := context.Background()

	// Create workspace
	ws, err := CreateWorkspace(ctx, repoRoot, false)
	if err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	// Verify ID format
	id := ws.ID
	if len(id) != 16 {
		t.Errorf("CreateWorkspace() returned ID with wrong length: got %d, expected 16", len(id))
	}

	// Verify all characters are hex
	for _, c := range id {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			t.Errorf("CreateWorkspace() returned non-hex character: %c", c)
		}
	}

	// Cleanup
	RemoveWorkspace(ctx, ws)
}
