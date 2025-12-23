package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/watany-dev/raptor/internal/executor"
	"github.com/watany-dev/raptor/internal/expression"
	"github.com/watany-dev/raptor/internal/workflow"
)

// TestStepExecutor_Execute tests the Execute method of StepExecutor
func TestStepExecutor_Execute(t *testing.T) {
	t.Run("handles evaluation error gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFilePath := filepath.Join(tmpDir, "GITHUB_ENV")
		pathFilePath := filepath.Join(tmpDir, "GITHUB_PATH")

		mock := newMockExecutor(executor.Result{ExitCode: 0, Stdout: "test\n"})
		evaluator := expression.NewConditionEvaluator()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		se := NewStepExecutor(mock, evaluator, stdout, stderr, tmpDir, envFilePath, pathFilePath)

		ctx := NewExecutionContext(map[string]string{})

		// Step with invalid if condition that will cause evaluation warning
		step := &workflow.Step{
			Name: "Test Step",
			If:   "${{ invalid.syntax.here }}",
			Run:  "echo test",
		}

		result, err := se.Execute(step, 0, ctx)
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		// Step should still run on evaluation error (defaults to true)
		if result.Skipped {
			t.Error("Step should not be skipped on evaluation error")
		}
	})

	t.Run("handles step without name", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFilePath := filepath.Join(tmpDir, "GITHUB_ENV")
		pathFilePath := filepath.Join(tmpDir, "GITHUB_PATH")

		mock := newMockExecutor(executor.Result{ExitCode: 0})
		evaluator := expression.NewConditionEvaluator()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		se := NewStepExecutor(mock, evaluator, stdout, stderr, tmpDir, envFilePath, pathFilePath)

		ctx := NewExecutionContext(map[string]string{})

		step := &workflow.Step{
			Run: "echo test",
		}

		result, err := se.Execute(step, 0, ctx)
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		// Step name should default to "Step 1"
		if result.StepName != "Step 1" {
			t.Errorf("StepName = %q, want %q", result.StepName, "Step 1")
		}
	})
}

// TestStepExecutor_handleSkippedStep tests the handleSkippedStep method
func TestStepExecutor_handleSkippedStep(t *testing.T) {
	t.Run("updates steps context for step with ID", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFilePath := filepath.Join(tmpDir, "GITHUB_ENV")
		pathFilePath := filepath.Join(tmpDir, "GITHUB_PATH")

		mock := newMockExecutor()
		evaluator := expression.NewConditionEvaluator()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		se := NewStepExecutor(mock, evaluator, stdout, stderr, tmpDir, envFilePath, pathFilePath)

		ctx := NewExecutionContext(map[string]string{})

		step := &workflow.Step{
			ID:   "my-step",
			Name: "My Step",
			If:   "false",
			Run:  "echo test",
		}

		result, err := se.handleSkippedStep(step, 0, "My Step", ctx)
		if err != nil {
			t.Fatalf("handleSkippedStep() error = %v", err)
		}

		if !result.Skipped {
			t.Error("Result should be marked as skipped")
		}

		if result.Outcome != "skipped" {
			t.Errorf("Outcome = %q, want %q", result.Outcome, "skipped")
		}

		// Check steps context was updated
		stepCtx, exists := ctx.StepsContext["my-step"]
		if !exists {
			t.Fatal("Steps context should contain my-step")
		}

		if stepCtx.Outcome != "skipped" {
			t.Errorf("StepContext.Outcome = %q, want %q", stepCtx.Outcome, "skipped")
		}
	})

	t.Run("handles step without ID", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFilePath := filepath.Join(tmpDir, "GITHUB_ENV")
		pathFilePath := filepath.Join(tmpDir, "GITHUB_PATH")

		mock := newMockExecutor()
		evaluator := expression.NewConditionEvaluator()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		se := NewStepExecutor(mock, evaluator, stdout, stderr, tmpDir, envFilePath, pathFilePath)

		ctx := NewExecutionContext(map[string]string{})

		step := &workflow.Step{
			Name: "My Step",
			If:   "false",
			Run:  "echo test",
		}

		result, err := se.handleSkippedStep(step, 0, "My Step", ctx)
		if err != nil {
			t.Fatalf("handleSkippedStep() error = %v", err)
		}

		if !result.Skipped {
			t.Error("Result should be marked as skipped")
		}

		// Steps context should not be updated for step without ID
		if len(ctx.StepsContext) != 0 {
			t.Errorf("Steps context should be empty, got %d entries", len(ctx.StepsContext))
		}
	})
}

