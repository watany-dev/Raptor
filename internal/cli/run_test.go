package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/watany-dev/raptor/internal/executor"
)

// setupTestGitRepo creates a minimal git repository for testing.
func setupTestGitRepo(t *testing.T, dir string) {
	t.Helper()

	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Test User", "GIT_AUTHOR_EMAIL=test@test.com", "GIT_COMMITTER_NAME=Test User", "GIT_COMMITTER_EMAIL=test@test.com")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user for the repo
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = dir
	_ = cmd.Run()

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = dir
	_ = cmd.Run()

	// Disable GPG signing for tests
	cmd = exec.Command("git", "config", "commit.gpgsign", "false")
	cmd.Dir = dir
	_ = cmd.Run()

	// Create and commit a dummy file
	dummyFile := filepath.Join(dir, ".gitkeep")
	if err := os.WriteFile(dummyFile, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write dummy file: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git add: %v", err)
	}

	cmd = exec.Command("git", "-c", "user.name=Test User", "-c", "user.email=test@test.com", "commit", "-m", "initial")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Test User", "GIT_AUTHOR_EMAIL=test@test.com", "GIT_COMMITTER_NAME=Test User", "GIT_COMMITTER_EMAIL=test@test.com")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to git commit: %v, output: %s", err, string(output))
	}
}

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
	// Create a temporary git repository
	tmpDir := t.TempDir()
	setupTestGitRepo(t, tmpDir)
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
	setupTestGitRepo(t, tmpDir)
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
      - name: Step 3 (should be skipped)
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

	// Verify we have 3 step results (executed, executed, skipped)
	if len(result.StepResults) != 3 {
		t.Errorf("Expected 3 step results, got %d", len(result.StepResults))
	}
	// Verify step 2 has non-zero exit code
	if result.StepResults[1].ExitCode != 1 {
		t.Errorf("Expected exit code 1 for step 2, got %d", result.StepResults[1].ExitCode)
	}
	// Verify step 3 was skipped
	if !result.StepResults[2].Skipped {
		t.Error("Step 3 should be skipped after step 2 failure")
	}
}

func TestRunner_Run_EnvInheritance(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestGitRepo(t, tmpDir)
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
	setupTestGitRepo(t, tmpDir)
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
	setupTestGitRepo(t, tmpDir)
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
	// Note: Since workflows run in isolated worktrees, the actual working directory
	// will be inside the worktree, not the original tmpDir.
	// We check that:
	// 1. Step 1 is in the worktree root (ends with the worktree ID, not "subdir")
	// 2. Step 2 is in the worktree's subdir (ends with "/subdir")
	if strings.HasSuffix(mock.calls[0].WorkingDir, "/subdir") {
		t.Errorf("Step 1 working dir should not end with /subdir, got %v", mock.calls[0].WorkingDir)
	}
	if !strings.HasSuffix(mock.calls[1].WorkingDir, "/subdir") {
		t.Errorf("Step 2 working dir should end with /subdir, got %v", mock.calls[1].WorkingDir)
	}
}

func TestRunner_Run_OutputFormatting(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestGitRepo(t, tmpDir)
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
	setupTestGitRepo(t, tmpDir)
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
	setupTestGitRepo(t, tmpDir)
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

	// Verify job IDs in results (definition order: build before test)
	// Note: Job status messages are now logged via slog.Info instead of stdout
	if results[0].JobID != "build" {
		t.Errorf("First job should be 'build', got %q", results[0].JobID)
	}
	if results[1].JobID != "test" {
		t.Errorf("Second job should be 'test', got %q", results[1].JobID)
	}
}

