// Copyright 2024 watany-dev
// SPDX-License-Identifier: MIT

// Package worktree provides isolated git worktree management for secure execution.
//
// This package creates and manages temporary git worktrees used to isolate
// workflow execution from the main repository. This provides security benefits:
//   - Workflows cannot modify the original working directory
//   - Changes made during execution are automatically cleaned up
//   - Each run gets a fresh, isolated environment
//
// Workspaces are created under .raptor/ws-<id>/ in the repository root,
// using detached HEAD to avoid branch conflicts.
//
// Example usage:
//
//	ws, err := worktree.CreateWorkspace(ctx, repoRoot, true)
//	if err != nil {
//	    return err
//	}
//	defer worktree.RemoveWorkspace(ctx, ws)
//
//	// Execute workflow in ws.Path
package worktree
