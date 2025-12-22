package security

import (
	"fmt"
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
