package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/watany-dev/raptor/internal/executor"
	"github.com/watany-dev/raptor/internal/expression"
	"github.com/watany-dev/raptor/internal/runtime"
	"github.com/watany-dev/raptor/internal/util"
	"github.com/watany-dev/raptor/internal/workflow"
	"github.com/watany-dev/raptor/internal/worktree"
)

// Runner handles the execution of workflow jobs.
type Runner struct {
	executor  executor.Executor
	evaluator *expression.ConditionEvaluator
	stdout    io.Writer
	stderr    io.Writer
}

// NewRunner creates a new Runner with the given executor.
func NewRunner(exec executor.Executor) *Runner {
	return &Runner{
		executor:  exec,
		evaluator: expression.NewConditionEvaluator(),
		stdout:    os.Stdout,
		stderr:    os.Stderr,
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
	ctx := context.Background()

	// Configure evaluator based on flags
	r.evaluator.StrictMode = !opts.IgnoreIfErrors

	// Load the workflow file
	wf, err := workflow.LoadWorkflowFile(opts.Workflow)
	if err != nil {
		return nil, fmt.Errorf("failed to load workflow: %w", err)
	}

	// Determine which jobs to run
	jobIDs := r.determineJobIDs(wf, opts)

	// Dry-run mode: show what would be executed without running
	if opts.DryRun {
		formatter := NewDryRunFormatter(r.stdout)
		return formatter.Format(wf, jobIDs, opts.Workflow)
	}

	// Print security warning
	r.printSecurityWarning(opts)

	runCtx, cleanup, err := r.setupRunContext(ctx, opts)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	return r.executeJobs(wf, jobIDs, opts, runCtx)
}

// determineJobIDs determines which jobs to run based on options.
func (r *Runner) determineJobIDs(wf *workflow.WorkflowFile, opts *RunOptions) []string {
	if opts.Job != "" {
		return []string{opts.Job}
	}
	return wf.JobOrder
}

// executeJobs executes the specified jobs sequentially.
func (r *Runner) executeJobs(wf *workflow.WorkflowFile, jobIDs []string, opts *RunOptions, runCtx *runContext) ([]*RunResult, error) {
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

// setupRunContext sets up an isolated git worktree for secure execution.
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

	slog.Info("created isolated workspace", "path", ws.Path)

	// Get git info from original repo
	sha, _ := util.GitHeadSHA(ctx, repoRoot)
	ref, _ := util.GitHeadRef(ctx, repoRoot)

	cleanup := func() {
		if err := worktree.RemoveWorkspace(ctx, ws); err != nil {
			slog.Warn("failed to remove workspace", "path", ws.Path, "error", err)
		} else {
			slog.Info("cleaned up workspace", "path", ws.Path)
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

	slog.Info("running job", "job_id", jobID)

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

	// Create execution context
	execCtx := NewExecutionContext(accumulatedEnv)

	// Create step executor
	stepExecutor := NewStepExecutor(
		r.executor,
		r.evaluator,
		r.stdout,
		r.stderr,
		runCtx.workDir,
		envFilePath,
		pathFilePath,
	)

	// Execute steps sequentially
	for i, step := range job.Steps {
		stepResult, err := stepExecutor.Execute(&step, i, execCtx)
		if err != nil {
			return nil, err
		}

		result.StepResults = append(result.StepResults, *stepResult)

		// Check if step failed
		if stepResult.ExitCode != 0 {
			result.Success = false
			// Continue to allow always()/failure() steps to run
		}
	}

	return result, nil
}

// printSecurityWarning prints a security warning before execution.
func (r *Runner) printSecurityWarning(opts *RunOptions) {
	slog.Warn("security warning: executing workflow with user privileges",
		"workflow", opts.Workflow,
		"execution_mode", "isolated git worktree",
		"note", "only run workflows from trusted sources",
	)
}
