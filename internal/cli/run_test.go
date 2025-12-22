package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/watany-dev/raptor/internal/executor"
)

// mockExecutor is a mock implementation of executor.Executor for testing.
type mockExecutor struct {
	results []executor.Result
	calls   []executor.Config
	callIdx int
}

func newMockExecutor(results ...executor.Result) *mockExecutor {
	return &mockExecutor{
		results: results,
		calls:   make([]executor.Config, 0),
	}
}

func (m *mockExecutor) Execute(config executor.Config) (executor.Result, error) {
	m.calls = append(m.calls, config)
	if m.callIdx < len(m.results) {
		result := m.results[m.callIdx]
		m.callIdx++
		return result, nil
	}
	return executor.Result{ExitCode: 0}, nil
}

func TestRunner_Run_MultipleStepsExecuted(t *testing.T) {
	// Create a temporary workflow file
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	workflowContent := `
name: Test Workflow
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Step 1
        run: echo "step 1"
      - name: Step 2
        run: echo "step 2"
      - name: Step 3
        run: echo "step 3"
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	mock := newMockExecutor(
		executor.Result{ExitCode: 0, Stdout: "step 1\n"},
		executor.Result{ExitCode: 0, Stdout: "step 2\n"},
		executor.Result{ExitCode: 0, Stdout: "step 3\n"},
	)

	runner := NewRunner(mock)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner.SetOutput(stdout, stderr)

	results, err := runner.Run(&RunOptions{
		Workflow:   workflowPath,
		Job:        "test",
		WorkingDir: tmpDir,
	})

	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify all steps were executed
	if len(mock.calls) != 3 {
		t.Errorf("Expected 3 step executions, got %d", len(mock.calls))
	}

	// Verify single job result
	if len(results) != 1 {
		t.Fatalf("Expected 1 job result, got %d", len(results))
	}
	result := results[0]

	// Verify all steps completed successfully
	if !result.Success {
		t.Error("Expected job to succeed")
	}

	if len(result.StepResults) != 3 {
		t.Errorf("Expected 3 step results, got %d", len(result.StepResults))
	}
}

func TestRunner_Run_StepFailure(t *testing.T) {
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	workflowContent := `
name: Test Workflow
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Step 1
        run: echo "success"
      - name: Step 2 (fails)
        run: exit 1
      - name: Step 3 (should not run)
        run: echo "never"
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	mock := newMockExecutor(
		executor.Result{ExitCode: 0, Stdout: "success\n"},
		executor.Result{ExitCode: 1, Stderr: "failed\n"},
	)

	runner := NewRunner(mock)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner.SetOutput(stdout, stderr)

	results, err := runner.Run(&RunOptions{
		Workflow:   workflowPath,
		Job:        "test",
		WorkingDir: tmpDir,
	})

	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify single job result
	if len(results) != 1 {
		t.Fatalf("Expected 1 job result, got %d", len(results))
	}
	result := results[0]

	// Verify job failed
	if result.Success {
		t.Error("Expected job to fail")
	}

	// Verify only 2 steps executed (third should be skipped)
	if len(mock.calls) != 2 {
		t.Errorf("Expected 2 step executions, got %d", len(mock.calls))
	}

	// Verify step 2 has non-zero exit code
	if len(result.StepResults) != 2 {
		t.Errorf("Expected 2 step results, got %d", len(result.StepResults))
	}
	if result.StepResults[1].ExitCode != 1 {
		t.Errorf("Expected exit code 1 for step 2, got %d", result.StepResults[1].ExitCode)
	}
}

