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

	// Extract job order from YAML structure
	wf.JobOrder, err = extractJobOrder(data)
	if err != nil {
		return nil, fmt.Errorf("failed to extract job order: %w", err)
	}

	return &wf, nil
}

// extractJobOrder parses the YAML to get job IDs in definition order.
func extractJobOrder(data []byte) ([]string, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, err
	}

	// root.Content[0] is the document node
	if len(root.Content) == 0 {
		return nil, nil
	}

	doc := root.Content[0]
	if doc.Kind != yaml.MappingNode {
		return nil, nil
	}

	// Find the "jobs" key in the mapping
	for i := 0; i < len(doc.Content)-1; i += 2 {
		keyNode := doc.Content[i]
		valueNode := doc.Content[i+1]

		if keyNode.Value == "jobs" && valueNode.Kind == yaml.MappingNode {
			// Extract job IDs in order
			var jobOrder []string
			for j := 0; j < len(valueNode.Content)-1; j += 2 {
				jobKeyNode := valueNode.Content[j]
				jobOrder = append(jobOrder, jobKeyNode.Value)
			}
			return jobOrder, nil
		}
	}

	return nil, nil
}
