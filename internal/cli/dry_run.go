package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/watany-dev/raptor/internal/workflow"
)

// DryRunFormatter formats and displays dry-run preview output.
type DryRunFormatter struct {
	stdout io.Writer
}

// NewDryRunFormatter creates a new DryRunFormatter with the given output writer.
func NewDryRunFormatter(stdout io.Writer) *DryRunFormatter {
	return &DryRunFormatter{
		stdout: stdout,
	}
}

// Format displays the dry-run preview for the given workflow and jobs.
func (drf *DryRunFormatter) Format(wf *workflow.WorkflowFile, jobIDs []string, workflowPath string) ([]*RunResult, error) {
	_, _ = fmt.Fprintln(drf.stdout, "")
	_, _ = fmt.Fprintln(drf.stdout, "🔍 DRY RUN MODE")
	_, _ = fmt.Fprintln(drf.stdout, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	_, _ = fmt.Fprintf(drf.stdout, "Workflow: %s\n", workflowPath)
	if wf.Name != "" {
		_, _ = fmt.Fprintf(drf.stdout, "Name: %s\n", wf.Name)
	}
	_, _ = fmt.Fprintln(drf.stdout, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	_, _ = fmt.Fprintln(drf.stdout, "")

	var results []*RunResult

	for _, jobID := range jobIDs {
		job, err := workflow.SelectJob(wf, jobID)
		if err != nil {
			return nil, fmt.Errorf("failed to select job: %w", err)
		}

		result := drf.formatJob(jobID, job)
		results = append(results, result)
	}

	_, _ = fmt.Fprintln(drf.stdout, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	_, _ = fmt.Fprintln(drf.stdout, "To execute this workflow, use: raptor run -w", workflowPath)
	_, _ = fmt.Fprintln(drf.stdout, "")

	return results, nil
}

// formatJob formats a single job's dry-run preview.
func (drf *DryRunFormatter) formatJob(jobID string, job *workflow.Job) *RunResult {
	_, _ = fmt.Fprintf(drf.stdout, "📋 Job: %s\n", jobID)
	if job.Name != "" && job.Name != jobID {
		_, _ = fmt.Fprintf(drf.stdout, "   Name: %s\n", job.Name)
	}
	if len(job.Needs) > 0 {
		_, _ = fmt.Fprintf(drf.stdout, "   Depends on: %s\n", strings.Join(job.Needs, ", "))
	}
	if job.RunsOn != "" {
		_, _ = fmt.Fprintf(drf.stdout, "   Runs-on: %s\n", job.RunsOn)
	}
	_, _ = fmt.Fprintln(drf.stdout, "")

	result := &RunResult{
		JobID:       jobID,
		Success:     true,
		StepResults: make([]StepResult, 0, len(job.Steps)),
	}

	for i, step := range job.Steps {
		stepName := step.Name
		if stepName == "" {
			stepName = fmt.Sprintf("Step %d", i+1)
		}

		drf.formatStep(i, stepName, &step)

		result.StepResults = append(result.StepResults, StepResult{
			StepIndex: i,
			StepName:  stepName,
			ExitCode:  0,
		})
	}

	return result
}

// formatStep formats a single step's dry-run preview.
func (drf *DryRunFormatter) formatStep(index int, stepName string, step *workflow.Step) {
	_, _ = fmt.Fprintf(drf.stdout, "   [%d] %s\n", index+1, stepName)

	// Show warning for unsupported GitHub Actions
	if step.IsAction() {
		_, _ = fmt.Fprintf(drf.stdout, "       ⚠️  Uses: %s (not supported, will be skipped)\n", step.Uses)
		if len(step.With) > 0 {
			_, _ = fmt.Fprintf(drf.stdout, "       With: %d parameter(s)\n", len(step.With))
		}
	}

	if step.WorkingDirectory != "" {
		_, _ = fmt.Fprintf(drf.stdout, "       Working directory: %s\n", step.WorkingDirectory)
	}
	if step.If != "" {
		_, _ = fmt.Fprintf(drf.stdout, "       Condition: %s\n", step.If)
	}
	if len(step.Env) > 0 {
		_, _ = fmt.Fprintf(drf.stdout, "       Environment: %d variable(s)\n", len(step.Env))
	}
	if step.Run != "" {
		// Show the command, indented
		lines := strings.Split(strings.TrimSpace(step.Run), "\n")
		_, _ = fmt.Fprintln(drf.stdout, "       Command:")
		for _, line := range lines {
			_, _ = fmt.Fprintf(drf.stdout, "         %s\n", line)
		}
	}
	_, _ = fmt.Fprintln(drf.stdout, "")
}
