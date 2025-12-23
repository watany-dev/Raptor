package cli

import (
	"flag"
	"fmt"
	"os"
)

// RunOptions contains the options for the run command.
type RunOptions struct {
	// Workflow is the path to the workflow file.
	Workflow string
	// Job is the job ID to run.
	Job string
	// WorkingDir is the working directory for execution.
	WorkingDir string
	// Isolate is always true - workflows run in isolated git worktree for security.
	// This field is kept for internal use but the flag has been removed.
	Isolate bool
	// DryRun shows what would be executed without actually running commands.
	DryRun bool
}

// ParseRunFlags parses command-line flags for the run command.
// It returns the parsed options or an error if validation fails.
func ParseRunFlags(args []string) (*RunOptions, error) {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)

	opts := &RunOptions{}
	fs.StringVar(&opts.Workflow, "workflow", "", "Path to the workflow file (required)")
	fs.StringVar(&opts.Workflow, "w", "", "Path to the workflow file (shorthand)")
	fs.StringVar(&opts.Job, "job", "", "Job ID to run (if omitted, runs all jobs)")
	fs.StringVar(&opts.Job, "j", "", "Job ID to run (shorthand)")
	fs.StringVar(&opts.WorkingDir, "workdir", "", "Working directory for execution")
	fs.StringVar(&opts.WorkingDir, "C", "", "Working directory for execution (shorthand)")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "Show what would be executed without running")
	fs.BoolVar(&opts.DryRun, "n", false, "Show what would be executed (shorthand)")

	// Note: --isolate flag has been removed. Isolated execution is now mandatory for security.
	// All workflows run in isolated git worktrees.

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	// Always enable isolated execution for security
	opts.Isolate = true

	// Validate required fields
	if opts.Workflow == "" {
		return nil, fmt.Errorf("--workflow flag is required")
	}

	// Set default working directory
	if opts.WorkingDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		opts.WorkingDir = wd
	}

	return opts, nil
}

// PrintHelp prints the help message for the CLI.
func PrintHelp() {
	fmt.Println("Usage: raptor <command> [options]")
	fmt.Println("       raptor [options]           (dry-run mode)")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  run      Run a workflow job")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -w, --workflow  Path to the workflow file (required)")
	fmt.Println("  -j, --job       Job ID to run (if omitted, runs all jobs)")
	fmt.Println("  -C, --workdir   Working directory for execution")
	fmt.Println("  -n, --dry-run   Show what would be executed without running")
	fmt.Println()
	fmt.Println("Dry-run mode:")
	fmt.Println("  When called without 'run' command, raptor operates in dry-run mode.")
	fmt.Println("  This shows what would be executed without actually running commands.")
	fmt.Println()
	fmt.Println("Security:")
	fmt.Println("  All workflows run in isolated git worktrees for security.")
	fmt.Println("  See SECURITY.md for details.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  raptor -w ci.yml              # Dry-run: show what would be executed")
	fmt.Println("  raptor run -w ci.yml          # Execute the workflow")
	fmt.Println("  raptor run -w ci.yml -j test  # Execute specific job")
	fmt.Println("  raptor run -w ci.yml -n       # Dry-run with explicit flag")
}
