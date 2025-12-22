package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/watany-dev/raptor/internal/executor"
)

func TestRunWithAbsolutePathBlocked(t *testing.T) {
	// Create a temporary git repository
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	// Create a workflow with malicious absolute path
	workflowContent := `
name: Malicious
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Try absolute path
        working-directory: /etc
        run: cat passwd
`
	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	mock := newMockExecutor(
		executor.Result{ExitCode: 0},
	)

	runner := NewRunner(mock)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner.SetOutput(stdout, stderr)

	_, err := runner.Run(&RunOptions{
		Workflow:   workflowPath,
		Job:        "test",
		WorkingDir: tmpDir,
		Isolate:    true,
	})

	// Error should occur due to absolute path
	if err == nil {
		t.Fatal("Expected error for absolute path, got nil")
	}
	if !strings.Contains(err.Error(), "absolute paths are not allowed") {
		t.Errorf("Expected error about absolute paths, got: %v", err)
	}
}

func TestRunWithPathTraversalBlocked(t *testing.T) {
	// Create a temporary git repository
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	// Create a workflow with path traversal attempt
	workflowContent := `
name: Malicious
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Try path traversal
        working-directory: ../../etc
        run: cat passwd
`
	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	mock := newMockExecutor(
		executor.Result{ExitCode: 0},
	)

	runner := NewRunner(mock)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner.SetOutput(stdout, stderr)

	_, err := runner.Run(&RunOptions{
		Workflow:   workflowPath,
		Job:        "test",
		WorkingDir: tmpDir,
		Isolate:    true,
	})

	// Error should occur due to path traversal
	if err == nil {
		t.Fatal("Expected error for path traversal, got nil")
	}
	if !strings.Contains(err.Error(), "cannot traverse outside") {
		t.Errorf("Expected error about path traversal, got: %v", err)
	}
}

func TestRunWithValidRelativePath(t *testing.T) {
	// Create a temporary git repository
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	// Create a workflow with valid relative path
	workflowContent := `
name: Valid
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Valid relative path
        working-directory: subdir
        run: pwd
`
	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	mock := newMockExecutor(
		executor.Result{ExitCode: 0, Stdout: subDir + "\n"},
	)

	runner := NewRunner(mock)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner.SetOutput(stdout, stderr)

	_, err := runner.Run(&RunOptions{
		Workflow:   workflowPath,
		Job:        "test",
		WorkingDir: tmpDir,
		Isolate:    true,
	})

	// Should succeed
	if err != nil {
		t.Fatalf("Expected success for valid relative path, got: %v", err)
	}

	// Verify working directory was set correctly
	if len(mock.calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(mock.calls))
	}
	// Working directory should end with "subdir"
	if !strings.HasSuffix(mock.calls[0].WorkingDir, "subdir") {
		t.Errorf("Working directory should end with 'subdir', got: %s", mock.calls[0].WorkingDir)
	}
}

func TestRunRequiresGitRepository(t *testing.T) {
	// Create a temporary directory (NOT a git repository)
	tmpDir := t.TempDir()

	// Create a workflow file
	workflowContent := `
name: Test
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Test
        run: echo "test"
`
	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	runner := NewRunner(executor.NewHostExecutor())
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner.SetOutput(stdout, stderr)

	_, err := runner.Run(&RunOptions{
		Workflow:   workflowPath,
		Job:        "test",
		WorkingDir: tmpDir,
		Isolate:    true,
	})

	// Error should occur due to not being in a git repository
	if err == nil {
		t.Fatal("Expected error for non-git directory, got nil")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("Expected error about git repository, got: %v", err)
	}
}

func TestRunWithNestedRelativePath(t *testing.T) {
	// Create a temporary git repository
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	// Create nested subdirectory
	nestedDir := filepath.Join(tmpDir, "a", "b", "c")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("Failed to create nested subdir: %v", err)
	}

	// Create a workflow with nested relative path
	workflowContent := `
name: Valid
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Nested relative path
        working-directory: a/b/c
        run: pwd
`
	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	mock := newMockExecutor(
		executor.Result{ExitCode: 0},
	)

	runner := NewRunner(mock)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner.SetOutput(stdout, stderr)

	_, err := runner.Run(&RunOptions{
		Workflow:   workflowPath,
		Job:        "test",
		WorkingDir: tmpDir,
		Isolate:    true,
	})

	// Should succeed
	if err != nil {
		t.Fatalf("Expected success for nested relative path, got: %v", err)
	}
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Test User", "GIT_AUTHOR_EMAIL=test@test.com", "GIT_COMMITTER_NAME=Test User", "GIT_COMMITTER_EMAIL=test@test.com")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user for the repo
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = dir
	_ = cmd.Run()

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = dir
	_ = cmd.Run()

	// Disable GPG signing for tests
	cmd = exec.Command("git", "config", "commit.gpgsign", "false")
	cmd.Dir = dir
	_ = cmd.Run()

	// Create an initial commit so worktree can be created
	dummyFile := filepath.Join(dir, "dummy.txt")
	if err := os.WriteFile(dummyFile, []byte("dummy"), 0644); err != nil {
		t.Fatalf("Failed to write dummy file: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git add: %v", err)
	}

	cmd = exec.Command("git", "-c", "user.name=Test User", "-c", "user.email=test@test.com", "commit", "-m", "initial")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Test User", "GIT_AUTHOR_EMAIL=test@test.com", "GIT_COMMITTER_NAME=Test User", "GIT_COMMITTER_EMAIL=test@test.com")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to git commit: %v, output: %s", err, string(output))
	}
}
