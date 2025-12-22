package executor

// Result holds the result of a command execution.
type Result struct {
	// ExitCode is the exit code of the executed command.
	ExitCode int
	// Stdout contains the standard output of the command.
	Stdout string
	// Stderr contains the standard error of the command.
	Stderr string
}

// Config holds the configuration for command execution.
type Config struct {
	// Command is the shell command to execute.
	Command string
	// Env contains environment variables for the command.
	Env map[string]string
	// WorkingDir is the working directory for the command.
	WorkingDir string
}

// Executor defines the interface for command execution.
type Executor interface {
	// Execute runs the given command and returns the result.
	Execute(config Config) (Result, error)
}
