package worktree

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	// raptorDir is the directory where Raptor stores its workspaces.
	raptorDir = ".raptor"
	// wsPrefix is the prefix for workspace directories.
	wsPrefix = "ws-"
)

// CreateWorkspace creates a new isolated git worktree for running workflows.
// The worktree is created under .raptor/ws-<id> in the repository root.
// If verified is true, skips git repository verification (use when caller has already verified).
func CreateWorkspace(ctx context.Context, repoRoot string, verified bool) (*Workspace, error) {
	// Verify this is a git repository (skip if already verified by caller)
	if !verified {
		if err := verifyGitRepo(ctx, repoRoot); err != nil {
			return nil, err
		}
	}

	// Generate unique workspace ID
	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate workspace ID: %w", err)
	}

	// Create .raptor directory if it doesn't exist
	raptorPath := filepath.Join(repoRoot, raptorDir)
	if err := os.MkdirAll(raptorPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create raptor directory: %w", err)
	}

	// Create worktree path
	wsPath := filepath.Join(raptorPath, wsPrefix+id)

	// Create the worktree using git worktree add --detach
	cmd := exec.CommandContext(ctx, "git", "worktree", "add", "--detach", wsPath)
	cmd.Dir = repoRoot

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to create worktree: %s", strings.TrimSpace(stderr.String()))
	}

	return &Workspace{
		RepoRoot: repoRoot,
		Path:     wsPath,
		ID:       id,
	}, nil
}

// RemoveWorkspace removes a workspace and cleans up the git worktree.
func RemoveWorkspace(ctx context.Context, ws *Workspace) error {
	if ws == nil {
		return nil
	}

	// Remove the worktree using git worktree remove
	cmd := exec.CommandContext(ctx, "git", "worktree", "remove", "--force", ws.Path)
	cmd.Dir = ws.RepoRoot

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// If worktree remove fails, try to clean up manually
		if removeErr := os.RemoveAll(ws.Path); removeErr != nil {
			return fmt.Errorf("failed to remove worktree: %s (manual cleanup also failed: %w)",
				strings.TrimSpace(stderr.String()), removeErr)
		}
		// Prune the worktree reference
		pruneCmd := exec.CommandContext(ctx, "git", "worktree", "prune")
		pruneCmd.Dir = ws.RepoRoot
		_ = pruneCmd.Run() // Ignore prune errors
	}

	return nil
}

// verifyGitRepo checks if the given path is a valid git repository.
func verifyGitRepo(ctx context.Context, path string) error {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--git-dir")
	cmd.Dir = path

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("not a git repository: %s", strings.TrimSpace(stderr.String()))
	}

	return nil
}

// randReader is the random reader used for generating IDs.
// It can be replaced in tests to simulate errors.
var randReader = rand.Reader

// generateID generates a unique identifier for a workspace.
func generateID() (string, error) {
	b := make([]byte, 8)
	if _, err := randReader.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
