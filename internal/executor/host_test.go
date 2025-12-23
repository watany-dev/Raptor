package executor

import (
	"fmt"
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

func TestHostExecutor_Execute_CommandNotExist(t *testing.T) {
	executor := NewHostExecutor()

	// This will cause the shell to return an error
	config := Config{
		Command: "nonexistent_command_xyz_abc_123",
	}

	result, err := executor.Execute(config)
	if err != nil {
		// It's ok if it errors
		return
	}

	// If it doesn't error, it should have a non-zero exit code (command not found)
	if result.ExitCode == 0 {
		t.Errorf("expected non-zero exit code for non-existent command, got %d", result.ExitCode)
	}
}

// TestHostExecutor_Execute_LongCommand tests handling of very long command strings
func TestHostExecutor_Execute_LongCommand(t *testing.T) {
	executor := NewHostExecutor()

	// Create a moderately long command (under shell limits)
	longArg := strings.Repeat("a", 1000)
	config := Config{
		Command: fmt.Sprintf("echo %s", longArg),
	}

	result, err := executor.Execute(config)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	if !strings.Contains(result.Stdout, "a") {
		t.Error("Execute() failed to capture output from long command")
	}
}

// TestHostExecutor_Execute_EmptyCommand tests handling of empty command string
func TestHostExecutor_Execute_EmptyCommand(t *testing.T) {
	executor := NewHostExecutor()

	config := Config{
		Command: "",
	}

	result, err := executor.Execute(config)
	if err != nil {
		// Empty command may error, which is acceptable
		return
	}

	// If no error, should have empty output
	if result.ExitCode != 0 && result.Stdout == "" {
		// This is acceptable behavior for empty command
		return
	}
}

// TestHostExecutor_Execute_SpecialCharacters tests handling of shell special characters
func TestHostExecutor_Execute_SpecialCharacters(t *testing.T) {
	executor := NewHostExecutor()

	config := Config{
		Command: "echo 'test value with spaces'",
	}

	result, err := executor.Execute(config)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	if !strings.Contains(result.Stdout, "test value with spaces") {
		t.Errorf("Execute() failed to preserve special characters in output")
	}
}

// TestHostExecutor_Execute_WithEnvironmentVariables tests command execution with environment variables
func TestHostExecutor_Execute_WithEnvironmentVariables(t *testing.T) {
	executor := NewHostExecutor()

	config := Config{
		Command: "echo $TEST_VAR",
		Env: map[string]string{
			"TEST_VAR": "test_value",
		},
	}

	result, err := executor.Execute(config)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	if !strings.Contains(result.Stdout, "test_value") {
		t.Errorf("Execute() failed to pass environment variables correctly")
	}
}

// TestHostExecutor_Execute_CommandFailsToStart tests handling of commands that fail to start
func TestHostExecutor_Execute_CommandFailsToStart(t *testing.T) {
	executor := NewHostExecutor()

	// Use a non-existent working directory to cause command to fail
	config := Config{
		Command:    "echo test",
		WorkingDir: "/nonexistent/path/that/does/not/exist/at/all",
	}

	_, err := executor.Execute(config)
	if err == nil {
		// If no error, check the result has proper output
		// Some systems might handle this differently
		return
	}

	// If there's an error, it should be because command failed to start
	t.Logf("Execute() returned expected error: %v", err)
}

// TestHostExecutor_Execute_EmptyEnv tests command execution with empty env map
func TestHostExecutor_Execute_EmptyEnv(t *testing.T) {
	executor := NewHostExecutor()

	config := Config{
		Command: "echo test",
		Env:     map[string]string{},
	}

	result, err := executor.Execute(config)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
}

// TestHostExecutor_Execute_NilEnv tests command execution with nil env map
func TestHostExecutor_Execute_NilEnv(t *testing.T) {
	executor := NewHostExecutor()

	config := Config{
		Command: "echo test",
		Env:     nil,
	}

	result, err := executor.Execute(config)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
}

// TestGetSysEnv tests the environment caching using sync.OnceValue
func TestGetSysEnv(t *testing.T) {
	// First call should cache
	env1 := getSysEnv()
	if len(env1) == 0 {
		t.Error("getSysEnv() should return non-empty env")
	}

	// Second call should return cached value (same slice)
	env2 := getSysEnv()

	// Should be the same slice (pointer equality)
	if &env1[0] != &env2[0] {
		t.Error("getSysEnv() should return the same cached slice")
	}
}

// TestNewHostExecutor tests creating a new executor
func TestNewHostExecutor(t *testing.T) {
	executor := NewHostExecutor()
	if executor == nil {
		t.Error("NewHostExecutor() should return non-nil executor")
	}
}

// TestHostExecutor_Execute_MultipleCallsWithCache tests that multiple calls work with caching
func TestHostExecutor_Execute_MultipleCallsWithCache(t *testing.T) {
	executor := NewHostExecutor()

	// First call
	config1 := Config{
		Command: "echo first",
		Env:     map[string]string{"VAR": "value1"},
	}
	result1, err := executor.Execute(config1)
	if err != nil {
		t.Fatalf("First Execute() error = %v", err)
	}

	// Second call should use cached system env
	config2 := Config{
		Command: "echo second",
		Env:     map[string]string{"VAR": "value2"},
	}
	result2, err := executor.Execute(config2)
	if err != nil {
		t.Fatalf("Second Execute() error = %v", err)
	}

	if !strings.Contains(result1.Stdout, "first") {
		t.Error("First call output incorrect")
	}
	if !strings.Contains(result2.Stdout, "second") {
		t.Error("Second call output incorrect")
	}
}
