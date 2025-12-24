package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/watany-dev/raptor/internal/envfiles"
	"github.com/watany-dev/raptor/internal/executor"
	"github.com/watany-dev/raptor/internal/expression"
	"github.com/watany-dev/raptor/internal/runtime"
	"github.com/watany-dev/raptor/internal/security"
	"github.com/watany-dev/raptor/internal/workflow"
)

// StepExecutor handles the execution of individual workflow steps.
type StepExecutor struct {
	executor     executor.Executor
	evaluator    *expression.ConditionEvaluator
	stdout       io.Writer
	stderr       io.Writer
	workDir      string
	envFilePath  string
	pathFilePath string
}

// NewStepExecutor creates a new StepExecutor.
func NewStepExecutor(
	exec executor.Executor,
	evaluator *expression.ConditionEvaluator,
	stdout, stderr io.Writer,
	workDir string,
	envFilePath, pathFilePath string,
) *StepExecutor {
	return &StepExecutor{
		executor:     exec,
		evaluator:    evaluator,
		stdout:       stdout,
		stderr:       stderr,
		workDir:      workDir,
		envFilePath:  envFilePath,
		pathFilePath: pathFilePath,
	}
}

// ExecutionContext holds the mutable state during step execution.
type ExecutionContext struct {
	AccumulatedEnv map[string]string
	StepsContext   map[string]*expression.StepContext
	JobSuccess     bool
}

// NewExecutionContext creates a new ExecutionContext with initial environment.
func NewExecutionContext(baseEnv map[string]string) *ExecutionContext {
	return &ExecutionContext{
		AccumulatedEnv: baseEnv,
		StepsContext:   make(map[string]*expression.StepContext),
		JobSuccess:     true,
	}
}

// updateStepContext updates the steps context for condition evaluation.
func (ctx *ExecutionContext) updateStepContext(stepID, outcome string) {
	if stepID != "" {
		ctx.StepsContext[stepID] = &expression.StepContext{
			Outcome:    outcome,
			Conclusion: outcome,
			Outputs:    map[string]string{},
		}
	}
}

// Execute executes a single step and returns the result.
func (se *StepExecutor) Execute(step *workflow.Step, index int, ctx *ExecutionContext) (*StepResult, error) {
	stepName := step.Name
	if stepName == "" {
		stepName = fmt.Sprintf("Step %d", index+1)
	}

	_, _ = fmt.Fprintf(se.stdout, "::group::%s\n", stepName)

	// Merge step-level env
	stepEnv := runtime.MergeEnv(ctx.AccumulatedEnv, step.Env)

	// Evaluate if condition
	// The evaluator handles StrictMode internally:
	// - StrictMode=true + error: returns (false, error)
	// - StrictMode=false + error: returns (true, error) with warning
	shouldRun, err := se.evaluator.EvaluateWithWorkDir(step.If, stepEnv, ctx.StepsContext, ctx.JobSuccess, se.workDir)
	if err != nil {
		if !shouldRun {
			// Strict mode - evaluator returned false, fail the step
			_, _ = fmt.Fprintf(se.stderr, "Error: %v\n", err)
			_, _ = fmt.Fprintf(se.stdout, "::endgroup::\n")
			return nil, fmt.Errorf("condition evaluation failed: %w", err)
		}
		// Permissive mode - evaluator returned true despite error, log warning and continue
		_, _ = fmt.Fprintf(se.stderr, "Warning: %v\n", err)
	}

	if !shouldRun {
		return se.handleSkippedStep(step, index, stepName, ctx)
	}

	return se.executeStep(step, index, stepName, stepEnv, ctx)
}

// handleSkippedStep handles a step that should be skipped due to condition evaluation.
func (se *StepExecutor) handleSkippedStep(step *workflow.Step, index int, stepName string, ctx *ExecutionContext) (*StepResult, error) {
	_, _ = fmt.Fprintf(se.stdout, "Skipping step (condition evaluated to false)\n")
	_, _ = fmt.Fprintf(se.stdout, "::endgroup::\n")

	stepResult := &StepResult{
		StepIndex: index,
		StepName:  stepName,
		StepID:    step.ID,
		ExitCode:  0,
		Skipped:   true,
		Outcome:   "skipped",
	}

	ctx.updateStepContext(step.ID, "skipped")
	return stepResult, nil
}

