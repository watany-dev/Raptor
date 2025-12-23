package executor

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"sync"

	"github.com/watany-dev/raptor/internal/security"
)

// HostExecutor executes commands on the host system using a shell.
type HostExecutor struct {
	cachedSysEnv []string
	once         sync.Once
}

// NewHostExecutor creates a new HostExecutor instance.
func NewHostExecutor() *HostExecutor {
	return &HostExecutor{}
}

// getCachedSysEnv returns the cached system environment variables.
// The cache is populated on first call using sync.Once for thread safety.
// Sensitive environment variables (credentials, tokens, etc.) are filtered out
// to prevent credential leakage to workflow commands.
func (h *HostExecutor) getCachedSysEnv() []string {
	h.once.Do(func() {
		h.cachedSysEnv = security.FilterSensitiveEnvVars(os.Environ())
	})
	return h.cachedSysEnv
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
		sysEnv := h.getCachedSysEnv()
		cmd.Env = make([]string, len(sysEnv), len(sysEnv)+len(config.Env))
		copy(cmd.Env, sysEnv)
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
