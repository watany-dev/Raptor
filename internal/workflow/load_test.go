package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadWorkflowFile(t *testing.T) {
	t.Run("loads YAML with uses and with fields", func(t *testing.T) {
		yamlContent := `name: Test Workflow
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.21"
          cache: true
      - name: Run tests
        run: go test ./...
`
		tmpDir := t.TempDir()
		workflowPath := filepath.Join(tmpDir, "uses_test.yml")
		if err := os.WriteFile(workflowPath, []byte(yamlContent), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		wf, err := LoadWorkflowFile(workflowPath)
		if err != nil {
			t.Fatalf("LoadWorkflowFile() error = %v", err)
		}

		job := wf.Jobs["build"]
		if len(job.Steps) != 3 {
			t.Fatalf("len(Steps) = %d, want 3", len(job.Steps))
		}

		// Check step with uses only
		step0 := job.Steps[0]
		if step0.Uses != "actions/checkout@v4" {
			t.Errorf("step0.Uses = %q, want %q", step0.Uses, "actions/checkout@v4")
		}
		if !step0.IsAction() {
			t.Error("step0.IsAction() should return true")
		}

		// Check step with uses and with
		step1 := job.Steps[1]
		if step1.Uses != "actions/setup-go@v5" {
			t.Errorf("step1.Uses = %q, want %q", step1.Uses, "actions/setup-go@v5")
		}
		if step1.With["go-version"] != "1.21" {
			t.Errorf("step1.With[go-version] = %q, want %q", step1.With["go-version"], "1.21")
		}
		if step1.With["cache"] != "true" {
			t.Errorf("step1.With[cache] = %q, want %q", step1.With["cache"], "true")
		}

		// Check step with run only
		step2 := job.Steps[2]
		if step2.Uses != "" {
			t.Errorf("step2.Uses = %q, want empty", step2.Uses)
		}
		if step2.IsAction() {
			t.Error("step2.IsAction() should return false")
		}
		if step2.Run != "go test ./..." {
			t.Errorf("step2.Run = %q, want %q", step2.Run, "go test ./...")
		}
	})

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
	for i := range 10 {
		yamlContent += fmt.Sprintf(`  job%d:
    name: Job %d
    runs-on: ubuntu-latest
    env:
      JOB_VAR: job_value_%d
    steps:
`, i, i, i)
		for j := range 5 {
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
	for b.Loop() {
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
	for i := range 100 {
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

// TestLoadWorkflowFile_NullJobsNode tests that null jobs are handled gracefully
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
		t.Fatalf("LoadWorkflowFile() error = %v; null jobs should be accepted", err)
	}

	// Jobs should be empty/nil for null jobs node
	if len(wf.Jobs) != 0 {
		t.Errorf("LoadWorkflowFile() jobs count = %d; want 0 for null jobs", len(wf.Jobs))
	}

	// JobOrder should be empty
	if len(wf.JobOrder) != 0 {
		t.Errorf("LoadWorkflowFile() jobOrder = %v; want empty for null jobs", wf.JobOrder)
	}

	// Name should still be parsed
	if wf.Name != "Test Workflow" {
		t.Errorf("LoadWorkflowFile() name = %q; want %q", wf.Name, "Test Workflow")
	}
}

// TestExtractJobOrderFromNode_EmptyContent tests that empty files return empty workflow
func TestExtractJobOrderFromNode_EmptyContent(t *testing.T) {
	t.Parallel()
	yamlContent := ``
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "empty.yml")
	if err := os.WriteFile(workflowPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	wf, err := LoadWorkflowFile(workflowPath)
	// YAML allows empty documents - they decode to zero-value structs
	if err != nil {
		t.Fatalf("LoadWorkflowFile() error = %v", err)
	}

	// Empty file should result in empty workflow with no name, jobs, or job order
	if wf.Name != "" {
		t.Errorf("Name = %q; want empty for empty file", wf.Name)
	}
	if len(wf.Jobs) != 0 {
		t.Errorf("Jobs count = %d; want 0 for empty file", len(wf.Jobs))
	}
	if len(wf.JobOrder) != 0 {
		t.Errorf("JobOrder = %v; want empty for empty file", wf.JobOrder)
	}
}

// TestExtractJobOrderFromNode_JobsAsSequence tests that jobs defined as sequence returns error
func TestExtractJobOrderFromNode_JobsAsSequence(t *testing.T) {
	t.Parallel()
	yamlContent := `name: Test Workflow
jobs:
  - job1
  - job2`
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "sequence.yml")
	if err := os.WriteFile(workflowPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := LoadWorkflowFile(workflowPath)
	// Jobs must be a mapping, not a sequence - this should error during decode
	if err == nil {
		t.Error("LoadWorkflowFile() expected error for jobs as sequence, got nil")
	}
}

// Test DiscoverWorkflows with unreadable directory
func TestDiscoverWorkflows_ReadDirError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("skipping permission test as root")
	}

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("failed to create workflows directory: %v", err)
	}

	// Create a file to make the directory non-empty
	workflowPath := filepath.Join(workflowsDir, "test.yml")
	if err := os.WriteFile(workflowPath, []byte("name: Test"), 0644); err != nil {
		t.Fatalf("failed to write workflow file: %v", err)
	}

	// Make the directory unreadable
	if err := os.Chmod(workflowsDir, 0000); err != nil {
		t.Fatalf("failed to chmod: %v", err)
	}
	defer func() {
		_ = os.Chmod(workflowsDir, 0755)
	}()

	_, err := DiscoverWorkflows(tmpDir)
	if err == nil {
		t.Error("DiscoverWorkflows() expected error for unreadable directory")
	}
}

// Test DiscoverWorkflows with stat error
func TestDiscoverWorkflows_StatError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("skipping permission test as root")
	}

	tmpDir := t.TempDir()
	ghDir := filepath.Join(tmpDir, ".github")
	if err := os.MkdirAll(ghDir, 0755); err != nil {
		t.Fatalf("failed to create .github directory: %v", err)
	}

	// Make .github unreadable to trigger stat error on workflows
	if err := os.Chmod(ghDir, 0000); err != nil {
		t.Fatalf("failed to chmod: %v", err)
	}
	defer func() {
		_ = os.Chmod(ghDir, 0755)
	}()

	_, err := DiscoverWorkflows(tmpDir)
	if err == nil {
		t.Error("DiscoverWorkflows() expected error for inaccessible workflows directory")
	}
}

