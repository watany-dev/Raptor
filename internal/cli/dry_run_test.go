package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/watany-dev/raptor/internal/workflow"
)

func TestNewDryRunFormatter(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := NewDryRunFormatter(buf)

	if formatter == nil {
		t.Fatal("expected non-nil formatter")
	}
	if formatter.stdout != buf {
		t.Error("stdout writer not set correctly")
	}
}

func TestDryRunFormatter_Format(t *testing.T) {
	t.Run("formats basic workflow", func(t *testing.T) {
		buf := &bytes.Buffer{}
		formatter := NewDryRunFormatter(buf)

		wf := &workflow.WorkflowFile{
			Name: "Test Workflow",
			Jobs: map[string]workflow.Job{
				"build": {
					Name:   "Build Job",
					RunsOn: "ubuntu-latest",
					Steps: []workflow.Step{
						{Name: "Step 1", Run: "echo hello"},
					},
				},
			},
		}

		results, err := formatter.Format(wf, []string{"build"}, "test.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := buf.String()

		// Check header
		if !strings.Contains(output, "DRY RUN MODE") {
			t.Error("output should contain DRY RUN MODE header")
		}
		if !strings.Contains(output, "Workflow: test.yml") {
			t.Error("output should contain workflow path")
		}
		if !strings.Contains(output, "Name: Test Workflow") {
			t.Error("output should contain workflow name")
		}

		// Check results
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].JobID != "build" {
			t.Errorf("expected job ID 'build', got %q", results[0].JobID)
		}
		if !results[0].Success {
			t.Error("expected success to be true")
		}
	})

	t.Run("formats workflow without name", func(t *testing.T) {
		buf := &bytes.Buffer{}
		formatter := NewDryRunFormatter(buf)

		wf := &workflow.WorkflowFile{
			Jobs: map[string]workflow.Job{
				"test": {
					Steps: []workflow.Step{
						{Run: "echo test"},
					},
				},
			},
		}

		_, err := formatter.Format(wf, []string{"test"}, "workflow.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := buf.String()
		// Should not contain "Name:" line for empty workflow name
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Name:") {
				t.Error("should not print Name: for empty workflow name")
			}
		}
	})

	t.Run("returns error for invalid job", func(t *testing.T) {
		buf := &bytes.Buffer{}
		formatter := NewDryRunFormatter(buf)

		wf := &workflow.WorkflowFile{
			Jobs: map[string]workflow.Job{
				"build": {
					Steps: []workflow.Step{
						{Run: "echo test"},
					},
				},
			},
		}

		_, err := formatter.Format(wf, []string{"nonexistent"}, "test.yml")
		if err == nil {
			t.Error("expected error for nonexistent job")
		}
		if !strings.Contains(err.Error(), "failed to select job") {
			t.Errorf("error should mention failed to select job, got: %v", err)
		}
	})

	t.Run("formats multiple jobs", func(t *testing.T) {
		buf := &bytes.Buffer{}
		formatter := NewDryRunFormatter(buf)

		wf := &workflow.WorkflowFile{
			Jobs: map[string]workflow.Job{
				"build": {
					Steps: []workflow.Step{{Run: "echo build"}},
				},
				"test": {
					Steps: []workflow.Step{{Run: "echo test"}},
				},
			},
		}

		results, err := formatter.Format(wf, []string{"build", "test"}, "test.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d", len(results))
		}

		output := buf.String()
		if !strings.Contains(output, "Job: build") {
			t.Error("output should contain Job: build")
		}
		if !strings.Contains(output, "Job: test") {
			t.Error("output should contain Job: test")
		}
	})

	t.Run("shows execution hint at end", func(t *testing.T) {
		buf := &bytes.Buffer{}
		formatter := NewDryRunFormatter(buf)

		wf := &workflow.WorkflowFile{
			Jobs: map[string]workflow.Job{
				"test": {
					Steps: []workflow.Step{{Run: "echo test"}},
				},
			},
		}

		_, err := formatter.Format(wf, []string{"test"}, "myworkflow.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "To execute this workflow") {
			t.Error("output should contain execution hint")
		}
		if !strings.Contains(output, "myworkflow.yml") {
			t.Error("execution hint should include workflow path")
		}
	})
}