// TestStepExecutor_executeStep tests the executeStep method
func TestStepExecutor_executeStep(t *testing.T) {
	t.Run("returns error for invalid working directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFilePath := filepath.Join(tmpDir, "GITHUB_ENV")
		pathFilePath := filepath.Join(tmpDir, "GITHUB_PATH")

		mock := newMockExecutor()
		evaluator := expression.NewConditionEvaluator()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		se := NewStepExecutor(mock, evaluator, stdout, stderr, tmpDir, envFilePath, pathFilePath)

		ctx := NewExecutionContext(map[string]string{})

		step := &workflow.Step{
			Name:             "Test Step",
			WorkingDirectory: "../../../etc",
			Run:              "echo test",
		}

		_, err := se.executeStep(step, 0, "Test Step", map[string]string{}, ctx)
		if err == nil {
			t.Error("executeStep() expected error for path traversal")
		}
	})

	t.Run("updates steps context for step with ID", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFilePath := filepath.Join(tmpDir, "GITHUB_ENV")
		pathFilePath := filepath.Join(tmpDir, "GITHUB_PATH")

		mock := newMockExecutor(executor.Result{ExitCode: 0, Stdout: "success\n"})
		evaluator := expression.NewConditionEvaluator()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		se := NewStepExecutor(mock, evaluator, stdout, stderr, tmpDir, envFilePath, pathFilePath)

		ctx := NewExecutionContext(map[string]string{})

		step := &workflow.Step{
			ID:   "my-step",
			Name: "Test Step",
			Run:  "echo success",
		}

		result, err := se.executeStep(step, 0, "Test Step", map[string]string{}, ctx)
		if err != nil {
			t.Fatalf("executeStep() error = %v", err)
		}

		if result.Outcome != "success" {
			t.Errorf("Outcome = %q, want %q", result.Outcome, "success")
		}

		// Check steps context was updated
		stepCtx, exists := ctx.StepsContext["my-step"]
		if !exists {
			t.Fatal("Steps context should contain my-step")
		}

		if stepCtx.Outcome != "success" {
			t.Errorf("StepContext.Outcome = %q, want %q", stepCtx.Outcome, "success")
		}
	})

	t.Run("handles step failure correctly", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFilePath := filepath.Join(tmpDir, "GITHUB_ENV")
		pathFilePath := filepath.Join(tmpDir, "GITHUB_PATH")

		mock := newMockExecutor(executor.Result{ExitCode: 1, Stderr: "error\n"})
		evaluator := expression.NewConditionEvaluator()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		se := NewStepExecutor(mock, evaluator, stdout, stderr, tmpDir, envFilePath, pathFilePath)

		ctx := NewExecutionContext(map[string]string{})

		step := &workflow.Step{
			ID:   "failing-step",
			Name: "Failing Step",
			Run:  "exit 1",
		}

		result, err := se.executeStep(step, 0, "Failing Step", map[string]string{}, ctx)
		if err != nil {
			t.Fatalf("executeStep() error = %v", err)
		}

		if result.Outcome != "failure" {
			t.Errorf("Outcome = %q, want %q", result.Outcome, "failure")
		}

		if ctx.JobSuccess {
			t.Error("JobSuccess should be false after step failure")
		}

		// Check steps context was updated
		stepCtx, exists := ctx.StepsContext["failing-step"]
		if !exists {
			t.Fatal("Steps context should contain failing-step")
		}

		if stepCtx.Outcome != "failure" {
			t.Errorf("StepContext.Outcome = %q, want %q", stepCtx.Outcome, "failure")
		}
	})

	t.Run("handles valid custom working directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "subdir")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("Failed to create subdir: %v", err)
		}

		envFilePath := filepath.Join(tmpDir, "GITHUB_ENV")
		pathFilePath := filepath.Join(tmpDir, "GITHUB_PATH")

		mock := newMockExecutor(executor.Result{ExitCode: 0, Stdout: "success\n"})
		evaluator := expression.NewConditionEvaluator()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		se := NewStepExecutor(mock, evaluator, stdout, stderr, tmpDir, envFilePath, pathFilePath)

		ctx := NewExecutionContext(map[string]string{})

		step := &workflow.Step{
			Name:             "Test Step",
			WorkingDirectory: "subdir",
			Run:              "echo success",
		}

		_, err := se.executeStep(step, 0, "Test Step", map[string]string{}, ctx)
		if err != nil {
			t.Fatalf("executeStep() error = %v", err)
		}

		// Verify the working directory was passed correctly
		if len(mock.calls) != 1 {
			t.Fatalf("Expected 1 call, got %d", len(mock.calls))
		}

		if !filepath.IsAbs(mock.calls[0].WorkingDir) {
			t.Error("Working directory should be absolute path")
		}
	})
}