// executeStep executes the step command and handles the result.
func (se *StepExecutor) executeStep(step *workflow.Step, index int, stepName string, stepEnv map[string]string, ctx *ExecutionContext) (*StepResult, error) {
	// Validate and determine working directory
	workDir := se.workDir
	if step.WorkingDirectory != "" {
		// Validate working directory for security
		if err := security.ValidateWorkingDirectory(step.WorkingDirectory, se.workDir); err != nil {
			return nil, fmt.Errorf("step %q: %w", stepName, err)
		}
		// Path is validated, safe to use
		workDir = filepath.Join(se.workDir, filepath.Clean(step.WorkingDirectory))
	}

	// Execute the step
	execResult, err := se.executor.Execute(executor.Config{
		Command:    step.Run,
		Env:        stepEnv,
		WorkingDir: workDir,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to execute step %q: %w", stepName, err)
	}

	// Print step output
	se.printOutput(execResult)

	// Determine outcome
	outcome := "success"
	if execResult.ExitCode != 0 {
		outcome = "failure"
		ctx.JobSuccess = false
	}

	stepResult := &StepResult{
		StepIndex: index,
		StepName:  stepName,
		StepID:    step.ID,
		ExitCode:  execResult.ExitCode,
		Stdout:    execResult.Stdout,
		Stderr:    execResult.Stderr,
		Skipped:   false,
		Outcome:   outcome,
	}

	ctx.updateStepContext(step.ID, outcome)
	_, _ = fmt.Fprintf(se.stdout, "::endgroup::\n")

	// Update accumulated environment from GITHUB_ENV
	if err := se.updateEnvironmentFromFiles(ctx); err != nil {
		return nil, err
	}

	return stepResult, nil
}

// printOutput prints the step's stdout and stderr.
func (se *StepExecutor) printOutput(execResult executor.Result) {
	if execResult.Stdout != "" {
		_, _ = fmt.Fprint(se.stdout, execResult.Stdout)
		if !strings.HasSuffix(execResult.Stdout, "\n") {
			_, _ = fmt.Fprintln(se.stdout)
		}
	}
	if execResult.Stderr != "" {
		_, _ = fmt.Fprint(se.stderr, execResult.Stderr)
		if !strings.HasSuffix(execResult.Stderr, "\n") {
			_, _ = fmt.Fprintln(se.stderr)
		}
	}
}

// updateEnvironmentFromFiles updates the accumulated environment from GITHUB_ENV and GITHUB_PATH files.
func (se *StepExecutor) updateEnvironmentFromFiles(ctx *ExecutionContext) error {
	// Update accumulated environment from GITHUB_ENV
	newEnv, err := envfiles.ParseEnvFile(se.envFilePath)
	if err != nil {
		// Print detailed security error message
		_, _ = fmt.Fprintln(se.stderr, "")
		_, _ = fmt.Fprintln(se.stderr, "❌ Security Error:")
		_, _ = fmt.Fprintln(se.stderr, err.Error())
		_, _ = fmt.Fprintln(se.stderr, "")
		_, _ = fmt.Fprintln(se.stderr, "This restriction protects your system from potentially malicious workflows.")
		_, _ = fmt.Fprintln(se.stderr, "See: https://github.com/watany-dev/raptor/blob/main/SECURITY.md")
		_, _ = fmt.Fprintln(se.stderr, "")

		return fmt.Errorf("security validation failed: %w", err)
	}
	ctx.AccumulatedEnv = runtime.MergeEnv(ctx.AccumulatedEnv, newEnv)

	// Update PATH from GITHUB_PATH
	newPaths, err := envfiles.ParsePathFile(se.pathFilePath)
	if err != nil {
		return fmt.Errorf("failed to parse GITHUB_PATH: %w", err)
	}
	if len(newPaths) > 0 {
		currentPath := ctx.AccumulatedEnv["PATH"]
		if currentPath == "" {
			currentPath = os.Getenv("PATH")
		}
		ctx.AccumulatedEnv["PATH"] = envfiles.PrependPath(currentPath, newPaths)
	}

	return nil
}
