package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWorkflowFile(t *testing.T) {
	t.Run("loads valid YAML with name, env, and jobs", func(t *testing.T) {
		yamlContent := `name: Test Workflow
env:
  GLOBAL_VAR: global_value
jobs:
  build:
    name: Build Job
    runs-on: ubuntu-latest
    env:
      JOB_VAR: job_value
    steps:
      - name: Run build
        run: echo "Building"
        env:
          STEP_VAR: step_value
      - id: test-step
        name: Run tests
        run: go test ./...
        working-directory: src
`
		tmpDir := t.TempDir()
		workflowPath := filepath.Join(tmpDir, "test.yml")
		if err := os.WriteFile(workflowPath, []byte(yamlContent), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		wf, err := LoadWorkflowFile(workflowPath)
		if err != nil {
			t.Fatalf("LoadWorkflowFile() error = %v", err)
		}

		// Check workflow name
		if wf.Name != "Test Workflow" {
			t.Errorf("Name = %q, want %q", wf.Name, "Test Workflow")
		}

		// Check workflow-level env
		if wf.Env["GLOBAL_VAR"] != "global_value" {
			t.Errorf("Env[GLOBAL_VAR] = %q, want %q", wf.Env["GLOBAL_VAR"], "global_value")
		}

		// Check jobs
		if len(wf.Jobs) != 1 {
			t.Fatalf("len(Jobs) = %d, want 1", len(wf.Jobs))
		}

		job, ok := wf.Jobs["build"]
		if !ok {
			t.Fatal("Jobs[build] not found")
		}

		if job.Name != "Build Job" {
			t.Errorf("job.Name = %q, want %q", job.Name, "Build Job")
		}

		if job.RunsOn != "ubuntu-latest" {
			t.Errorf("job.RunsOn = %q, want %q", job.RunsOn, "ubuntu-latest")
		}

		if job.Env["JOB_VAR"] != "job_value" {
			t.Errorf("job.Env[JOB_VAR] = %q, want %q", job.Env["JOB_VAR"], "job_value")
		}

		// Check steps
		if len(job.Steps) != 2 {
			t.Fatalf("len(Steps) = %d, want 2", len(job.Steps))
		}

		step0 := job.Steps[0]
		if step0.Name != "Run build" {
			t.Errorf("step0.Name = %q, want %q", step0.Name, "Run build")
		}
		if step0.Run != `echo "Building"` {
			t.Errorf("step0.Run = %q, want %q", step0.Run, `echo "Building"`)
		}
		if step0.Env["STEP_VAR"] != "step_value" {
			t.Errorf("step0.Env[STEP_VAR] = %q, want %q", step0.Env["STEP_VAR"], "step_value")
		}

		step1 := job.Steps[1]
		if step1.ID != "test-step" {
			t.Errorf("step1.ID = %q, want %q", step1.ID, "test-step")
		}
		if step1.WorkingDirectory != "src" {
			t.Errorf("step1.WorkingDirectory = %q, want %q", step1.WorkingDirectory, "src")
		}
	})

	t.Run("returns error for invalid YAML", func(t *testing.T) {
		invalidYAML := `name: Test
jobs:
  build:
    steps:
      - run: |
          echo "test
        invalid_indent:
  wrong: indentation
` // invalid indentation
		tmpDir := t.TempDir()
		workflowPath := filepath.Join(tmpDir, "invalid.yml")
		if err := os.WriteFile(workflowPath, []byte(invalidYAML), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		_, err := LoadWorkflowFile(workflowPath)
		if err == nil {
			t.Error("LoadWorkflowFile() expected error for invalid YAML, got nil")
		}
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := LoadWorkflowFile("/nonexistent/path/workflow.yml")
		if err == nil {
			t.Error("LoadWorkflowFile() expected error for non-existent file, got nil")
		}
	})

	t.Run("handles workflow with minimal content", func(t *testing.T) {
		yamlContent := `name: Minimal
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo hello
`
		tmpDir := t.TempDir()
		workflowPath := filepath.Join(tmpDir, "minimal.yml")
		if err := os.WriteFile(workflowPath, []byte(yamlContent), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		wf, err := LoadWorkflowFile(workflowPath)
		if err != nil {
			t.Fatalf("LoadWorkflowFile() error = %v", err)
		}

		if wf.Name != "Minimal" {
			t.Errorf("Name = %q, want %q", wf.Name, "Minimal")
		}

		// Env should be nil or empty
		if len(wf.Env) != 0 {
			t.Errorf("len(Env) = %d, want 0", len(wf.Env))
		}

		job := wf.Jobs["test"]
		if len(job.Steps) != 1 {
			t.Fatalf("len(Steps) = %d, want 1", len(job.Steps))
		}

		if job.Steps[0].Run != "echo hello" {
			t.Errorf("step.Run = %q, want %q", job.Steps[0].Run, "echo hello")
		}
	})
}

// BenchmarkLoadWorkflowFile benchmarks the workflow file loading performance.
func BenchmarkLoadWorkflowFile(b *testing.B) {
	// Create a workflow with multiple jobs and steps to simulate realistic usage
	yamlContent := `name: Benchmark Workflow
env:
  GLOBAL_VAR: global_value
jobs:
`
	// Add multiple jobs with multiple steps
	for i := 0; i < 10; i++ {
		yamlContent += fmt.Sprintf(`  job%d:
    name: Job %d
    runs-on: ubuntu-latest
    env:
      JOB_VAR: job_value_%d
    steps:
`, i, i, i)
		for j := 0; j < 5; j++ {
			yamlContent += fmt.Sprintf(`      - name: Step %d-%d
        run: echo "Running step %d in job %d"
        env:
          STEP_VAR: step_value_%d_%d
`, i, j, j, i, i, j)
		}
	}

	tmpDir := b.TempDir()
	workflowPath := filepath.Join(tmpDir, "benchmark.yml")
	if err := os.WriteFile(workflowPath, []byte(yamlContent), 0644); err != nil {
		b.Fatalf("failed to write test file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := LoadWorkflowFile(workflowPath)
		if err != nil {
			b.Fatalf("LoadWorkflowFile() error = %v", err)
		}
	}
}