// TestStepExecutor_printOutput tests the printOutput method
func TestStepExecutor_printOutput(t *testing.T) {
	t.Run("prints stdout with trailing newline", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFilePath := filepath.Join(tmpDir, "GITHUB_ENV")
		pathFilePath := filepath.Join(tmpDir, "GITHUB_PATH")

		mock := newMockExecutor()
		evaluator := expression.NewConditionEvaluator()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		se := NewStepExecutor(mock, evaluator, stdout, stderr, tmpDir, envFilePath, pathFilePath)

		se.printOutput(executor.Result{Stdout: "hello\n", Stderr: ""})

		if stdout.String() != "hello\n" {
			t.Errorf("stdout = %q, want %q", stdout.String(), "hello\n")
		}
	})

	t.Run("adds newline to stdout without trailing newline", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFilePath := filepath.Join(tmpDir, "GITHUB_ENV")
		pathFilePath := filepath.Join(tmpDir, "GITHUB_PATH")

		mock := newMockExecutor()
		evaluator := expression.NewConditionEvaluator()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		se := NewStepExecutor(mock, evaluator, stdout, stderr, tmpDir, envFilePath, pathFilePath)

		se.printOutput(executor.Result{Stdout: "hello", Stderr: ""})

		if stdout.String() != "hello\n" {
			t.Errorf("stdout = %q, want %q", stdout.String(), "hello\n")
		}
	})

	t.Run("prints stderr with trailing newline", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFilePath := filepath.Join(tmpDir, "GITHUB_ENV")
		pathFilePath := filepath.Join(tmpDir, "GITHUB_PATH")

		mock := newMockExecutor()
		evaluator := expression.NewConditionEvaluator()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		se := NewStepExecutor(mock, evaluator, stdout, stderr, tmpDir, envFilePath, pathFilePath)

		se.printOutput(executor.Result{Stdout: "", Stderr: "error\n"})

		if stderr.String() != "error\n" {
			t.Errorf("stderr = %q, want %q", stderr.String(), "error\n")
		}
	})

	t.Run("adds newline to stderr without trailing newline", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFilePath := filepath.Join(tmpDir, "GITHUB_ENV")
		pathFilePath := filepath.Join(tmpDir, "GITHUB_PATH")

		mock := newMockExecutor()
		evaluator := expression.NewConditionEvaluator()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		se := NewStepExecutor(mock, evaluator, stdout, stderr, tmpDir, envFilePath, pathFilePath)

		se.printOutput(executor.Result{Stdout: "", Stderr: "error"})

		if stderr.String() != "error\n" {
			t.Errorf("stderr = %q, want %q", stderr.String(), "error\n")
		}
	})

	t.Run("handles empty output", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFilePath := filepath.Join(tmpDir, "GITHUB_ENV")
		pathFilePath := filepath.Join(tmpDir, "GITHUB_PATH")

		mock := newMockExecutor()
		evaluator := expression.NewConditionEvaluator()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		se := NewStepExecutor(mock, evaluator, stdout, stderr, tmpDir, envFilePath, pathFilePath)

		se.printOutput(executor.Result{Stdout: "", Stderr: ""})

		if stdout.Len() != 0 {
			t.Errorf("stdout should be empty, got %q", stdout.String())
		}
		if stderr.Len() != 0 {
			t.Errorf("stderr should be empty, got %q", stderr.String())
		}
	})

	t.Run("prints both stdout and stderr", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFilePath := filepath.Join(tmpDir, "GITHUB_ENV")
		pathFilePath := filepath.Join(tmpDir, "GITHUB_PATH")

		mock := newMockExecutor()
		evaluator := expression.NewConditionEvaluator()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		se := NewStepExecutor(mock, evaluator, stdout, stderr, tmpDir, envFilePath, pathFilePath)

		se.printOutput(executor.Result{Stdout: "output\n", Stderr: "error\n"})

		if stdout.String() != "output\n" {
			t.Errorf("stdout = %q, want %q", stdout.String(), "output\n")
		}
		if stderr.String() != "error\n" {
			t.Errorf("stderr = %q, want %q", stderr.String(), "error\n")
		}
	})
}