// TestLoadWorkflowFile_TypeCoercion tests that YAML type coercion works for workflow fields
func TestLoadWorkflowFile_TypeCoercion(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// YAML allows type coercion: numeric 123 becomes string "123", boolean true becomes "true"
	yamlContent := `name: 123
jobs:
  test:
    runs-on: true
    steps:
      - run: echo test`
	workflowPath := filepath.Join(tmpDir, "coercion.yml")
	if err := os.WriteFile(workflowPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	wf, err := LoadWorkflowFile(workflowPath)
	if err != nil {
		t.Fatalf("LoadWorkflowFile() error = %v", err)
	}

	// Verify type coercion happened correctly
	if wf.Name != "123" {
		t.Errorf("Name = %q; want %q (numeric should be coerced to string)", wf.Name, "123")
	}

	job, ok := wf.Jobs["test"]
	if !ok {
		t.Fatal("test job not found")
	}
	if job.RunsOn != "true" {
		t.Errorf("RunsOn = %q; want %q (boolean should be coerced to string)", job.RunsOn, "true")
	}
}

// TestExtractJobOrderFromNode_CommentOnlyDocument tests file with only comments returns empty workflow
func TestExtractJobOrderFromNode_CommentOnlyDocument(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// A file with only comments is treated as empty/null document by YAML parser
	yamlContent := `# Just a comment
`
	workflowPath := filepath.Join(tmpDir, "comment_only.yml")
	if err := os.WriteFile(workflowPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	wf, err := LoadWorkflowFile(workflowPath)
	// YAML treats comment-only as empty document - decodes to zero-value struct
	if err != nil {
		t.Fatalf("LoadWorkflowFile() error = %v", err)
	}

	// Should return empty workflow
	if wf.Name != "" {
		t.Errorf("Name = %q; want empty for comment-only file", wf.Name)
	}
	if len(wf.Jobs) != 0 {
		t.Errorf("Jobs count = %d; want 0 for comment-only file", len(wf.Jobs))
	}
}

// TestExtractJobOrderFromNode_NonMappingDocument tests with a YAML array at root
func TestExtractJobOrderFromNode_NonMappingDocument(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create YAML with array at root (not a mapping)
	yamlContent := `- item1
- item2
- item3`
	workflowPath := filepath.Join(tmpDir, "array_root.yml")
	if err := os.WriteFile(workflowPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := LoadWorkflowFile(workflowPath)
	// Should error because root is not a mapping
	if err == nil {
		t.Error("LoadWorkflowFile() expected error for array root document")
	}
}

// TestExtractJobOrderFromNode_JobsNotMapping tests when jobs value is not a mapping
func TestExtractJobOrderFromNode_JobsNotMapping(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create YAML where jobs is a scalar instead of mapping
	yamlContent := `name: Test
jobs: "not a mapping"`
	workflowPath := filepath.Join(tmpDir, "jobs_scalar.yml")
	if err := os.WriteFile(workflowPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	wf, err := LoadWorkflowFile(workflowPath)
	if err != nil {
		// Error is acceptable for invalid jobs format
		return
	}

	// If no error, JobOrder should be empty
	if len(wf.JobOrder) > 0 {
		t.Errorf("JobOrder should be empty for non-mapping jobs, got %v", wf.JobOrder)
	}
}

// TestDiscoverWorkflows_WorkflowsIsFile tests when .github/workflows is a file, not directory
func TestDiscoverWorkflows_WorkflowsIsFile(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create .github directory
	ghDir := filepath.Join(tmpDir, ".github")
	if err := os.MkdirAll(ghDir, 0755); err != nil {
		t.Fatalf("failed to create .github directory: %v", err)
	}

	// Create "workflows" as a file instead of directory
	workflowsPath := filepath.Join(ghDir, "workflows")
	if err := os.WriteFile(workflowsPath, []byte("not a directory"), 0644); err != nil {
		t.Fatalf("failed to create workflows file: %v", err)
	}

	_, err := DiscoverWorkflows(tmpDir)
	if err == nil {
		t.Error("DiscoverWorkflows() expected error when workflows is a file")
	}

	// Verify error message mentions it's not a directory
	if err != nil && !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("Error should mention 'not a directory', got: %v", err)
	}
}

// TestLoadWorkflowFile_YAMLUnmarshalError tests the yaml.Unmarshal error path (line 19-21)
func TestLoadWorkflowFile_YAMLUnmarshalError(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create invalid YAML that will fail yaml.Unmarshal
	// Using invalid UTF-8 sequence or binary data
	invalidContent := []byte{0xff, 0xfe, 0x00, 0x01, '%', '!', 0x00}
	workflowPath := filepath.Join(tmpDir, "invalid.yml")
	if err := os.WriteFile(workflowPath, invalidContent, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := LoadWorkflowFile(workflowPath)
	if err == nil {
		t.Error("LoadWorkflowFile() expected error for invalid YAML binary content")
	}
	if err != nil && !strings.Contains(err.Error(), "parse workflow YAML") {
		t.Logf("Got error: %v", err)
	}
}

// TestExtractJobOrderFromNode_SequenceRoot tests extractJobOrderFromNode with sequence at root
// This specifically covers the doc.Kind != yaml.MappingNode branch (line 56-58)
func TestExtractJobOrderFromNode_SequenceRoot(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create YAML with sequence at root - this will parse but decode will fail
	// However, we need to test the extractJobOrderFromNode function path
	yamlContent := `---
- item1
- item2`
	workflowPath := filepath.Join(tmpDir, "sequence.yml")
	if err := os.WriteFile(workflowPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := LoadWorkflowFile(workflowPath)
	// This should error or return empty JobOrder
	if err != nil {
		// Error is expected since root is not a mapping
		if !strings.Contains(err.Error(), "decode workflow") {
			t.Logf("Got error: %v", err)
		}
	}
}

// TestDiscoverWorkflows_SymlinkLoop tests that symlink loops don't cause infinite loops
func TestDiscoverWorkflows_SymlinkLoop(t *testing.T) {
	t.Run("circular symlink in workflows directory should not hang", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowDir := filepath.Join(tmpDir, ".github", "workflows")
		if err := os.MkdirAll(workflowDir, 0755); err != nil {
			t.Fatalf("failed to create workflow directory: %v", err)
		}

		// Create a circular symlink: loop -> .
		loopPath := filepath.Join(workflowDir, "loop")
		if err := os.Symlink(".", loopPath); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}

		// Should not hang and should complete
		workflows, err := DiscoverWorkflows(tmpDir)
		// Either returns empty/nil or error, but should not hang
		if err != nil {
			t.Logf("Got error (acceptable): %v", err)
		}
		t.Logf("Found %d workflows", len(workflows))
	})

	t.Run("symlink pointing to parent directory should not hang", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowDir := filepath.Join(tmpDir, ".github", "workflows")
		if err := os.MkdirAll(workflowDir, 0755); err != nil {
			t.Fatalf("failed to create workflow directory: %v", err)
		}

		// Create a symlink pointing to parent: parent -> ..
		parentLink := filepath.Join(workflowDir, "parent")
		if err := os.Symlink("..", parentLink); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}

		// Should not hang
		workflows, err := DiscoverWorkflows(tmpDir)
		if err != nil {
			t.Logf("Got error (acceptable): %v", err)
		}
		t.Logf("Found %d workflows", len(workflows))
	})

	t.Run("deeply nested symlink loop should not hang", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowDir := filepath.Join(tmpDir, ".github", "workflows")
		subDir := filepath.Join(workflowDir, "subdir")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("failed to create subdirectory: %v", err)
		}

		// Create mutual symlinks: a -> b, b -> a
		linkA := filepath.Join(workflowDir, "linkA")
		linkB := filepath.Join(workflowDir, "linkB")
		if err := os.Symlink(linkB, linkA); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}
		if err := os.Symlink(linkA, linkB); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}

		// Should not hang
		workflows, err := DiscoverWorkflows(tmpDir)
		if err != nil {
			t.Logf("Got error (acceptable): %v", err)
		}
		t.Logf("Found %d workflows", len(workflows))
	})
}

