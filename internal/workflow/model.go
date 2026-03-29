package workflow

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// StringOrSlice handles both single string and string array YAML values.
// This is used for fields like "needs" which can be either:
//   - needs: build
//   - needs: [build, test]
type StringOrSlice []string

// UnmarshalYAML implements custom YAML unmarshaling for StringOrSlice.
func (s *StringOrSlice) UnmarshalYAML(value *yaml.Node) error {
	// Single string case: needs: build
	if value.Kind == yaml.ScalarNode {
		*s = []string{value.Value}
		return nil
	}

	// Array case: needs: [build, test]
	if value.Kind == yaml.SequenceNode {
		result := make([]string, 0, len(value.Content))
		for _, item := range value.Content {
			if item != nil && item.Kind == yaml.ScalarNode {
				result = append(result, item.Value)
			}
		}
		*s = result
		return nil
	}

	// Fallback for unexpected types
	var slice []string
	if err := value.Decode(&slice); err != nil {
		return err
	}
	*s = slice
	return nil
}

// WorkflowFile represents a GitHub Actions workflow file.
type WorkflowFile struct {
	// Name is the name of the workflow.
	Name string `yaml:"name"`
	// Env contains environment variables available to all jobs.
	Env map[string]string `yaml:"env"`
	// Jobs contains the jobs defined in the workflow.
	Jobs map[string]Job `yaml:"jobs"`
	// JobOrder contains job IDs in the order they are defined in the YAML file.
	JobOrder []string `yaml:"-"`
}

// UnmarshalYAML implements custom YAML unmarshaling for WorkflowFile.
// This extracts both the struct fields and JobOrder in a single node traversal,
// avoiding the overhead of traversing the YAML node tree twice.
func (w *WorkflowFile) UnmarshalYAML(node *yaml.Node) error {
	// Handle document node wrapper
	if node.Kind == yaml.DocumentNode {
		if len(node.Content) == 0 {
			return nil
		}
		node = node.Content[0]
	}

	if node.Kind != yaml.MappingNode {
		// Use default decoder for non-mapping nodes (will likely error)
		type rawWorkflowFile WorkflowFile
		return node.Decode((*rawWorkflowFile)(w))
	}

	// Single-pass extraction: decode fields and extract job order simultaneously
	for i := 0; i < len(node.Content)-1; i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		if keyNode == nil || valueNode == nil {
			continue
		}

		switch keyNode.Value {
		case "name":
			w.Name = valueNode.Value
		case "env":
			if valueNode.Kind == yaml.MappingNode {
				w.Env = make(map[string]string, len(valueNode.Content)/2)
				for j := 0; j < len(valueNode.Content)-1; j += 2 {
					envKey := valueNode.Content[j]
					envVal := valueNode.Content[j+1]
					if envKey != nil && envVal != nil {
						w.Env[envKey.Value] = envVal.Value
					}
				}
			}
		case "jobs":
			// Jobs must be a mapping node (not sequence or scalar)
			if valueNode.Kind == yaml.SequenceNode {
				// Return error for invalid jobs format (sequence instead of mapping)
				return fmt.Errorf("cannot unmarshal !!seq into map[string]workflow.Job")
			}
			if valueNode.Kind == yaml.MappingNode {
				// Pre-allocate maps with exact capacity
				jobCount := len(valueNode.Content) / 2
				w.Jobs = make(map[string]Job, jobCount)
				w.JobOrder = make([]string, 0, jobCount)

				// Extract jobs and job order in single pass
				for j := 0; j < len(valueNode.Content)-1; j += 2 {
					jobKeyNode := valueNode.Content[j]
					jobValueNode := valueNode.Content[j+1]

					if jobKeyNode == nil {
						continue
					}

					jobID := jobKeyNode.Value
					w.JobOrder = append(w.JobOrder, jobID)

					if jobValueNode != nil {
						var job Job
						if err := job.UnmarshalYAML(jobValueNode); err != nil {
							return err
						}
						w.Jobs[jobID] = job
					}
				}
			}
		}
	}

	return nil
}

// Job represents a job in a workflow.
type Job struct {
	// Name is the display name of the job.
	Name string `yaml:"name"`
	// RunsOn specifies the runner environment.
	RunsOn string `yaml:"runs-on"`
	// Needs specifies job dependencies. Jobs listed here must complete successfully
	// before this job runs. Can be a single job ID or a list of job IDs.
	Needs StringOrSlice `yaml:"needs"`
	// Env contains environment variables available to all steps in the job.
	Env map[string]string `yaml:"env"`
	// Steps contains the steps to execute in the job.
	Steps []Step `yaml:"steps"`
}

