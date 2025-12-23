package workflow

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
	// Env contains environment variables for this step.
	Env map[string]string `yaml:"env"`
	// WorkingDirectory specifies the working directory for run commands.
	WorkingDirectory string `yaml:"working-directory"`
}
