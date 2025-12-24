// Copyright 2024 watany-dev
// SPDX-License-Identifier: MIT

// Package cli provides the command-line interface for Raptor.
//
// This package handles workflow execution, including:
//   - Parsing command-line flags and options
//   - Loading and validating workflow files
//   - Executing workflow jobs and steps
//   - Managing execution context and environment
//   - Providing dry-run mode for previewing workflows
//
// The main entry point is the Runner type, which orchestrates
// workflow execution using isolated git worktrees for security.
//
// Example usage:
//
//	runner := cli.NewRunner(executor.NewHostExecutor())
//	opts := &cli.RunOptions{
//	    Workflow: ".github/workflows/test.yml",
//	    Job:      "build",
//	}
//	results, err := runner.Run(opts)
package cli
