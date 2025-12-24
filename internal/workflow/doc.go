// Copyright 2024 watany-dev
// SPDX-License-Identifier: MIT

// Package workflow provides parsing and modeling of GitHub Actions workflows.
//
// This package handles loading and parsing GitHub Actions workflow YAML files,
// providing Go types that represent the workflow structure.
//
// Supported workflow elements:
//   - Workflow name and environment variables
//   - Jobs with their configuration
//   - Steps with run commands, conditions, and environment
//   - Working directory configuration
//
// The package preserves job ordering from the YAML file to ensure
// deterministic execution order when running all jobs.
//
// Example usage:
//
//	wf, err := workflow.LoadWorkflowFile(".github/workflows/test.yml")
//	if err != nil {
//	    return err
//	}
//	job, err := workflow.SelectJob(wf, "build")
package workflow
