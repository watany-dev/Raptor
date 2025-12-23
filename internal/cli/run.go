package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/watany-dev/raptor/internal/envfiles"
	"github.com/watany-dev/raptor/internal/executor"
	"github.com/watany-dev/raptor/internal/runtime"
	"github.com/watany-dev/raptor/internal/security"
	"github.com/watany-dev/raptor/internal/util"
	"github.com/watany-dev/raptor/internal/workflow"
	"github.com/watany-dev/raptor/internal/worktree"
)

// Runner handles the execution of workflow jobs.
type Runner struct {
	executor executor.Executor
	stdout   io.Writer
	stderr   io.Writer
}

// NewRunner creates a new Runner with the given executor.
func NewRunner(exec executor.Executor) *Runner {
	return &Runner{
		executor: exec,
		stdout:   os.Stdout,
		stderr:   os.Stderr,
	}
}

// SetOutput sets the output writers for the runner.
func (r *Runner) SetOutput(stdout, stderr io.Writer) {
	r.stdout = stdout
	r.stderr = stderr
}

// StepResult contains the result of a single step execution.
type StepResult struct {
	StepIndex int
	StepName  string
	StepID    string
	ExitCode  int
	Stdout    string
	Stderr    string
	Skipped   bool
	Outcome   string // "success", "failure", or "skipped"
}

// RunResult contains the results of running a job.
type RunResult struct {
	JobID       string
	Success     bool
	StepResults []StepResult
}

// runContext holds the context for a workflow run.
type runContext struct {
	workDir   string
	repoRoot  string
	sha       string
	ref       string
	workspace *worktree.Workspace
}

// Run executes workflow job(s) with the given options.
// If opts.Job is specified, only that job is executed.
// If opts.Job is empty, all jobs in the workflow are executed.
func (r *Runner) Run(opts *RunOptions) ([]*RunResult, error) {
	// Print security warning
	r.printSecurityWarning(opts)

	ctx := context.Background()

	// Load the workflow file
	wf, err := workflow.LoadWorkflowFile(opts.Workflow)
	if err != nil {
		return nil, fmt.Errorf("failed to load workflow: %w", err)
	}

	// Setup run context (worktree if isolate mode, or working directory otherwise)
	runCtx, cleanup, err := r.setupRunContext(ctx, opts)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	// Determine which jobs to run
	var jobIDs []string
	if opts.Job != "" {
		// Run specific job
		jobIDs = []string{opts.Job}
	} else {
		// Run all jobs in definition order
		jobIDs = wf.JobOrder
	}

	var results []*RunResult
	for _, jobID := range jobIDs {
		result, err := r.runJob(wf, jobID, opts, runCtx)
		if err != nil {
			return results, err
		}
		results = append(results, result)
		if !result.Success {
			// Stop on first failure
			return results, nil
		}
	}

	return results, nil
}

// setupRunContext sets up the execution context.
// All workflows run in isolated git worktrees for security.
// Returns a cleanup function that should be called when execution is complete.
func (r *Runner) setupRunContext(ctx context.Context, opts *RunOptions) (*runContext, func(), error) {
	noopCleanup := func() {}

	// All workflows run in isolated git worktrees for security
	repoRoot, err := util.FindGitRoot(ctx, opts.WorkingDir)
	if err != nil {
		return nil, noopCleanup, fmt.Errorf(
			"not a git repository: %w\n"+
				"Raptor requires a git repository for secure isolated execution.\n"+
				"Initialize with: git init",
			err,
		)
	}

	// Pass verified=true since FindGitRoot already confirmed this is a git repository
	ws, err := worktree.CreateWorkspace(ctx, repoRoot, true)
	if err != nil {
		return nil, noopCleanup, fmt.Errorf("failed to create workspace: %w", err)
	}

	fmt.Fprintf(r.stdout, "Created isolated workspace: %s\n", ws.Path)

	// Get git info from original repo
	sha, _ := util.GitHeadSHA(ctx, repoRoot)
	ref, _ := util.GitHeadRef(ctx, repoRoot)

	cleanup := func() {
		if err := worktree.RemoveWorkspace(ctx, ws); err != nil {
			fmt.Fprintf(r.stderr, "Warning: failed to remove workspace: %v\n", err)
		} else {
			fmt.Fprintf(r.stdout, "Cleaned up workspace: %s\n", ws.Path)
		}
	}

	return &runContext{
		workDir:   ws.Path,
		repoRoot:  repoRoot,
		sha:       sha,
		ref:       ref,
		workspace: ws,
	}, cleanup, nil
}

