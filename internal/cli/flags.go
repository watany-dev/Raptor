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

// PrintHelp prints the help message for the CLI.
func PrintHelp() {
	fmt.Println("Usage: raptor <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  run      Run a workflow job")
	fmt.Println()
	fmt.Println("Run options:")
	fmt.Println("  -w, --workflow  Path to the workflow file (required)")
	fmt.Println("  -j, --job       Job ID to run (if omitted, runs all jobs)")
	fmt.Println("  -C, --workdir   Working directory for execution")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  raptor run --workflow .github/workflows/ci.yml --job build")
	fmt.Println("  raptor run -w ci.yml -j test")
	fmt.Println("  raptor run -w ci.yml  # Runs all jobs in the workflow")
}
