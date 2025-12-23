package executor

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// getSysEnv returns the cached system environment variables.
// Uses sync.OnceValue (Go 1.21+) for thread-safe lazy initialization.
var getSysEnv = sync.OnceValue(func() []string {
	return os.Environ()
})

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
		// Use cached system environment to avoid repeated os.Environ() calls
		sysEnv := getSysEnv()
		cmd.Env = make([]string, len(sysEnv), len(sysEnv)+len(config.Env))
		copy(cmd.Env, sysEnv)
		// Add custom environment variables
		for key, value := range config.Env {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	// Extract exit code
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		} else {
			// Command failed to start
			return result, err
		}
	}

	return result, nil
}