// runJob executes a single job from the workflow.
func (r *Runner) runJob(wf *workflow.WorkflowFile, jobID string, opts *RunOptions, runCtx *runContext) (*RunResult, error) {
	// Select the job
	job, err := workflow.SelectJob(wf, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to select job: %w", err)
	}

	fmt.Fprintf(r.stdout, "=== Running job: %s ===\n", jobID)

	result := &RunResult{
		JobID:       jobID,
		Success:     true,
		StepResults: make([]StepResult, 0, len(job.Steps)),
	}

	// Create temporary directory for environment files
	tmpDir, err := os.MkdirTemp("", "raptor-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	envFilePath := filepath.Join(tmpDir, "GITHUB_ENV")
	pathFilePath := filepath.Join(tmpDir, "GITHUB_PATH")

	// Initialize environment with default GitHub Actions env, then workflow-level env
	defaultEnv := runtime.DefaultBaseEnv(runCtx.workDir, runCtx.sha, runCtx.ref)
	accumulatedEnv := runtime.MergeEnv(defaultEnv, wf.Env, job.Env)

	// Add GITHUB_ENV and GITHUB_PATH paths to env
	accumulatedEnv["GITHUB_ENV"] = envFilePath
	accumulatedEnv["GITHUB_PATH"] = pathFilePath

	// Track step results for condition evaluation (steps context)
	stepsContext := make(map[string]*stepContext)
	jobSuccess := true // Track overall job success for success()/failure() functions

	// Execute steps sequentially
	for i, step := range job.Steps {
		stepName := step.Name
		if stepName == "" {
			stepName = fmt.Sprintf("Step %d", i+1)
		}

		fmt.Fprintf(r.stdout, "::group::%s\n", stepName)

		// Merge step-level env
		stepEnv := runtime.MergeEnv(accumulatedEnv, step.Env)

		// Evaluate if condition
		shouldRun, err := r.evaluateStepCondition(step.If, stepEnv, stepsContext, jobSuccess, runCtx.workDir)
		if err != nil {
			fmt.Fprintf(r.stderr, "Warning: failed to evaluate if condition: %v\n", err)
			// On evaluation error, default to running the step
			shouldRun = true
		}

		if !shouldRun {
			fmt.Fprintf(r.stdout, "Skipping step (condition evaluated to false)\n")
			fmt.Fprintf(r.stdout, "::endgroup::\n")

			stepResult := StepResult{
				StepIndex: i,
				StepName:  stepName,
				StepID:    step.ID,
				ExitCode:  0,
				Skipped:   true,
				Outcome:   "skipped",
			}
			result.StepResults = append(result.StepResults, stepResult)

			// Update steps context for skipped step
			if step.ID != "" {
				stepsContext[step.ID] = &stepContext{
					outcome:    "skipped",
					conclusion: "skipped",
					outputs:    map[string]string{},
				}
			}
			continue
		}

		// Validate and determine working directory
		workDir := runCtx.workDir
		if step.WorkingDirectory != "" {
			// Validate working directory for security
			if err := security.ValidateWorkingDirectory(step.WorkingDirectory, runCtx.workDir); err != nil {
				return nil, fmt.Errorf("step %q: %w", stepName, err)
			}
			// Path is validated, safe to use
			workDir = filepath.Join(runCtx.workDir, filepath.Clean(step.WorkingDirectory))
		}

		// Execute the step
		execResult, err := r.executor.Execute(executor.Config{
			Command:    step.Run,
			Env:        stepEnv,
			WorkingDir: workDir,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to execute step %q: %w", stepName, err)
		}

		// Print step output
		if execResult.Stdout != "" {
			fmt.Fprint(r.stdout, execResult.Stdout)
			if !strings.HasSuffix(execResult.Stdout, "\n") {
				fmt.Fprintln(r.stdout)
			}
		}
		if execResult.Stderr != "" {
			fmt.Fprint(r.stderr, execResult.Stderr)
			if !strings.HasSuffix(execResult.Stderr, "\n") {
				fmt.Fprintln(r.stderr)
			}
		}

		// Determine outcome
		outcome := "success"
		if execResult.ExitCode != 0 {
			outcome = "failure"
			jobSuccess = false
		}

		stepResult := StepResult{
			StepIndex: i,
			StepName:  stepName,
			StepID:    step.ID,
			ExitCode:  execResult.ExitCode,
			Stdout:    execResult.Stdout,
			Stderr:    execResult.Stderr,
			Skipped:   false,
			Outcome:   outcome,
		}
		result.StepResults = append(result.StepResults, stepResult)

		// Update steps context for condition evaluation
		if step.ID != "" {
			stepsContext[step.ID] = &stepContext{
				outcome:    outcome,
				conclusion: outcome,
				outputs:    map[string]string{},
			}
		}

		fmt.Fprintf(r.stdout, "::endgroup::\n")

		// Check if step failed
		if execResult.ExitCode != 0 {
			result.Success = false
			// Continue to allow always()/failure() steps to run
			// The loop will skip non-always/failure steps due to jobSuccess=false
		}

		// Update accumulated environment from GITHUB_ENV
		newEnv, err := envfiles.ParseEnvFile(envFilePath)
		if err != nil {
			// Print detailed security error message
			fmt.Fprintln(r.stderr, "")
			fmt.Fprintln(r.stderr, "❌ Security Error:")
			fmt.Fprintln(r.stderr, err.Error())
			fmt.Fprintln(r.stderr, "")
			fmt.Fprintln(r.stderr, "This restriction protects your system from potentially malicious workflows.")
			fmt.Fprintln(r.stderr, "See: https://github.com/watany-dev/raptor/blob/main/SECURITY.md")
			fmt.Fprintln(r.stderr, "")

			return nil, fmt.Errorf("security validation failed: %w", err)
		}
		accumulatedEnv = runtime.MergeEnv(accumulatedEnv, newEnv)

		// Update PATH from GITHUB_PATH
		newPaths, err := envfiles.ParsePathFile(pathFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse GITHUB_PATH: %w", err)
		}
		if len(newPaths) > 0 {
			currentPath := accumulatedEnv["PATH"]
			if currentPath == "" {
				currentPath = os.Getenv("PATH")
			}
			accumulatedEnv["PATH"] = envfiles.PrependPath(currentPath, newPaths)
		}
	}

	return result, nil
}

// stepContext holds the result of a step for condition evaluation.
type stepContext struct {
	outcome    string            // "success", "failure", or "skipped"
	conclusion string            // "success", "failure", or "skipped"
	outputs    map[string]string // step outputs
}

// evaluateStepCondition evaluates the if condition for a step.
// Returns true if the step should run, false if it should be skipped.
func (r *Runner) evaluateStepCondition(
	condition string,
	env map[string]string,
	stepsContext map[string]*stepContext,
	jobSuccess bool,
	workDir string,
) (bool, error) {
	// If no condition is specified, use default behavior (run if previous steps succeeded)
	if condition == "" {
		return jobSuccess, nil
	}

	// Normalize condition: remove ${{ }} wrapper if present
	cond := strings.TrimSpace(condition)
	if strings.HasPrefix(cond, "${{") && strings.HasSuffix(cond, "}}") {
		cond = strings.TrimSpace(cond[3 : len(cond)-2])
	}

	// Handle boolean literals
	if cond == "true" {
		return true, nil
	}
	if cond == "false" {
		return false, nil
	}

	// Handle status check functions
	if cond == "always()" {
		return true, nil
	}
	if cond == "success()" {
		return jobSuccess, nil
	}
	if cond == "failure()" {
		return !jobSuccess, nil
	}
	if cond == "cancelled()" {
		return false, nil // We don't support cancellation yet
	}

	// Handle env.VAR == 'value' comparisons
	envCompareRegex := regexp.MustCompile(`env\.(\w+)\s*==\s*'([^']*)'`)
	if matches := envCompareRegex.FindStringSubmatch(cond); matches != nil {
		varName := matches[1]
		expectedValue := matches[2]
		actualValue := env[varName]
		return actualValue == expectedValue, nil
	}

	// Handle env.VAR != 'value' comparisons
	envNotEqualRegex := regexp.MustCompile(`env\.(\w+)\s*!=\s*'([^']*)'`)
	if matches := envNotEqualRegex.FindStringSubmatch(cond); matches != nil {
		varName := matches[1]
		expectedValue := matches[2]
		actualValue := env[varName]
		return actualValue != expectedValue, nil
	}

	// Handle steps.ID.outcome == 'value' comparisons
	stepsOutcomeRegex := regexp.MustCompile(`steps\.(\w+)\.outcome\s*==\s*'([^']*)'`)
	if matches := stepsOutcomeRegex.FindStringSubmatch(cond); matches != nil {
		stepID := matches[1]
		expectedOutcome := matches[2]
		if step, ok := stepsContext[stepID]; ok {
			return step.outcome == expectedOutcome, nil
		}
		return false, nil
	}

	// For unsupported expressions, default to running the step (with warning)
	return true, fmt.Errorf("unsupported condition syntax: %s (defaulting to true)", cond)
}

// printSecurityWarning prints a security warning before execution.
func (r *Runner) printSecurityWarning(opts *RunOptions) {
	fmt.Fprintln(r.stderr, "")
	fmt.Fprintln(r.stderr, "⚠️  SECURITY WARNING")
	fmt.Fprintln(r.stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(r.stderr, "This tool executes commands from workflow files with your user privileges.")
	fmt.Fprintln(r.stderr, "Only run workflows from trusted sources.")
	fmt.Fprintln(r.stderr, "")
	fmt.Fprintf(r.stderr, "Workflow: %s\n", opts.Workflow)
	fmt.Fprintln(r.stderr, "Execution: Isolated git worktree (secure mode)")
	fmt.Fprintln(r.stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(r.stderr, "")
}