func TestDryRunFormatter_formatJob(t *testing.T) {
	t.Run("formats job with name different from ID", func(t *testing.T) {
		buf := &bytes.Buffer{}
		formatter := NewDryRunFormatter(buf)

		job := &workflow.Job{
			Name:   "Build Application",
			RunsOn: "ubuntu-latest",
			Steps: []workflow.Step{
				{Name: "Step 1", Run: "echo hello"},
			},
		}

		result := formatter.formatJob("build", job)

		output := buf.String()
		if !strings.Contains(output, "Job: build") {
			t.Error("output should contain Job: build")
		}
		if !strings.Contains(output, "Name: Build Application") {
			t.Error("output should contain job name")
		}
		if !strings.Contains(output, "Runs-on: ubuntu-latest") {
			t.Error("output should contain runs-on")
		}

		if result.JobID != "build" {
			t.Errorf("expected JobID 'build', got %q", result.JobID)
		}
		if len(result.StepResults) != 1 {
			t.Errorf("expected 1 step result, got %d", len(result.StepResults))
		}
	})

	t.Run("formats job with name same as ID", func(t *testing.T) {
		buf := &bytes.Buffer{}
		formatter := NewDryRunFormatter(buf)

		job := &workflow.Job{
			Name: "build",
			Steps: []workflow.Step{
				{Run: "echo test"},
			},
		}

		_ = formatter.formatJob("build", job)

		output := buf.String()
		// Should not contain "Name:" when name equals ID
		if strings.Contains(output, "Name: build") {
			t.Error("should not print Name: when name equals ID")
		}
	})

	t.Run("formats job without runs-on", func(t *testing.T) {
		buf := &bytes.Buffer{}
		formatter := NewDryRunFormatter(buf)

		job := &workflow.Job{
			Steps: []workflow.Step{
				{Run: "echo test"},
			},
		}

		_ = formatter.formatJob("test", job)

		output := buf.String()
		if strings.Contains(output, "Runs-on:") {
			t.Error("should not print Runs-on: when empty")
		}
	})

	t.Run("formats steps with default names", func(t *testing.T) {
		buf := &bytes.Buffer{}
		formatter := NewDryRunFormatter(buf)

		job := &workflow.Job{
			Steps: []workflow.Step{
				{Run: "echo first"},
				{Run: "echo second"},
			},
		}

		result := formatter.formatJob("test", job)

		output := buf.String()
		if !strings.Contains(output, "Step 1") {
			t.Error("output should contain Step 1")
		}
		if !strings.Contains(output, "Step 2") {
			t.Error("output should contain Step 2")
		}

		if len(result.StepResults) != 2 {
			t.Errorf("expected 2 step results, got %d", len(result.StepResults))
		}
	})
}

func TestDryRunFormatter_formatStep(t *testing.T) {
	t.Run("formats step with working directory", func(t *testing.T) {
		buf := &bytes.Buffer{}
		formatter := NewDryRunFormatter(buf)

		step := &workflow.Step{
			WorkingDirectory: "src",
			Run:              "go build",
		}

		formatter.formatStep(0, "Build", step)

		output := buf.String()
		if !strings.Contains(output, "[1] Build") {
			t.Error("output should contain step number and name")
		}
		if !strings.Contains(output, "Working directory: src") {
			t.Error("output should contain working directory")
		}
	})

	t.Run("formats step with condition", func(t *testing.T) {
		buf := &bytes.Buffer{}
		formatter := NewDryRunFormatter(buf)

		step := &workflow.Step{
			If:  "success()",
			Run: "echo done",
		}

		formatter.formatStep(0, "Cleanup", step)

		output := buf.String()
		if !strings.Contains(output, "Condition: success()") {
			t.Error("output should contain condition")
		}
	})

	t.Run("formats step with environment variables", func(t *testing.T) {
		buf := &bytes.Buffer{}
		formatter := NewDryRunFormatter(buf)

		step := &workflow.Step{
			Env: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
			},
			Run: "echo $VAR1",
		}

		formatter.formatStep(0, "Test", step)

		output := buf.String()
		if !strings.Contains(output, "Environment: 2 variable(s)") {
			t.Error("output should contain environment count")
		}
	})

	t.Run("formats multiline command", func(t *testing.T) {
		buf := &bytes.Buffer{}
		formatter := NewDryRunFormatter(buf)

		step := &workflow.Step{
			Run: "echo line1\necho line2\necho line3",
		}

		formatter.formatStep(0, "Multi", step)

		output := buf.String()
		if !strings.Contains(output, "Command:") {
			t.Error("output should contain Command:")
		}
		if !strings.Contains(output, "echo line1") {
			t.Error("output should contain first line")
		}
		if !strings.Contains(output, "echo line2") {
			t.Error("output should contain second line")
		}
		if !strings.Contains(output, "echo line3") {
			t.Error("output should contain third line")
		}
	})

	t.Run("formats step without run command", func(t *testing.T) {
		buf := &bytes.Buffer{}
		formatter := NewDryRunFormatter(buf)

		step := &workflow.Step{
			WorkingDirectory: "test",
		}

		formatter.formatStep(0, "Empty", step)

		output := buf.String()
		if strings.Contains(output, "Command:") {
			t.Error("should not contain Command: for empty run")
		}
	})

	t.Run("step index is 1-based in output", func(t *testing.T) {
		buf := &bytes.Buffer{}
		formatter := NewDryRunFormatter(buf)

		step := &workflow.Step{Run: "test"}

		formatter.formatStep(4, "Fifth Step", step)

		output := buf.String()
		if !strings.Contains(output, "[5]") {
			t.Error("step index should be 1-based (5 for index 4)")
		}
	})
}
