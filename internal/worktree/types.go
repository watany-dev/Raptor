package worktree

// Workspace represents an isolated git worktree for running workflows.
type Workspace struct {
	// RepoRoot is the path to the original repository root.
	RepoRoot string
	// Path is the path to the worktree directory.
	Path string
	// ID is the unique identifier for this workspace.
	ID string
}
