// Copyright 2024 watany-dev
// SPDX-License-Identifier: MIT

// Package expression provides evaluation of GitHub Actions expressions.
//
// This package implements a parser and evaluator for GitHub Actions
// workflow expressions used in conditional (if) statements. It supports
// the expression syntax documented at:
// https://docs.github.com/en/actions/learn-github-actions/expressions
//
// Supported features:
//
// Status check functions:
//   - success() - true if all previous steps succeeded
//   - failure() - true if any previous step failed
//   - always()  - always returns true
//   - cancelled() - true if workflow was cancelled
//
// String functions:
//   - contains(haystack, needle)
//   - startsWith(string, prefix)
//   - endsWith(string, suffix)
//
// Other functions:
//   - hashFiles(pattern...) - SHA256 hash of matching files
//
// Operators:
//   - Comparison: ==, !=
//   - Logical: &&, ||, !
//
// Context access:
//   - env.VARIABLE_NAME
//   - steps.step_id.outcome
//   - steps.step_id.outputs.name
//
// Example usage:
//
//	evaluator := expression.NewConditionEvaluator()
//	shouldRun, err := evaluator.Evaluate(
//	    "success() && env.DEBUG == 'true'",
//	    env,
//	    stepsContext,
//	    jobSuccess,
//	)
package expression
