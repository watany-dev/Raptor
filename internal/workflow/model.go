package workflow

import "gopkg.in/yaml.v3"

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

// IsAction returns true if this step uses a GitHub Action (has uses: field).
func (s *Step) IsAction() bool {
	return s.Uses != ""
}
