package main

import (
	"fmt"
	"os"

	"github.com/watany-dev/raptor/internal/cli"
	"github.com/watany-dev/raptor/internal/executor"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		cli.PrintHelp()
		return nil
	}

	command := args[0]
	cmdArgs := args[1:]

	switch command {
	case "run":
		return runCommand(cmdArgs)
	case "help", "-h", "--help":
		cli.PrintHelp()
		return nil
	case "version", "-v", "--version":
		fmt.Printf("raptor %s (commit: %s, built: %s)\n", version, commit, date)
		return nil
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

func runCommand(args []string) error {
	opts, err := cli.ParseRunFlags(args)
	if err != nil {
		return err
	}

	runner := cli.NewRunner(executor.NewHostExecutor())
	results, err := runner.Run(opts)
	if err != nil {
		return err
	}

	// Check for failures in any job
	for _, result := range results {
		if !result.Success {
			// Find the failed step
			for _, stepResult := range result.StepResults {
				if stepResult.ExitCode != 0 {
					return fmt.Errorf("job %q failed at step %q with exit code %d", result.JobID, stepResult.StepName, stepResult.ExitCode)
				}
			}
		}
	}

	if len(results) == 1 {
		fmt.Println("Job completed successfully")
	} else {
		fmt.Printf("All %d jobs completed successfully\n", len(results))
	}
	return nil
}
