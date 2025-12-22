package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DiscoverWorkflows finds all workflow files in the .github/workflows directory.
// It returns a slice of absolute paths to workflow files (.yml or .yaml).
func DiscoverWorkflows(repoRoot string) ([]string, error) {
	workflowsDir := filepath.Join(repoRoot, ".github", "workflows")

	info, err := os.Stat(workflowsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("workflows directory does not exist: %s", workflowsDir)
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
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext == ".yml" || ext == ".yaml" {
			workflows = append(workflows, filepath.Join(workflowsDir, name))
		}
	}

	return workflows, nil
}