// TestStepExecutor_updateEnvironmentFromFiles tests the updateEnvironmentFromFiles method
func TestStepExecutor_updateEnvironmentFromFiles(t *testing.T) {
	t.Run("returns error for security validation failure in env file", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFilePath := filepath.Join(tmpDir, "GITHUB_ENV")
		pathFilePath := filepath.Join(tmpDir, "GITHUB_PATH")

		// Write a file with blocked env var (security error)
		if err := os.WriteFile(envFilePath, []byte("LD_PRELOAD=/malicious/lib.so\n"), 0644); err != nil {
			t.Fatalf("Failed to write env file: %v", err)
		}

		mock := newMockExecutor()
		evaluator := expression.NewConditionEvaluator()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		se := NewStepExecutor(mock, evaluator, stdout, stderr, tmpDir, envFilePath, pathFilePath)

		ctx := NewExecutionContext(map[string]string{})

		err := se.updateEnvironmentFromFiles(ctx)
		if err == nil {
			t.Error("updateEnvironmentFromFiles() expected error for blocked env var")
		}

		// Check that security error message was printed
		stderrStr := stderr.String()
		if !bytes.Contains([]byte(stderrStr), []byte("Security Error")) {
			t.Error("Expected Security Error message in stderr")
		}
	})

	t.Run("handles non-existent env file gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFilePath := filepath.Join(tmpDir, "GITHUB_ENV")
		pathFilePath := filepath.Join(tmpDir, "GITHUB_PATH")

		// Don't create the files - they don't exist

		mock := newMockExecutor()
		evaluator := expression.NewConditionEvaluator()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		se := NewStepExecutor(mock, evaluator, stdout, stderr, tmpDir, envFilePath, pathFilePath)

		ctx := NewExecutionContext(map[string]string{})

		err := se.updateEnvironmentFromFiles(ctx)
		if err != nil {
			t.Errorf("updateEnvironmentFromFiles() error = %v, expected nil for non-existent files", err)
		}
	})

	t.Run("updates accumulated environment from env file", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFilePath := filepath.Join(tmpDir, "GITHUB_ENV")
		pathFilePath := filepath.Join(tmpDir, "GITHUB_PATH")

		// Write a valid env file
		if err := os.WriteFile(envFilePath, []byte("MY_VAR=my_value\n"), 0644); err != nil {
			t.Fatalf("Failed to write env file: %v", err)
		}

		mock := newMockExecutor()
		evaluator := expression.NewConditionEvaluator()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		se := NewStepExecutor(mock, evaluator, stdout, stderr, tmpDir, envFilePath, pathFilePath)

		ctx := NewExecutionContext(map[string]string{"EXISTING": "value"})

		err := se.updateEnvironmentFromFiles(ctx)
		if err != nil {
			t.Fatalf("updateEnvironmentFromFiles() error = %v", err)
		}

		if ctx.AccumulatedEnv["MY_VAR"] != "my_value" {
			t.Errorf("MY_VAR = %q, want %q", ctx.AccumulatedEnv["MY_VAR"], "my_value")
		}

		if ctx.AccumulatedEnv["EXISTING"] != "value" {
			t.Errorf("EXISTING = %q, want %q", ctx.AccumulatedEnv["EXISTING"], "value")
		}
	})

	t.Run("updates PATH from path file", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFilePath := filepath.Join(tmpDir, "GITHUB_ENV")
		pathFilePath := filepath.Join(tmpDir, "GITHUB_PATH")

		// Write a valid path file
		if err := os.WriteFile(pathFilePath, []byte("/new/path/bin\n"), 0644); err != nil {
			t.Fatalf("Failed to write path file: %v", err)
		}

		mock := newMockExecutor()
		evaluator := expression.NewConditionEvaluator()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		se := NewStepExecutor(mock, evaluator, stdout, stderr, tmpDir, envFilePath, pathFilePath)

		ctx := NewExecutionContext(map[string]string{"PATH": "/usr/bin:/bin"})

		err := se.updateEnvironmentFromFiles(ctx)
		if err != nil {
			t.Fatalf("updateEnvironmentFromFiles() error = %v", err)
		}

		expectedPath := "/new/path/bin:/usr/bin:/bin"
		if ctx.AccumulatedEnv["PATH"] != expectedPath {
			t.Errorf("PATH = %q, want %q", ctx.AccumulatedEnv["PATH"], expectedPath)
		}
	})

	t.Run("uses system PATH when context PATH is empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFilePath := filepath.Join(tmpDir, "GITHUB_ENV")
		pathFilePath := filepath.Join(tmpDir, "GITHUB_PATH")

		// Write a valid path file
		if err := os.WriteFile(pathFilePath, []byte("/new/path/bin\n"), 0644); err != nil {
			t.Fatalf("Failed to write path file: %v", err)
		}

		mock := newMockExecutor()
		evaluator := expression.NewConditionEvaluator()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		se := NewStepExecutor(mock, evaluator, stdout, stderr, tmpDir, envFilePath, pathFilePath)

		// Context without PATH
		ctx := NewExecutionContext(map[string]string{})

		err := se.updateEnvironmentFromFiles(ctx)
		if err != nil {
			t.Fatalf("updateEnvironmentFromFiles() error = %v", err)
		}

		// PATH should contain /new/path/bin
		path := ctx.AccumulatedEnv["PATH"]
		if len(path) == 0 {
			t.Error("PATH should not be empty")
		}

		if path[:14] != "/new/path/bin:" && path != "/new/path/bin" {
			t.Errorf("PATH should start with /new/path/bin, got %q", path)
		}
	})

	t.Run("handles empty path file", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFilePath := filepath.Join(tmpDir, "GITHUB_ENV")
		pathFilePath := filepath.Join(tmpDir, "GITHUB_PATH")

		// Write an empty path file
		if err := os.WriteFile(pathFilePath, []byte(""), 0644); err != nil {
			t.Fatalf("Failed to write path file: %v", err)
		}

		mock := newMockExecutor()
		evaluator := expression.NewConditionEvaluator()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		se := NewStepExecutor(mock, evaluator, stdout, stderr, tmpDir, envFilePath, pathFilePath)

		originalPath := "/usr/bin:/bin"
		ctx := NewExecutionContext(map[string]string{"PATH": originalPath})

		err := se.updateEnvironmentFromFiles(ctx)
		if err != nil {
			t.Fatalf("updateEnvironmentFromFiles() error = %v", err)
		}

		// PATH should remain unchanged
		if ctx.AccumulatedEnv["PATH"] != originalPath {
			t.Errorf("PATH = %q, want %q", ctx.AccumulatedEnv["PATH"], originalPath)
		}
	})
}

