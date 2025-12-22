package workflow

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadWorkflowFile loads and parses a GitHub Actions workflow file from the given path.
func LoadWorkflowFile(path string) (*WorkflowFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}

	var wf WorkflowFile
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return nil, fmt.Errorf("failed to parse workflow YAML: %w", err)
	}

	return &wf, nil
}
