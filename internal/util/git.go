package util

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// runGit executes a git command and returns the trimmed stdout.
func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %w", strings.TrimSpace(stderr.String()), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// FindGitRoot finds the root directory of a git repository starting from startDir.
func FindGitRoot(ctx context.Context, startDir string) (string, error) {
	out, err := runGit(ctx, startDir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("not a git repository: %w", err)
	}
	return out, nil
}

// GitHeadSHA returns the SHA of the current HEAD commit.
func GitHeadSHA(ctx context.Context, repoRoot string) (string, error) {
	out, err := runGit(ctx, repoRoot, "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD SHA: %w", err)
	}
	return out, nil
}

// GitHeadRef returns the ref of the current HEAD (e.g., refs/heads/main).
// Returns empty string if HEAD is detached.
func GitHeadRef(ctx context.Context, repoRoot string) (string, error) {
	out, err := runGit(ctx, repoRoot, "symbolic-ref", "-q", "HEAD")
	if err != nil {
		// git symbolic-ref -q exits with code 1 for detached HEAD (not an error)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return "", nil
		}
		return "", fmt.Errorf("failed to get HEAD ref: %w", err)
	}
	return out, nil
}
