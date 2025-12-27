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

	// Parse YAML into yaml.Node once (instead of parsing twice)
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("failed to parse workflow YAML: %w", err)
	}

	// Extract both WorkflowFile and JobOrder from the single parsed Node
	wf, jobOrder, err := parseWorkflowFromNode(&root)
	if err != nil {
		return nil, fmt.Errorf("failed to parse workflow: %w", err)
	}
	wf.JobOrder = jobOrder

	return wf, nil
}

// parseWorkflowFromNode extracts WorkflowFile and job order from a parsed yaml.Node.
// This avoids parsing the YAML twice by reusing the same Node for both operations.
func parseWorkflowFromNode(root *yaml.Node) (*WorkflowFile, []string, error) {
	var wf WorkflowFile
	// Decode the Node directly into the struct (no re-parsing needed)
	if err := root.Decode(&wf); err != nil {
		return nil, nil, fmt.Errorf("failed to decode workflow: %w", err)
	}

	// Extract job order from the same Node
	jobOrder := extractJobOrderFromNode(root)

	return &wf, jobOrder, nil
}

// extractJobOrderFromNode extracts job IDs in definition order from a parsed yaml.Node.
func extractJobOrderFromNode(root *yaml.Node) []string {
	if root == nil || len(root.Content) == 0 {
		return nil
	}

	// root.Content[0] is the document node
	doc := root.Content[0]
	if doc == nil || doc.Kind != yaml.MappingNode {
		return nil
	}

	// Find the "jobs" key in the mapping
	for i := 0; i < len(doc.Content)-1; i += 2 {
		keyNode := doc.Content[i]
		valueNode := doc.Content[i+1]

		if keyNode == nil || valueNode == nil {
			continue
		}

		if keyNode.Value == "jobs" && valueNode.Kind == yaml.MappingNode {
			// Extract job IDs in order
			var jobOrder []string
			for j := 0; j < len(valueNode.Content)-1; j += 2 {
				jobKeyNode := valueNode.Content[j]
				if jobKeyNode == nil {
					continue
				}
				jobOrder = append(jobOrder, jobKeyNode.Value)
			}
			return jobOrder
		}
	}

	return nil
}