// TestNewExecutionContext tests the NewExecutionContext function
func TestNewExecutionContext(t *testing.T) {
	t.Run("creates context with initial environment", func(t *testing.T) {
		baseEnv := map[string]string{
			"VAR1": "value1",
			"VAR2": "value2",
		}

		ctx := NewExecutionContext(baseEnv)

		if ctx.AccumulatedEnv["VAR1"] != "value1" {
			t.Errorf("VAR1 = %q, want %q", ctx.AccumulatedEnv["VAR1"], "value1")
		}

		if !ctx.JobSuccess {
			t.Error("JobSuccess should be true initially")
		}

		if ctx.StepsContext == nil {
			t.Error("StepsContext should not be nil")
		}
	})
}

// TestNewStepExecutor tests the NewStepExecutor function
func TestNewStepExecutor(t *testing.T) {
	t.Run("creates step executor with correct fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFilePath := filepath.Join(tmpDir, "GITHUB_ENV")
		pathFilePath := filepath.Join(tmpDir, "GITHUB_PATH")

		mock := newMockExecutor()
		evaluator := expression.NewConditionEvaluator()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		se := NewStepExecutor(mock, evaluator, stdout, stderr, tmpDir, envFilePath, pathFilePath)

		if se.workDir != tmpDir {
			t.Errorf("workDir = %q, want %q", se.workDir, tmpDir)
		}

		if se.envFilePath != envFilePath {
			t.Errorf("envFilePath = %q, want %q", se.envFilePath, envFilePath)
		}

		if se.pathFilePath != pathFilePath {
			t.Errorf("pathFilePath = %q, want %q", se.pathFilePath, pathFilePath)
		}
	})
}

