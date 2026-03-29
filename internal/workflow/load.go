package workflow

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadWorkflowFile loads and parses a GitHub Actions workflow file from the given path.
// The parsing is optimized to extract both the workflow structure and job order
// in a single pass through the YAML node tree.
func LoadWorkflowFile(path string) (*WorkflowFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}

	// Parse YAML and decode in a single operation.
	// WorkflowFile.UnmarshalYAML handles both struct decoding and JobOrder extraction
	// in a single node traversal, avoiding the overhead of double traversal.
	var wf WorkflowFile
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return nil, fmt.Errorf("failed to parse workflow YAML: %w", err)
	}

	return &wf, nil
}
