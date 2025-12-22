package executor

import (
	"bytes"
	"os"
	"os/exec"
)

// HostExecutor executes commands on the host system using a shell.
type HostExecutor struct{}

// NewHostExecutor creates a new HostExecutor instance.
func NewHostExecutor() *HostExecutor {
	return &HostExecutor{}
}

// Execute runs the given command on the host system.
func (h *HostExecutor) Execute(config Config) (Result, error) {
	cmd := exec.Command("sh", "-c", config.Command)

	// Set working directory if specified
	if config.WorkingDir != "" {
		cmd.Dir = config.WorkingDir
	}

	// Set environment variables
	if len(config.Env) > 0 {
		// Start with current environment
		cmd.Env = os.Environ()
		// Add custom environment variables
		for key, value := range config.Env {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	// Extract exit code
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			// Command failed to start
			return result, err
		}
	}

	return result, nil
}