// errorMockExecutor is a mock that returns an error
type errorMockExecutor struct{}

func (m *errorMockExecutor) Execute(_ executor.Config) (executor.Result, error) {
	return executor.Result{}, os.ErrPermission
}

// TestStepExecutor_executeStep_ExecutorError tests the error path when executor.Execute fails
func TestStepExecutor_executeStep_ExecutorError(t *testing.T) {
	tmpDir := t.TempDir()
	envFilePath := filepath.Join(tmpDir, "GITHUB_ENV")
	pathFilePath := filepath.Join(tmpDir, "GITHUB_PATH")

	mock := &errorMockExecutor{}
	evaluator := expression.NewConditionEvaluator()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	se := NewStepExecutor(mock, evaluator, stdout, stderr, tmpDir, envFilePath, pathFilePath)

	ctx := NewExecutionContext(map[string]string{})

	step := &workflow.Step{
		Name: "Test Step",
		Run:  "echo test",
	}

	_, err := se.executeStep(step, 0, "Test Step", map[string]string{}, ctx)
	if err == nil {
		t.Error("executeStep() expected error when executor fails")
	}
}

// TestStepExecutor_updateEnvironmentFromFiles_PathFileError tests ParsePathFile error path
func TestStepExecutor_updateEnvironmentFromFiles_PathFileError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("skipping permission test as root")
	}

	tmpDir := t.TempDir()
	envFilePath := filepath.Join(tmpDir, "GITHUB_ENV")
	pathFilePath := filepath.Join(tmpDir, "GITHUB_PATH")

	// Create path file with no read permission
	if err := os.WriteFile(pathFilePath, []byte("/some/path\n"), 0000); err != nil {
		t.Fatalf("Failed to write path file: %v", err)
	}
	defer func() {
		_ = os.Chmod(pathFilePath, 0644)
	}()

	mock := newMockExecutor()
	evaluator := expression.NewConditionEvaluator()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	se := NewStepExecutor(mock, evaluator, stdout, stderr, tmpDir, envFilePath, pathFilePath)

	ctx := NewExecutionContext(map[string]string{})

	err := se.updateEnvironmentFromFiles(ctx)
	if err == nil {
		t.Error("updateEnvironmentFromFiles() expected error for unreadable path file")
	}
}
