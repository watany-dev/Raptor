// Copyright 2024 watany-dev
// SPDX-License-Identifier: MIT

// Package security provides security validation for workflow execution.
//
// This package protects against common security vulnerabilities that could
// be exploited through malicious workflow files. It validates:
//
// Environment variables:
//   - Blocks dangerous variables (LD_PRELOAD, BASH_ENV, etc.)
//   - Validates variable name format
//   - Limits value length to prevent DoS
//   - Rejects null bytes in values
//
// Working directories:
//   - Blocks absolute paths
//   - Prevents path traversal (../) outside workspace
//   - Ensures execution stays within repository bounds
//
// Blocked environment variables include:
//   - Dynamic linker: LD_PRELOAD, LD_LIBRARY_PATH, DYLD_*
//   - Shell hooks: BASH_ENV, ENV, PROMPT_COMMAND
//   - Git internals: GIT_DIR, GIT_WORK_TREE
//   - Command hijacking: PAGER, EDITOR, VISUAL
//
// All workflow inputs should be validated through this package
// before use to ensure safe execution.
package security
