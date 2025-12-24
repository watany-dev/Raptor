// Copyright 2024 watany-dev
// SPDX-License-Identifier: MIT

// Package runtime provides runtime environment utilities for workflow execution.
//
// This package handles environment variable management, including:
//   - Merging environment variables from workflow, job, and step levels
//   - Providing default GitHub Actions environment variables
//
// The merge order follows GitHub Actions behavior where later definitions
// override earlier ones: workflow -> job -> step.
//
// Default environment variables provided:
//   - CI=true
//   - GITHUB_ACTIONS=true
//   - GITHUB_WORKSPACE - path to the workspace
//   - GITHUB_SHA - current commit SHA
//   - GITHUB_REF - current git ref
package runtime
