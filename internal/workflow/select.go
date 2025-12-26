package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SelectJob selects a job from a workflow by its ID.
// It returns the job if found, or an error if the job ID doesn't exist.
func SelectJob(wf *WorkflowFile, jobID string) (*Job, error) {
	if wf == nil {
		return nil, fmt.Errorf("workflow is nil")
	}
	if jobID == "" {
		return nil, fmt.Errorf("job ID is empty")
	}

	job, exists := wf.Jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job %q not found in workflow", jobID)
	}

	return &job, nil
}

// DiscoverWorkflows finds all workflow files in the .github/workflows directory.
// It returns a slice of absolute paths to workflow files (.yml or .yaml).
func DiscoverWorkflows(repoRoot string) ([]string, error) {
	workflowsDir := filepath.Join(repoRoot, ".github", "workflows")

	info, err := os.Stat(workflowsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("workflows directory does not exist: %w", err)
		}
		return nil, fmt.Errorf("failed to access workflows directory: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("workflows path is not a directory: %s", workflowsDir)
	}

	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflows directory: %w", err)
	}

	var workflows []string
	for _, entry := range entries {
		// Skip directories
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".yml" && ext != ".yaml" {
			continue
		}

		fullPath := filepath.Join(workflowsDir, name)

		// Check if this is a symlink and handle it properly
		fileInfo, err := entry.Info()
		if err != nil {
			// Skip entries we can't get info for
			continue
		}

		// If it's a symlink, resolve it and verify it's a regular file
		if fileInfo.Mode()&os.ModeSymlink != 0 {
			// Resolve the symlink
			resolvedPath, err := filepath.EvalSymlinks(fullPath)
			if err != nil {
				// Skip broken symlinks or symlink loops
				continue
			}

			// Check if the resolved path is a regular file (not a directory)
			resolvedInfo, err := os.Stat(resolvedPath)
			if err != nil {
				// Skip if we can't stat the resolved path
				continue
			}

			if resolvedInfo.IsDir() {
				// Skip symlinks pointing to directories
				continue
			}
		}

		workflows = append(workflows, fullPath)
	}

	return workflows, nil
}
