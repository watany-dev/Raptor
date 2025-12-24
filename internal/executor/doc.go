// Copyright 2024 watany-dev
// SPDX-License-Identifier: MIT

// Package executor provides command execution for workflow steps.
//
// This package defines the Executor interface and implementations
// for running shell commands. The main implementation is HostExecutor,
// which executes commands on the local host using bash.
//
// The package supports:
//   - Configurable working directory
//   - Custom environment variables
//   - Capturing stdout and stderr
//   - Exit code handling
//
// Example usage:
//
//	exec := executor.NewHostExecutor()
//	result, err := exec.Execute(executor.Config{
//	    Command:    "echo hello",
//	    WorkingDir: "/path/to/dir",
//	    Env:        map[string]string{"FOO": "bar"},
//	})
package executor
