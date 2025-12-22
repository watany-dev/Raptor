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
// If isolate mode is enabled, it creates a git worktree for isolated execution.
// Returns a cleanup function that should be called when execution is complete.
func (r *Runner) setupRunContext(ctx context.Context, opts *RunOptions) (*runContext, func(), error) {
	noopCleanup := func() {}

	if !opts.Isolate {
		// Non-isolated mode: run directly in the working directory
		sha, _ := util.GitHeadSHA(ctx, opts.WorkingDir)
		ref, _ := util.GitHeadRef(ctx, opts.WorkingDir)
		return &runContext{
			workDir:  opts.WorkingDir,
			repoRoot: opts.WorkingDir,
			sha:      sha,
			ref:      ref,
		}, noopCleanup, nil
	}

	// Isolated mode: create a git worktree
	repoRoot, err := util.FindGitRoot(ctx, opts.WorkingDir)
	if err != nil {
		return nil, noopCleanup, fmt.Errorf("failed to find git root: %w", err)
	}

	ws, err := worktree.CreateWorkspace(ctx, repoRoot)
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

		// Determine working directory (use runCtx.workDir as base)
		workDir := runCtx.workDir
		if step.WorkingDirectory != "" {
			if filepath.IsAbs(step.WorkingDirectory) {
				workDir = step.WorkingDirectory
			} else {
				workDir = filepath.Join(runCtx.workDir, step.WorkingDirectory)
			}
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
			return nil, fmt.Errorf("failed to parse GITHUB_ENV: %w", err)
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