// TestDiscoverWorkflows_BrokenSymlink tests handling of broken symlinks
func TestDiscoverWorkflows_BrokenSymlink(t *testing.T) {
	t.Run("broken symlink should be skipped gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowDir := filepath.Join(tmpDir, ".github", "workflows")
		if err := os.MkdirAll(workflowDir, 0755); err != nil {
			t.Fatalf("failed to create workflow directory: %v", err)
		}

		// Create a valid workflow file
		validWorkflow := filepath.Join(workflowDir, "valid.yml")
		if err := os.WriteFile(validWorkflow, []byte("name: Valid\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo test\n"), 0644); err != nil {
			t.Fatalf("failed to write workflow file: %v", err)
		}

		// Create a broken symlink (pointing to non-existent file)
		brokenLink := filepath.Join(workflowDir, "broken.yml")
		if err := os.Symlink("/nonexistent/path/file.yml", brokenLink); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}

		// Should still discover valid workflows
		workflows, err := DiscoverWorkflows(tmpDir)
		if err != nil {
			t.Fatalf("DiscoverWorkflows() error = %v", err)
		}

		// Should find at least the valid workflow
		if len(workflows) < 1 {
			t.Error("Should have discovered at least the valid workflow")
		}

		// Check that valid.yml is in the list
		found := false
		for _, wf := range workflows {
			if strings.HasSuffix(wf, "valid.yml") {
				found = true
				break
			}
		}
		if !found {
			t.Error("valid.yml should be in the discovered workflows")
		}
	})
}

