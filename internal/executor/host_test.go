package executor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHostExecutor_Execute_SimpleCommand(t *testing.T) {
	executor := NewHostExecutor()

	config := Config{
		Command: "echo hello",
	}

	result, err := executor.Execute(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	expectedOutput := "hello"
	if strings.TrimSpace(result.Stdout) != expectedOutput {
		t.Errorf("expected stdout %q, got %q", expectedOutput, strings.TrimSpace(result.Stdout))
	}
}

func TestHostExecutor_Execute_ExitCode(t *testing.T) {
	executor := NewHostExecutor()

	tests := []struct {
		name         string
		command      string
		expectedCode int
	}{
		{
			name:         "exit 0",
			command:      "exit 0",
			expectedCode: 0,
		},
		{
			name:         "exit 1",
			command:      "exit 1",
			expectedCode: 1,
		},
		{
			name:         "exit 42",
			command:      "exit 42",
			expectedCode: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Command: tt.command,
			}

			result, err := executor.Execute(config)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.ExitCode != tt.expectedCode {
				t.Errorf("expected exit code %d, got %d", tt.expectedCode, result.ExitCode)
			}
		})
	}
}

func TestHostExecutor_Execute_EnvVars(t *testing.T) {
	executor := NewHostExecutor()

	config := Config{
		Command: "echo $MY_VAR",
		Env: map[string]string{
			"MY_VAR": "test_value",
		},
	}

	result, err := executor.Execute(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	expectedOutput := "test_value"
	if strings.TrimSpace(result.Stdout) != expectedOutput {
		t.Errorf("expected stdout %q, got %q", expectedOutput, strings.TrimSpace(result.Stdout))
	}
}

func TestHostExecutor_Execute_MultipleEnvVars(t *testing.T) {
	executor := NewHostExecutor()

	config := Config{
		Command: "echo $VAR1 $VAR2",
		Env: map[string]string{
			"VAR1": "hello",
			"VAR2": "world",
		},
	}

	result, err := executor.Execute(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	expectedOutput := "hello world"
	if strings.TrimSpace(result.Stdout) != expectedOutput {
		t.Errorf("expected stdout %q, got %q", expectedOutput, strings.TrimSpace(result.Stdout))
	}
}

func TestHostExecutor_Execute_WorkingDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "executor_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	executor := NewHostExecutor()

	config := Config{
		Command:    "pwd",
		WorkingDir: tmpDir,
	}

	result, err := executor.Execute(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	// Resolve symlinks to handle macOS /private/tmp -> /tmp
	expectedDir, _ := filepath.EvalSymlinks(tmpDir)
	actualDir := strings.TrimSpace(result.Stdout)
	actualDir, _ = filepath.EvalSymlinks(actualDir)

	if actualDir != expectedDir {
		t.Errorf("expected working dir %q, got %q", expectedDir, actualDir)
	}
}

func TestHostExecutor_Execute_Stderr(t *testing.T) {
	executor := NewHostExecutor()

	config := Config{
		Command: "echo error >&2",
	}

	result, err := executor.Execute(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	expectedStderr := "error"
	if strings.TrimSpace(result.Stderr) != expectedStderr {
		t.Errorf("expected stderr %q, got %q", expectedStderr, strings.TrimSpace(result.Stderr))
	}
}

func TestHostExecutor_Execute_MultilineScript(t *testing.T) {
	executor := NewHostExecutor()

	config := Config{
		Command: `echo "line1"
echo "line2"
echo "line3"`,
	}

	result, err := executor.Execute(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	expectedLines := []string{"line1", "line2", "line3"}
	actualLines := strings.Split(strings.TrimSpace(result.Stdout), "\n")

	if len(actualLines) != len(expectedLines) {
		t.Fatalf("expected %d lines, got %d", len(expectedLines), len(actualLines))
	}

	for i, expected := range expectedLines {
		if actualLines[i] != expected {
			t.Errorf("line %d: expected %q, got %q", i, expected, actualLines[i])
		}
	}
}
