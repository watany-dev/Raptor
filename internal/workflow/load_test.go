package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// TestExtractJobOrder tests that job order is correctly extracted from a workflow
func TestExtractJobOrder(t *testing.T) {
	t.Run("extracts job order correctly", func(t *testing.T) {
		yamlContent := `name: Test Workflow
jobs:
  first:
    runs-on: ubuntu-latest
    steps:
      - run: echo first
  second:
    runs-on: ubuntu-latest
    steps:
      - run: echo second
  third:
    runs-on: ubuntu-latest
    steps:
      - run: echo third
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

		if len(wf.JobOrder) != 3 {
			t.Fatalf("len(JobOrder) = %d, want 3", len(wf.JobOrder))
		}

		expectedOrder := []string{"first", "second", "third"}
		for i, jobID := range expectedOrder {
			if wf.JobOrder[i] != jobID {
				t.Errorf("JobOrder[%d] = %q, want %q", i, wf.JobOrder[i], jobID)
			}
		}
	})

	t.Run("handles workflow without jobs", func(t *testing.T) {
		yamlContent := `name: Empty Workflow
env:
  VAR: value
`
		tmpDir := t.TempDir()
		workflowPath := filepath.Join(tmpDir, "empty.yml")
		if err := os.WriteFile(workflowPath, []byte(yamlContent), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		wf, err := LoadWorkflowFile(workflowPath)
		if err != nil {
			t.Fatalf("LoadWorkflowFile() error = %v", err)
		}

		if len(wf.JobOrder) != 0 {
			t.Errorf("len(JobOrder) = %d, want 0", len(wf.JobOrder))
		}
	})

	t.Run("handles malformed YAML with non-mapping jobs", func(t *testing.T) {
		// This YAML has jobs as an array instead of a mapping
		yamlContent := `name: Test Workflow
jobs:
  - job1
  - job2
`
		tmpDir := t.TempDir()
		workflowPath := filepath.Join(tmpDir, "malformed.yml")
		if err := os.WriteFile(workflowPath, []byte(yamlContent), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		wf, err := LoadWorkflowFile(workflowPath)
		// This should not error because YAML is technically valid, but jobs won't be parsed
		if err != nil {
			// This is acceptable if it errors
			return
		}

		// If no error, JobOrder should be empty since jobs is not a mapping
		if len(wf.JobOrder) > 0 {
			t.Errorf("JobOrder should be empty for non-mapping jobs, got %v", wf.JobOrder)
		}
	})

	t.Run("handles workflow with only single job", func(t *testing.T) {
		yamlContent := `name: Single Job Workflow
jobs:
  single:
    runs-on: ubuntu-latest
    steps:
      - run: echo test
`
		tmpDir := t.TempDir()
		workflowPath := filepath.Join(tmpDir, "single.yml")
		if err := os.WriteFile(workflowPath, []byte(yamlContent), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		wf, err := LoadWorkflowFile(workflowPath)
		if err != nil {
			t.Fatalf("LoadWorkflowFile() error = %v", err)
		}

		if len(wf.JobOrder) != 1 || wf.JobOrder[0] != "single" {
			t.Errorf("JobOrder = %v, want [\"single\"]", wf.JobOrder)
		}
	})

	t.Run("handles YAML with non-mapping root", func(t *testing.T) {
		// YAML that is just a scalar, not a mapping
		yamlContent := `just a string`
		tmpDir := t.TempDir()
		workflowPath := filepath.Join(tmpDir, "scalar.yml")
		if err := os.WriteFile(workflowPath, []byte(yamlContent), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		_, err := LoadWorkflowFile(workflowPath)
		// This should error because it cannot decode a scalar as a WorkflowFile
		if err == nil {
			t.Error("LoadWorkflowFile() expected error for scalar YAML, got nil")
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

// Test DiscoverWorkflows with non-existent workflows directory
func TestDiscoverWorkflows_NonExistentDirectory(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	// Don't create .github/workflows directory

	workflows, err := DiscoverWorkflows(tmpDir)
	// Should either return empty list or error gracefully
	// Both returning error and empty list are acceptable behaviors
	if err == nil && len(workflows) != 0 {
		// If no error, should return empty list
		t.Errorf("DiscoverWorkflows() expected empty list for non-existent directory, got %d workflows", len(workflows))
	}
}

// Test DiscoverWorkflows with symlinks
func TestDiscoverWorkflows_Symlinks(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	workflowDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		t.Fatalf("failed to create workflow directory: %v", err)
	}

	// Create a workflow file
	workflowPath := filepath.Join(workflowDir, "test.yml")
	if err := os.WriteFile(workflowPath, []byte("name: Test\njobs:\n  test:\n    runs-on: ubuntu-latest\n"), 0644); err != nil {
		t.Fatalf("failed to write workflow file: %v", err)
	}

	// Create a symlink to the workflow file
	symlinkPath := filepath.Join(workflowDir, "symlink.yml")
	if err := os.Symlink(workflowPath, symlinkPath); err != nil {
		t.Skipf("symlink not supported on this platform: %v", err)
	}

	workflows, err := DiscoverWorkflows(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverWorkflows() error = %v", err)
	}

	// Should discover both files (symlink and original)
	if len(workflows) < 1 {
		t.Error("DiscoverWorkflows() should discover at least the original file")
	}
}

// Test LoadWorkflowFile with large files
func TestLoadWorkflowFile_LargeFile(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create a large workflow file with many jobs
	var yamlContent strings.Builder
	yamlContent.WriteString("name: Large Workflow\njobs:\n")
	for i := 0; i < 100; i++ {
		yamlContent.WriteString(fmt.Sprintf("  job%d:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo test\n", i))
	}

	workflowPath := filepath.Join(tmpDir, "large.yml")
	if err := os.WriteFile(workflowPath, []byte(yamlContent.String()), 0644); err != nil {
		t.Fatalf("failed to write workflow file: %v", err)
	}

	wf, err := LoadWorkflowFile(workflowPath)
	if err != nil {
		t.Fatalf("LoadWorkflowFile() error = %v", err)
	}

	if len(wf.Jobs) != 100 {
		t.Errorf("LoadWorkflowFile() expected 100 jobs, got %d", len(wf.Jobs))
	}
}

// Test LoadWorkflowFile with null jobs node
func TestLoadWorkflowFile_NullJobsNode(t *testing.T) {
	t.Parallel()
	yamlContent := `name: Test Workflow
jobs: null`
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "null_jobs.yml")
	if err := os.WriteFile(workflowPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	wf, err := LoadWorkflowFile(workflowPath)
	if err != nil {
		// It's acceptable to error on null jobs
		return
	}

	// Or if it doesn't error, jobs should be empty/nil
	if len(wf.Jobs) > 0 {
		t.Error("LoadWorkflowFile() expected nil or empty jobs for null jobs node")
	}
}
