// Copyright 2024 watany-dev
// SPDX-License-Identifier: MIT

// Package envfiles provides parsing for GitHub Actions environment files.
//
// GitHub Actions uses special files (GITHUB_ENV, GITHUB_PATH) to allow
// steps to set environment variables and modify PATH for subsequent steps.
// This package handles parsing these files.
//
// Supported formats:
//
// GITHUB_ENV file format:
//
//	KEY=value                     # Simple key-value
//	KEY<<EOF                      # Multiline with heredoc
//	multiline
//	value
//	EOF
//
// GITHUB_PATH file format:
//
//	/path/to/add
//	/another/path
//
// All values are validated for security before use.
package envfiles
