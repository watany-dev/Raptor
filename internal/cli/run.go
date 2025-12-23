package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	ExitCode  int
	Stdout    string
	Stderr    string
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
	ctx := context.Background()

	// Load the workflow file
	wf, err := workflow.LoadWorkflowFile(opts.Workflow)
	if err != nil {
		return nil, fmt.Errorf("failed to load workflow: %w", err)
	}

	// Determine which jobs to run
	var jobIDs []string
	if opts.Job != "" {
		// Run specific job
		jobIDs = []string{opts.Job}
	} else {
		// Run all jobs in definition order
		jobIDs = wf.JobOrder
	}

	// Dry-run mode: show what would be executed without running
	if opts.DryRun {
		return r.dryRun(wf, jobIDs, opts)
	}

	// Print security warning
	r.printSecurityWarning(opts)

	// Setup run context (worktree if isolate mode, or working directory otherwise)
	runCtx, cleanup, err := r.setupRunContext(ctx, opts)
	if err != nil {
		return nil, err
	}
	defer cleanup()

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

	// Execute steps sequentially
	for i, step := range job.Steps {
		stepName := step.Name
		if stepName == "" {
			stepName = fmt.Sprintf("Step %d", i+1)
		}

		fmt.Fprintf(r.stdout, "::group::%s\n", stepName)

		// Merge step-level env
		stepEnv := runtime.MergeEnv(accumulatedEnv, step.Env)

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

		stepResult := StepResult{
			StepIndex: i,
			StepName:  stepName,
			ExitCode:  execResult.ExitCode,
			Stdout:    execResult.Stdout,
			Stderr:    execResult.Stderr,
		}
		result.StepResults = append(result.StepResults, stepResult)

		fmt.Fprintf(r.stdout, "::endgroup::\n")

		// Check if step failed
		if execResult.ExitCode != 0 {
			result.Success = false
			return result, nil
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

// dryRun shows what would be executed without actually running commands.
func (r *Runner) dryRun(wf *workflow.WorkflowFile, jobIDs []string, opts *RunOptions) ([]*RunResult, error) {
	fmt.Fprintln(r.stdout, "")
	fmt.Fprintln(r.stdout, "🔍 DRY RUN MODE")
	fmt.Fprintln(r.stdout, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintf(r.stdout, "Workflow: %s\n", opts.Workflow)
	if wf.Name != "" {
		fmt.Fprintf(r.stdout, "Name: %s\n", wf.Name)
	}
	fmt.Fprintln(r.stdout, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(r.stdout, "")

	var results []*RunResult

	for _, jobID := range jobIDs {
		job, err := workflow.SelectJob(wf, jobID)
		if err != nil {
			return nil, fmt.Errorf("failed to select job: %w", err)
		}

		fmt.Fprintf(r.stdout, "📋 Job: %s\n", jobID)
		if job.Name != "" && job.Name != jobID {
			fmt.Fprintf(r.stdout, "   Name: %s\n", job.Name)
		}
		if job.RunsOn != "" {
			fmt.Fprintf(r.stdout, "   Runs-on: %s\n", job.RunsOn)
		}
		fmt.Fprintln(r.stdout, "")

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

			fmt.Fprintf(r.stdout, "   [%d] %s\n", i+1, stepName)
			if step.WorkingDirectory != "" {
				fmt.Fprintf(r.stdout, "       Working directory: %s\n", step.WorkingDirectory)
			}
			if len(step.Env) > 0 {
				fmt.Fprintf(r.stdout, "       Environment: %d variable(s)\n", len(step.Env))
			}
			if step.Run != "" {
				// Show the command, indented
				lines := strings.Split(strings.TrimSpace(step.Run), "\n")
				fmt.Fprintln(r.stdout, "       Command:")
				for _, line := range lines {
					fmt.Fprintf(r.stdout, "         %s\n", line)
				}
			}
			fmt.Fprintln(r.stdout, "")

			result.StepResults = append(result.StepResults, StepResult{
				StepIndex: i,
				StepName:  stepName,
				ExitCode:  0,
			})
		}

		results = append(results, result)
	}

	fmt.Fprintln(r.stdout, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(r.stdout, "To execute this workflow, use: raptor run -w", opts.Workflow)
	fmt.Fprintln(r.stdout, "")

	return results, nil
}
