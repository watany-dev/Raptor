package util

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// FindGitRoot finds the root directory of a git repository starting from startDir.
// It returns an error if the directory is not within a git repository.
func FindGitRoot(ctx context.Context, startDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	cmd.Dir = startDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("not a git repository: %s", strings.TrimSpace(stderr.String()))
	}

	return strings.TrimSpace(stdout.String()), nil
}

// GitHeadSHA returns the SHA of the current HEAD commit.
func GitHeadSHA(ctx context.Context, repoRoot string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = repoRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get HEAD SHA: %s", strings.TrimSpace(stderr.String()))
	}

	return strings.TrimSpace(stdout.String()), nil
}

// GitHeadRef returns the ref of the current HEAD (e.g., refs/heads/main).
// Returns empty string if HEAD is detached.
func GitHeadRef(ctx context.Context, repoRoot string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "symbolic-ref", "-q", "HEAD")
	cmd.Dir = repoRoot

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		// Detached HEAD is not an error, just return empty string
		return "", nil
	}

	return strings.TrimSpace(stdout.String()), nil
}