func TestRunner_Run_EnvInheritance(t *testing.T) {
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	workflowContent := `
name: Test Workflow
env:
  WORKFLOW_VAR: workflow_value
jobs:
  test:
    runs-on: ubuntu-latest
    env:
      JOB_VAR: job_value
    steps:
      - name: Step 1
        env:
          STEP_VAR: step_value
        run: echo "test"
      - name: Step 2
        run: echo "inherit"
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	mock := newMockExecutor(
		executor.Result{ExitCode: 0},
		executor.Result{ExitCode: 0},
	)

	runner := NewRunner(mock)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner.SetOutput(stdout, stderr)

	_, err := runner.Run(&RunOptions{
		Workflow:   workflowPath,
		Job:        "test",
		WorkingDir: tmpDir,
	})

	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Check step 1 has all env vars
	step1Env := mock.calls[0].Env
	if step1Env["WORKFLOW_VAR"] != "workflow_value" {
		t.Errorf("Step 1 missing WORKFLOW_VAR, got %v", step1Env["WORKFLOW_VAR"])
	}
	if step1Env["JOB_VAR"] != "job_value" {
		t.Errorf("Step 1 missing JOB_VAR, got %v", step1Env["JOB_VAR"])
	}
	if step1Env["STEP_VAR"] != "step_value" {
		t.Errorf("Step 1 missing STEP_VAR, got %v", step1Env["STEP_VAR"])
	}

	// Check step 2 has workflow and job vars but not step 1's var
	step2Env := mock.calls[1].Env
	if step2Env["WORKFLOW_VAR"] != "workflow_value" {
		t.Errorf("Step 2 missing WORKFLOW_VAR, got %v", step2Env["WORKFLOW_VAR"])
	}
	if step2Env["JOB_VAR"] != "job_value" {
		t.Errorf("Step 2 missing JOB_VAR, got %v", step2Env["JOB_VAR"])
	}
	// STEP_VAR should not be inherited from step 1
	if _, exists := step2Env["STEP_VAR"]; exists {
		t.Error("Step 2 should not have STEP_VAR from step 1")
	}
}

func TestRunner_Run_GithubEnvPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	workflowContent := `
name: Test Workflow
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Set env
        run: echo "MY_VAR=hello" >> $GITHUB_ENV
      - name: Use env
        run: echo $MY_VAR
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Create a mock that writes to GITHUB_ENV on first call
	mock := &envSettingMock{
		results: []executor.Result{
			{ExitCode: 0, Stdout: "setting env\n"},
			{ExitCode: 0, Stdout: "hello\n"},
		},
	}

	runner := NewRunner(mock)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner.SetOutput(stdout, stderr)

	_, err := runner.Run(&RunOptions{
		Workflow:   workflowPath,
		Job:        "test",
		WorkingDir: tmpDir,
	})

	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Check that step 2 has the env var from GITHUB_ENV
	if len(mock.calls) != 2 {
		t.Fatalf("Expected 2 calls, got %d", len(mock.calls))
	}
	step2Env := mock.calls[1].Env
	if step2Env["MY_VAR"] != "hello" {
		t.Errorf("Step 2 missing MY_VAR from GITHUB_ENV, got %v", step2Env["MY_VAR"])
	}
}

// envSettingMock writes to GITHUB_ENV file to test env persistence
type envSettingMock struct {
	results []executor.Result
	calls   []executor.Config
	callIdx int
}

func (m *envSettingMock) Execute(config executor.Config) (executor.Result, error) {
	m.calls = append(m.calls, config)

	// On first call, write to GITHUB_ENV if it exists
	if m.callIdx == 0 && config.Env["GITHUB_ENV"] != "" {
		if err := os.WriteFile(config.Env["GITHUB_ENV"], []byte("MY_VAR=hello\n"), 0644); err != nil {
			return executor.Result{}, err
		}
	}

	result := executor.Result{ExitCode: 0}
	if m.callIdx < len(m.results) {
		result = m.results[m.callIdx]
	}
	m.callIdx++
	return result, nil
}