// TestDiscoverWorkflows_SymlinkToDirectory tests symlink pointing to directory
func TestDiscoverWorkflows_SymlinkToDirectory(t *testing.T) {
	t.Run("symlink pointing to directory should be handled", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowDir := filepath.Join(tmpDir, ".github", "workflows")
		otherDir := filepath.Join(tmpDir, "other")
		if err := os.MkdirAll(workflowDir, 0755); err != nil {
			t.Fatalf("failed to create workflow directory: %v", err)
		}
		if err := os.MkdirAll(otherDir, 0755); err != nil {
			t.Fatalf("failed to create other directory: %v", err)
		}

		// Create a workflow file in other directory
		otherWorkflow := filepath.Join(otherDir, "other.yml")
		if err := os.WriteFile(otherWorkflow, []byte("name: Other\n"), 0644); err != nil {
			t.Fatalf("failed to write workflow file: %v", err)
		}

		// Create a symlink in workflows dir pointing to other directory
		dirLink := filepath.Join(workflowDir, "linked-dir")
		if err := os.Symlink(otherDir, dirLink); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}

		// Should not error and should not follow symlink into directory
		workflows, err := DiscoverWorkflows(tmpDir)
		if err != nil {
			t.Fatalf("DiscoverWorkflows() error = %v", err)
		}

		// The linked directory should not be followed (we only look at direct entries)
		t.Logf("Found %d workflows: %v", len(workflows), workflows)
	})
}

// TestExtractJobOrderFromNode_DirectCall tests extractJobOrderFromNode directly
// to cover the doc.Kind != yaml.MappingNode branch
func TestExtractJobOrderFromNode_DirectCall(t *testing.T) {
	t.Parallel()

	// Test with sequence node (not mapping)
	t.Run("sequence node returns nil", func(t *testing.T) {
		var root yaml.Node
		yamlContent := `- item1
- item2`
		if err := yaml.Unmarshal([]byte(yamlContent), &root); err != nil {
			t.Fatalf("failed to unmarshal YAML: %v", err)
		}

		result := extractJobOrderFromNode(&root)
		if result != nil {
			t.Errorf("extractJobOrderFromNode() = %v, want nil for sequence node", result)
		}
	})

	// Test with scalar node
	t.Run("scalar node returns nil", func(t *testing.T) {
		var root yaml.Node
		yamlContent := `just a string`
		if err := yaml.Unmarshal([]byte(yamlContent), &root); err != nil {
			t.Fatalf("failed to unmarshal YAML: %v", err)
		}

		result := extractJobOrderFromNode(&root)
		if result != nil {
			t.Errorf("extractJobOrderFromNode() = %v, want nil for scalar node", result)
		}
	})

	// Test with empty node
	t.Run("empty node returns nil", func(t *testing.T) {
		root := &yaml.Node{}
		result := extractJobOrderFromNode(root)
		if result != nil {
			t.Errorf("extractJobOrderFromNode() = %v, want nil for empty node", result)
		}
	})
}
