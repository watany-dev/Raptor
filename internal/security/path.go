package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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

// ValidatePathWithSymlinkResolution validates a path with symlink resolution.
// It ensures that after resolving any symlinks, the final path stays within the workspace.
// This prevents symlink-based path traversal attacks where a symlink inside the workspace
// points to a location outside of it.
func ValidatePathWithSymlinkResolution(path, basePath string) error {
	// First, perform basic validation
	if err := ValidateWorkingDirectory(path, basePath); err != nil {
		return err
	}

	// Construct the full path
	fullPath := filepath.Join(basePath, path)

	// Check if the path exists
	_, err := os.Lstat(fullPath)
	if os.IsNotExist(err) {
		// Path doesn't exist yet, can't check symlinks
		// This is acceptable for paths that will be created
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to stat path: %w", err)
	}

	// Resolve all symlinks to get the real path
	realPath, err := filepath.EvalSymlinks(fullPath)
	if err != nil {
		// If EvalSymlinks fails (e.g., broken symlink), treat it as an error
		return fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	// Resolve basePath symlinks as well for accurate comparison
	realBasePath, err := filepath.EvalSymlinks(basePath)
	if err != nil {
		return fmt.Errorf("failed to resolve base path symlinks: %w", err)
	}

	// Ensure the resolved path is within the workspace
	relPath, err := filepath.Rel(realBasePath, realPath)
	if err != nil {
		return fmt.Errorf(
			"symlink target escapes workspace: resolved path %q cannot be made relative to %q",
			realPath, realBasePath,
		)
	}

	// Check if the relative path escapes the workspace
	if strings.HasPrefix(relPath, "..") {
		return fmt.Errorf(
			"symlink-based path traversal detected: %q resolves to %q which is outside the workspace",
			path, realPath,
		)
	}

	return nil
}
