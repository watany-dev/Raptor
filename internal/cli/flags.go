package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
)

// RunOptions contains the options for the run command.
type RunOptions struct {
	Workflow       string // Path to the workflow file
	Job            string // Job ID to run (if omitted, runs all jobs)
	WorkingDir     string // Working directory for execution
	DryRun         bool   // Show what would be executed without running
	IgnoreIfErrors bool   // Ignore condition evaluation errors and run steps (legacy behavior)
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
	fs.BoolVar(&opts.IgnoreIfErrors, "ignore-if-errors", false, "Ignore condition evaluation errors and run steps (legacy behavior)")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

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

// PrintHelp prints the help message for the CLI to stdout.
func PrintHelp() {
	FprintHelp(os.Stdout)
}

// FprintHelp writes the help message for the CLI to the specified writer.
func FprintHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: raptor <command> [options]")
	fmt.Fprintln(w, "       raptor [options]           (dry-run mode)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  run      Run a workflow job")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  -w, --workflow         Path to the workflow file (required)")
	fmt.Fprintln(w, "  -j, --job              Job ID to run (if omitted, runs all jobs)")
	fmt.Fprintln(w, "  -C, --workdir          Working directory for execution")
	fmt.Fprintln(w, "  -n, --dry-run          Show what would be executed without running")
	fmt.Fprintln(w, "  --ignore-if-errors     Ignore condition evaluation errors (legacy mode)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Dry-run mode:")
	fmt.Fprintln(w, "  When called without 'run' command, raptor operates in dry-run mode.")
	fmt.Fprintln(w, "  This shows what would be executed without actually running commands.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Security:")
	fmt.Fprintln(w, "  All workflows run in isolated git worktrees for security.")
	fmt.Fprintln(w, "  See SECURITY.md for details.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  raptor -w ci.yml              # Dry-run: show what would be executed")
	fmt.Fprintln(w, "  raptor run -w ci.yml          # Execute the workflow")
	fmt.Fprintln(w, "  raptor run -w ci.yml -j test  # Execute specific job")
	fmt.Fprintln(w, "  raptor run -w ci.yml -n       # Dry-run with explicit flag")
}
