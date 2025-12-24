// Copyright 2024 watany-dev
// SPDX-License-Identifier: MIT

// Raptor is a local GitHub Actions workflow runner.
//
// Raptor allows developers to test and debug GitHub Actions workflows
// locally before pushing to remote repositories. It executes workflow
// steps in isolated git worktrees for security.
//
// Usage:
//
//	raptor run -w .github/workflows/test.yml -j build
//	raptor -w .github/workflows/test.yml  # dry-run mode
//
// Commands:
//
//	run       Execute a workflow job
//	help      Show help information
//	version   Show version information
//
// Flags for run command:
//
//	-w, --workflow   Path to workflow file (required)
//	-j, --job        Job ID to run (optional, runs all if omitted)
//	-C, --directory  Working directory
//	--dry-run        Show what would be executed without running
//
// For more information, see: https://github.com/watany-dev/raptor
package main