func TestRunner_Run_WorkingDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	workflowContent := `
name: Test Workflow
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Default dir
        run: pwd
      - name: Custom dir
        working-directory: subdir
        run: pwd
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	mock := newMockExecutor(
		executor.Result{ExitCode: 0},
		executor.Result{ExitCode: 0},
	)

	runner := NewRunner(mock)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner.SetOutput(stdout, stderr)

	_, err := runner.Run(&RunOptions{
		Workflow:   workflowPath,
		Job:        "test",
		WorkingDir: tmpDir,
	})

	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Check working directories
	if mock.calls[0].WorkingDir != tmpDir {
		t.Errorf("Step 1 working dir = %v, want %v", mock.calls[0].WorkingDir, tmpDir)
	}
	expectedSubDir := filepath.Join(tmpDir, "subdir")
	if mock.calls[1].WorkingDir != expectedSubDir {
		t.Errorf("Step 2 working dir = %v, want %v", mock.calls[1].WorkingDir, expectedSubDir)
	}
}

func TestRunner_Run_OutputFormatting(t *testing.T) {
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	workflowContent := `
name: Test Workflow
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Test Step
        run: echo "hello"
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	mock := newMockExecutor(
		executor.Result{ExitCode: 0, Stdout: "hello\n"},
	)

	runner := NewRunner(mock)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner.SetOutput(stdout, stderr)

	_, err := runner.Run(&RunOptions{
		Workflow:   workflowPath,
		Job:        "test",
		WorkingDir: tmpDir,
	})

	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	output := stdout.String()
	// Check for group markers
	if !strings.Contains(output, "::group::Test Step") {
		t.Error("Output should contain group start marker")
	}
	if !strings.Contains(output, "::endgroup::") {
		t.Error("Output should contain group end marker")
	}
	if !strings.Contains(output, "hello") {
		t.Error("Output should contain step output")
	}
}

func TestRunner_Run_InvalidWorkflow(t *testing.T) {
	runner := NewRunner(executor.NewHostExecutor())

	_, err := runner.Run(&RunOptions{
		Workflow:   "/nonexistent/workflow.yml",
		Job:        "test",
		WorkingDir: "/tmp",
	})

	if err == nil {
		t.Error("Expected error for invalid workflow path")
	}
}

func TestRunner_Run_InvalidJob(t *testing.T) {
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	workflowContent := `
name: Test Workflow
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - run: echo "test"
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	runner := NewRunner(executor.NewHostExecutor())

	_, err := runner.Run(&RunOptions{
		Workflow:   workflowPath,
		Job:        "nonexistent",
		WorkingDir: tmpDir,
	})

	if err == nil {
		t.Error("Expected error for invalid job ID")
	}
}

func TestRunner_Run_AllJobs(t *testing.T) {
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	workflowContent := `
name: Test Workflow
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Build
        run: echo "building"
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Test
        run: echo "testing"
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	mock := newMockExecutor(
		executor.Result{ExitCode: 0, Stdout: "building\n"},
		executor.Result{ExitCode: 0, Stdout: "testing\n"},
	)

	runner := NewRunner(mock)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner.SetOutput(stdout, stderr)

	// Run with empty Job to run all jobs
	results, err := runner.Run(&RunOptions{
		Workflow:   workflowPath,
		Job:        "", // Empty means all jobs
		WorkingDir: tmpDir,
	})

	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify both jobs were executed
	if len(results) != 2 {
		t.Errorf("Expected 2 job results, got %d", len(results))
	}

	// Verify all steps were executed (1 per job)
	if len(mock.calls) != 2 {
		t.Errorf("Expected 2 step executions, got %d", len(mock.calls))
	}

	// Verify both jobs succeeded
	for _, result := range results {
		if !result.Success {
			t.Errorf("Job %s should have succeeded", result.JobID)
		}
	}

	// Verify job names are present (in definition order)
	output := stdout.String()
	if !strings.Contains(output, "=== Running job: build ===") {
		t.Error("Output should contain build job header")
	}
	if !strings.Contains(output, "=== Running job: test ===") {
		t.Error("Output should contain test job header")
	}

	// Verify definition order (build before test)
	buildIdx := strings.Index(output, "=== Running job: build ===")
	testIdx := strings.Index(output, "=== Running job: test ===")
	if buildIdx > testIdx {
		t.Error("Jobs should run in definition order (build before test)")
	}
}