// UnmarshalYAML implements custom YAML unmarshaling for Job.
// This avoids reflection overhead by directly parsing YAML nodes.
func (j *Job) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		type rawJob Job
		return node.Decode((*rawJob)(j))
	}

	for i := 0; i < len(node.Content)-1; i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		if keyNode == nil || valueNode == nil {
			continue
		}

		switch keyNode.Value {
		case "name":
			j.Name = valueNode.Value
		case "runs-on":
			j.RunsOn = valueNode.Value
		case "needs":
			// Handle needs directly to avoid Decode overhead
			if valueNode.Kind == yaml.ScalarNode {
				j.Needs = []string{valueNode.Value}
			} else if valueNode.Kind == yaml.SequenceNode {
				j.Needs = make([]string, 0, len(valueNode.Content))
				for _, item := range valueNode.Content {
					if item != nil && item.Kind == yaml.ScalarNode {
						j.Needs = append(j.Needs, item.Value)
					}
				}
			} else if valueNode.Kind == yaml.MappingNode {
				// needs must be a string or array, not a mapping
				return fmt.Errorf("cannot unmarshal !!map into []string")
			}
		case "env":
			if valueNode.Kind == yaml.MappingNode {
				j.Env = make(map[string]string, len(valueNode.Content)/2)
				for k := 0; k < len(valueNode.Content)-1; k += 2 {
					envKey := valueNode.Content[k]
					envVal := valueNode.Content[k+1]
					if envKey != nil && envVal != nil {
						j.Env[envKey.Value] = envVal.Value
					}
				}
			}
		case "steps":
			if valueNode.Kind == yaml.SequenceNode {
				j.Steps = make([]Step, 0, len(valueNode.Content))
				for _, stepNode := range valueNode.Content {
					if stepNode == nil {
						continue
					}
					var step Step
					if err := step.UnmarshalYAML(stepNode); err != nil {
						return err
					}
					j.Steps = append(j.Steps, step)
				}
			}
		}
	}

	return nil
}

// Step represents a step in a job.
type Step struct {
	// ID is an optional unique identifier for the step.
	ID string `yaml:"id"`
	// Name is an optional display name for the step.
	Name string `yaml:"name"`
	// If contains the condition expression for this step.
	// If the condition evaluates to false, the step is skipped.
	If string `yaml:"if"`
	// Run contains the shell command to execute.
	Run string `yaml:"run"`
	// Uses specifies a GitHub Action to use.
	// NOTE: This field is parsed but not executed. Steps with uses will be skipped
	// with an explicit message. This provides better UX than silently ignoring.
	Uses string `yaml:"uses"`
	// With contains input parameters for the action specified in Uses.
	// NOTE: This field is parsed but not executed (see Uses note above).
	With map[string]string `yaml:"with"`
	// Env contains environment variables for this step.
	Env map[string]string `yaml:"env"`
	// WorkingDirectory specifies the working directory for run commands.
	WorkingDirectory string `yaml:"working-directory"`
}

// UnmarshalYAML implements custom YAML unmarshaling for Step.
// This avoids reflection overhead by directly parsing YAML nodes.
func (s *Step) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		type rawStep Step
		return node.Decode((*rawStep)(s))
	}

	for i := 0; i < len(node.Content)-1; i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		if keyNode == nil || valueNode == nil {
			continue
		}

		switch keyNode.Value {
		case "id":
			s.ID = valueNode.Value
		case "name":
			s.Name = valueNode.Value
		case "if":
			s.If = valueNode.Value
		case "run":
			s.Run = valueNode.Value
		case "uses":
			s.Uses = valueNode.Value
		case "with":
			if valueNode.Kind == yaml.MappingNode {
				s.With = make(map[string]string, len(valueNode.Content)/2)
				for k := 0; k < len(valueNode.Content)-1; k += 2 {
					withKey := valueNode.Content[k]
					withVal := valueNode.Content[k+1]
					if withKey != nil && withVal != nil {
						s.With[withKey.Value] = withVal.Value
					}
				}
			}
		case "env":
			if valueNode.Kind == yaml.MappingNode {
				s.Env = make(map[string]string, len(valueNode.Content)/2)
				for k := 0; k < len(valueNode.Content)-1; k += 2 {
					envKey := valueNode.Content[k]
					envVal := valueNode.Content[k+1]
					if envKey != nil && envVal != nil {
						s.Env[envKey.Value] = envVal.Value
					}
				}
			}
		case "working-directory":
			s.WorkingDirectory = valueNode.Value
		}
	}

	return nil
}

// IsAction returns true if this step uses a GitHub Action (has uses: field).
func (s *Step) IsAction() bool {
	return s.Uses != ""
}