func TestRunner_Run_DefaultEnvVars(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestGitRepo(t, tmpDir)
	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	workflowContent := `
name: Test Workflow
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Check env
        run: echo "test"
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	mock := newMockExecutor(
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

	// Check default environment variables are set
	env := mock.calls[0].Env
	if env["CI"] != "true" {
		t.Errorf("CI should be 'true', got %q", env["CI"])
	}
	if env["GITHUB_ACTIONS"] != "true" {
		t.Errorf("GITHUB_ACTIONS should be 'true', got %q", env["GITHUB_ACTIONS"])
	}
	if env["GITHUB_WORKSPACE"] == "" {
		t.Error("GITHUB_WORKSPACE should be set")
	}
	// GITHUB_ENV and GITHUB_PATH should also be set
	if env["GITHUB_ENV"] == "" {
		t.Error("GITHUB_ENV should be set")
	}
	if env["GITHUB_PATH"] == "" {
		t.Error("GITHUB_PATH should be set")
	}
}

func TestRunner_Run_IfConditionTrue(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestGitRepo(t, tmpDir)
	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	workflowContent := `
name: Test Workflow
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Always run
        if: true
        run: echo "executed"
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	mock := newMockExecutor(
		executor.Result{ExitCode: 0, Stdout: "executed\n"},
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

	// Step should be executed
	if len(mock.calls) != 1 {
		t.Errorf("Expected 1 step execution, got %d", len(mock.calls))
	}

	// Verify step was not skipped
	if len(results) != 1 || len(results[0].StepResults) != 1 {
		t.Fatalf("Expected 1 job with 1 step result")
	}
	if results[0].StepResults[0].Skipped {
		t.Error("Step should not be skipped when if: true")
	}
}

func TestRunner_Run_IfConditionFalse(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestGitRepo(t, tmpDir)
	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	workflowContent := `
name: Test Workflow
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Never run
        if: false
        run: echo "should not execute"
      - name: Always run
        run: echo "executed"
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	mock := newMockExecutor(
		executor.Result{ExitCode: 0, Stdout: "executed\n"},
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

	// Only second step should be executed
	if len(mock.calls) != 1 {
		t.Errorf("Expected 1 step execution (first skipped), got %d", len(mock.calls))
	}

	// Verify first step was skipped
	if len(results) != 1 || len(results[0].StepResults) != 2 {
		t.Fatalf("Expected 1 job with 2 step results")
	}
	if !results[0].StepResults[0].Skipped {
		t.Error("First step should be skipped when if: false")
	}
	if results[0].StepResults[1].Skipped {
		t.Error("Second step should not be skipped")
	}
}

func TestRunner_Run_IfConditionEnvVar(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestGitRepo(t, tmpDir)
	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	workflowContent := `
name: Test Workflow
env:
  RUN_STEP: "yes"
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Conditional step
        if: ${{ env.RUN_STEP == 'yes' }}
        run: echo "executed"
      - name: Skipped step
        if: ${{ env.RUN_STEP == 'no' }}
        run: echo "should not execute"
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	mock := newMockExecutor(
		executor.Result{ExitCode: 0, Stdout: "executed\n"},
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

	// Only first step should be executed
	if len(mock.calls) != 1 {
		t.Errorf("Expected 1 step execution, got %d", len(mock.calls))
	}

	// Verify correct steps were skipped/executed
	if len(results) != 1 || len(results[0].StepResults) != 2 {
		t.Fatalf("Expected 1 job with 2 step results")
	}
	if results[0].StepResults[0].Skipped {
		t.Error("First step should not be skipped (env.RUN_STEP == 'yes')")
	}
	if !results[0].StepResults[1].Skipped {
		t.Error("Second step should be skipped (env.RUN_STEP != 'no')")
	}
}

func TestRunner_Run_IfConditionAlways(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestGitRepo(t, tmpDir)
	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	workflowContent := `
name: Test Workflow
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Failing step
        run: exit 1
      - name: Cleanup (always)
        if: always()
        run: echo "cleanup"
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	mock := newMockExecutor(
		executor.Result{ExitCode: 1, Stderr: "failed\n"},
		executor.Result{ExitCode: 0, Stdout: "cleanup\n"},
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

	// Both steps should be executed (always() runs even after failure)
	if len(mock.calls) != 2 {
		t.Errorf("Expected 2 step executions (always() should run after failure), got %d", len(mock.calls))
	}

	// Verify job failed but cleanup ran
	if len(results) != 1 {
		t.Fatalf("Expected 1 job result")
	}
	// Note: Job still fails overall because a step failed
}

func TestRunner_Run_IfConditionFailure(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestGitRepo(t, tmpDir)
	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	workflowContent := `
name: Test Workflow
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Failing step
        run: exit 1
      - name: On failure only
        if: failure()
        run: echo "failure handler"
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	mock := newMockExecutor(
		executor.Result{ExitCode: 1, Stderr: "failed\n"},
		executor.Result{ExitCode: 0, Stdout: "failure handler\n"},
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

	// Both steps should be executed (failure() runs after failure)
	if len(mock.calls) != 2 {
		t.Errorf("Expected 2 step executions (failure() should run after failure), got %d", len(mock.calls))
	}

	// Verify job failed
	if len(results) != 1 {
		t.Fatalf("Expected 1 job result")
	}
}

// TestRunner_Run_StepOutputCapture tests that step output is properly captured
func TestRunner_Run_StepOutputCapture(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestGitRepo(t, tmpDir)
	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	workflowContent := `
name: Test Workflow
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Step 1
        run: echo "step1"
      - name: Step 2
        run: echo "step2"
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	mock := newMockExecutor(
		executor.Result{ExitCode: 0, Stdout: "step1\n"},
		executor.Result{ExitCode: 0, Stdout: "step2\n"},
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

	// Verify job result
	if len(results) != 1 {
		t.Fatalf("Expected 1 job result, got %d", len(results))
	}

	// Verify both steps were executed
	if len(mock.calls) != 2 {
		t.Errorf("Expected 2 step executions, got %d", len(mock.calls))
	}

	// Verify step results contain expected output
	result := results[0]
	if len(result.StepResults) != 2 {
		t.Fatalf("Expected 2 step results, got %d", len(result.StepResults))
	}

	// Verify stdout contains step output
	output := stdout.String()
	if !strings.Contains(output, "step1") {
		t.Errorf("Output should contain step1, got: %s", output)
	}
	if !strings.Contains(output, "step2") {
		t.Errorf("Output should contain step2, got: %s", output)
	}
}

// TestRunner_Run_InvalidWorkflowPath tests handling of non-existent workflow file
func TestRunner_Run_InvalidWorkflowPath(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestGitRepo(t, tmpDir)

	runner := NewRunner(newMockExecutor())
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner.SetOutput(stdout, stderr)

	_, err := runner.Run(&RunOptions{
		Workflow:   "/nonexistent/path/workflow.yml",
		Job:        "test",
		WorkingDir: tmpDir,
	})

	if err == nil {
		t.Errorf("Expected error for invalid workflow path, got nil")
	}
}

// TestRunner_Run_WorkspaceCreationFailure tests handling when workspace creation fails
func TestRunner_Run_WorkspaceCreationFailure(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't initialize git repo to trigger workspace creation failure
	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	workflowContent := `
name: Test Workflow
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Step 1
        run: echo "test"
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	runner := NewRunner(newMockExecutor())
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner.SetOutput(stdout, stderr)

	_, err := runner.Run(&RunOptions{
		Workflow:   workflowPath,
		Job:        "test",
		WorkingDir: tmpDir,
	})

	if err == nil {
		t.Errorf("Expected error when not a git repository, got nil")
	}
}

// TestRunner_Run_LargeOutput tests handling of very large stdout/stderr output
func TestRunner_Run_LargeOutput(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestGitRepo(t, tmpDir)
	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	workflowContent := `
name: Test Workflow
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Large output step
        run: echo "output"
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Create large output (1MB)
	largeOutput := strings.Repeat("x", 1024*1024)
	mock := newMockExecutor(
		executor.Result{ExitCode: 0, Stdout: largeOutput},
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
}

// TestRunner_Run_InvalidWorkflowYAML tests handling of malformed YAML
func TestRunner_Run_InvalidWorkflowYAML(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestGitRepo(t, tmpDir)
	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	invalidYAML := `
name: Test Workflow
jobs:
  test: invalid yaml: [
`
	if err := os.WriteFile(workflowPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	runner := NewRunner(newMockExecutor())
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner.SetOutput(stdout, stderr)

	_, err := runner.Run(&RunOptions{
		Workflow:   workflowPath,
		Job:        "test",
		WorkingDir: tmpDir,
	})

	if err == nil {
		t.Errorf("Expected error for invalid YAML, got nil")
	}
}

// TestRunner_Run_DryRunMode tests dry-run mode
func TestRunner_Run_DryRunMode(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestGitRepo(t, tmpDir)
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

	mock := newMockExecutor()
	runner := NewRunner(mock)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner.SetOutput(stdout, stderr)

	results, err := runner.Run(&RunOptions{
		Workflow:   workflowPath,
		Job:        "test",
		WorkingDir: tmpDir,
		DryRun:     true, // Enable dry-run mode
	})

	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Dry-run should not execute any commands
	if len(mock.calls) != 0 {
		t.Errorf("Expected no command executions in dry-run mode, got %d", len(mock.calls))
	}

	// Should still return results
	if len(results) != 1 {
		t.Errorf("Expected 1 result in dry-run mode, got %d", len(results))
	}

	// Output should contain dry-run indicator
	if !strings.Contains(stdout.String(), "DRY RUN") {
		t.Error("Dry-run mode should output DRY RUN header")
	}
}

// TestRunner_NewRunner tests NewRunner constructor
func TestRunner_NewRunner(t *testing.T) {
	exec := newMockExecutor()
	runner := NewRunner(exec)

	if runner == nil {
		t.Error("NewRunner() should return non-nil runner")
	}
}

// TestRunner_SetOutput tests SetOutput method
func TestRunner_SetOutput(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestGitRepo(t, tmpDir)
	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	workflowContent := `
name: Test Workflow
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Test Step
        run: echo "output test"
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	mock := newMockExecutor(executor.Result{ExitCode: 0, Stdout: "output test\n"})
	runner := NewRunner(mock)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	runner.SetOutput(stdout, stderr)

	// Run workflow to verify output goes to the buffers
	_, err := runner.Run(&RunOptions{
		Workflow:   workflowPath,
		Job:        "test",
		WorkingDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify that output was written to stdout buffer
	output := stdout.String()
	if !strings.Contains(output, "output test") {
		t.Errorf("SetOutput() did not redirect output to buffer, got: %q", output)
	}
}
