package main

import (
	"fmt"
	"os"

	"github.com/watany-dev/raptor/internal/cli"
	"github.com/watany-dev/raptor/internal/executor"
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
		fmt.Println("raptor version 0.1.0")
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
	result, err := runner.Run(opts)
	if err != nil {
		return err
	}

	if !result.Success {
		// Find the failed step
		for _, stepResult := range result.StepResults {
			if stepResult.ExitCode != 0 {
				return fmt.Errorf("job failed at step %q with exit code %d", stepResult.StepName, stepResult.ExitCode)
			}
		}
	}

	fmt.Println("Job completed successfully")
	return nil
}
