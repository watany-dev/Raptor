package security

import (
	"fmt"
	"path/filepath"
	"strings"
)

// DangerousPathPrefixes contains system paths that should never be added to PATH.
// Adding these could allow malicious binaries to shadow system commands.
var DangerousPathPrefixes = []string{
	"/tmp",
	"/var/tmp",
	"/dev/shm",
	"/run",
	"/proc",
	"/sys",
}

// ValidateGitHubPath validates a path entry for GITHUB_PATH.
// It ensures the path is safe to add to the system PATH.
// workspacePath is the isolated worktree path where execution occurs.
func ValidateGitHubPath(pathEntry, workspacePath string) error {
	if strings.TrimSpace(pathEntry) == "" {
		return fmt.Errorf("GITHUB_PATH entry cannot be empty")
	}

	// Null byte check
	if strings.Contains(pathEntry, "\x00") {
		return fmt.Errorf("GITHUB_PATH entry contains null bytes")
	}

	// Normalize the path
	cleanPath := filepath.Clean(pathEntry)

	// Check for dangerous system paths
	for _, dangerous := range DangerousPathPrefixes {
		if strings.HasPrefix(cleanPath, dangerous+"/") || cleanPath == dangerous {
			return fmt.Errorf(
				"GITHUB_PATH entry %q is blocked for security: "+
					"adding paths under %s could allow command shadowing attacks",
				pathEntry, dangerous,
			)
		}
	}

	// If workspace is provided, validate path is within workspace
	if workspacePath != "" {
		// For absolute paths, check if they're within workspace
		if filepath.IsAbs(cleanPath) {
			relPath, err := filepath.Rel(workspacePath, cleanPath)
			if err != nil || strings.HasPrefix(relPath, "..") {
				return fmt.Errorf(
					"GITHUB_PATH entry %q is outside the workspace: "+
						"for security, paths must be within the workspace directory",
					pathEntry,
				)
			}
		}
	}

	return nil
}

// ValidateWorkingDirectory validates that a working directory is safe.
// It ensures:
// - No absolute paths
// - No path traversal outside the workspace
func ValidateWorkingDirectory(workDir, basePath string) error {
	if workDir == "" {
		return nil // Empty is allowed (uses default)
	}

	// 1. Block absolute paths
	if filepath.IsAbs(workDir) {
		return fmt.Errorf(
			"absolute paths are not allowed in working-directory: %q\n"+
				"Use relative paths from the repository root instead",
			workDir,
		)
	}

	// 2. Path traversal check
	cleanPath := filepath.Clean(workDir)
	if strings.HasPrefix(cleanPath, "..") {
		return fmt.Errorf(
			"working-directory cannot traverse outside the workspace: %q",
			workDir,
		)
	}

	// 3. Calculate normalized path
	fullPath := filepath.Join(basePath, cleanPath)

	// 4. Ensure path stays within basePath
	relPath, err := filepath.Rel(basePath, fullPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return fmt.Errorf(
			"working-directory must be within the workspace: %q",
			workDir,
		)
	}

	return nil
}
