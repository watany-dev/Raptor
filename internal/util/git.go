package util

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// GitRepoInfo contains information about a git repository.
// This struct is populated by a single git command to minimize subprocess overhead.
type GitRepoInfo struct {
	// Root is the absolute path to the repository root directory.
	Root string
	// HeadSHA is the SHA of the current HEAD commit.
	HeadSHA string
	// HeadRef is the symbolic ref of HEAD (e.g., refs/heads/main).
	// Empty string if HEAD is detached.
	HeadRef string
}

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

// GetGitRepoInfo retrieves git repository information with a single git command.
// This is more efficient than calling FindGitRoot, GitHeadSHA, and GitHeadRef separately,
// as it reduces the number of subprocess invocations from 3 to 2.
func GetGitRepoInfo(ctx context.Context, startDir string) (*GitRepoInfo, error) {
	// Get root and HEAD SHA in a single command
	// git rev-parse outputs each value on a separate line
	out, err := runGit(ctx, startDir, "rev-parse", "--show-toplevel", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("not a git repository: %w", err)
	}

	lines := strings.Split(out, "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("unexpected git rev-parse output: %s", out)
	}

	info := &GitRepoInfo{
		Root:    lines[0],
		HeadSHA: lines[1],
	}

	// Get symbolic ref separately (it's a different git command)
	// Use -q to suppress errors for detached HEAD
	ref, err := runGit(ctx, startDir, "symbolic-ref", "-q", "HEAD")
	if err != nil {
		// git symbolic-ref -q exits with code 1 for detached HEAD (not an error)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			info.HeadRef = ""
		} else {
			// For other errors, we still return the info we have
			// HeadRef is optional, so we don't fail the whole operation
			info.HeadRef = ""
		}
	} else {
		info.HeadRef = ref
	}

	return info, nil
}
