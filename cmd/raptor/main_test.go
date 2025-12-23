package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestGitRepo creates a minimal git repository for testing
func setupTestGitRepo(t *testing.T, dir string) {
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

	// Create and commit a dummy file
	dummyFile := filepath.Join(dir, ".gitkeep")
	if err := os.WriteFile(dummyFile, []byte(""), 0644); err != nil {
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

func TestRun_NoArgs(t *testing.T) {
	err := run([]string{})
	if err != nil {
		t.Errorf("run() with no args should not error, got: %v", err)
	}
}

func TestRun_Help(t *testing.T) {
	testCases := []string{"help", "-h", "--help"}

	for _, arg := range testCases {
		t.Run(arg, func(t *testing.T) {
			err := run([]string{arg})
			if err != nil {
				t.Errorf("run(%q) should not error, got: %v", arg, err)
			}
		})
	}
}

func TestRun_Version(t *testing.T) {
	testCases := []string{"version", "-v", "--version"}

	for _, arg := range testCases {
		t.Run(arg, func(t *testing.T) {
			err := run([]string{arg})
			if err != nil {
				t.Errorf("run(%q) should not error, got: %v", arg, err)
			}
		})
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	err := run([]string{"unknown"})
	if err == nil {
		t.Error("run(unknown) should error")
	}

	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("error should mention unknown command, got: %v", err)
	}
}

func TestRun_RunCommand_MissingWorkflow(t *testing.T) {
	err := run([]string{"run"})
	if err == nil {
		t.Error("run(run) without workflow should error")
	}

	if !strings.Contains(err.Error(), "--workflow flag is required") {
		t.Errorf("error should mention workflow flag, got: %v", err)
	}
}

func TestRun_RunCommand_InvalidWorkflow(t *testing.T) {
	err := run([]string{"run", "-w", "/nonexistent/workflow.yml"})
	if err == nil {
		t.Error("run with nonexistent workflow should error")
	}
}

func TestRun_DryRunMode(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestGitRepo(t, tmpDir)

	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	workflowContent := `
name: Test Workflow
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Test Step
        run: echo "hello"
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Test dry-run mode (flags without "run" command)
	err := run([]string{"-w", workflowPath, "-C", tmpDir})
	if err != nil {
		t.Errorf("run() in dry-run mode should not error, got: %v", err)
	}
}

func TestRun_RunWithDryRunFlag(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestGitRepo(t, tmpDir)

	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	workflowContent := `
name: Test Workflow
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Test Step
        run: echo "hello"
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Test run command with dry-run flag
	err := run([]string{"run", "-w", workflowPath, "-C", tmpDir, "-n"})
	if err != nil {
		t.Errorf("run() with dry-run flag should not error, got: %v", err)
	}
}

func TestRunCommand_Success(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestGitRepo(t, tmpDir)

	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	workflowContent := `
name: Test Workflow
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Test Step
        run: echo "hello"
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Test successful run
	err := runCommand([]string{"-w", workflowPath, "-C", tmpDir}, false)
	if err != nil {
		t.Errorf("runCommand() should not error for successful workflow, got: %v", err)
	}
}

func TestRunCommand_Failure(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestGitRepo(t, tmpDir)

	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	workflowContent := `
name: Test Workflow
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Failing Step
        run: exit 1
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Test failing run
	err := runCommand([]string{"-w", workflowPath, "-C", tmpDir}, false)
	if err == nil {
		t.Error("runCommand() should error for failing workflow")
	}

	if !strings.Contains(err.Error(), "failed at step") {
		t.Errorf("error should mention failed step, got: %v", err)
	}
}

func TestRunCommand_MultipleJobs(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestGitRepo(t, tmpDir)

	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	workflowContent := `
name: Test Workflow
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Build
        run: echo "building"
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Test
        run: echo "testing"
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Test run all jobs
	err := runCommand([]string{"-w", workflowPath, "-C", tmpDir}, false)
	if err != nil {
		t.Errorf("runCommand() should not error for multiple successful jobs, got: %v", err)
	}
}

func TestRunCommand_SpecificJob(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestGitRepo(t, tmpDir)

	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	workflowContent := `
name: Test Workflow
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Build
        run: echo "building"
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Test
        run: echo "testing"
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Test run specific job
	err := runCommand([]string{"-w", workflowPath, "-C", tmpDir, "-j", "build"}, false)
	if err != nil {
		t.Errorf("runCommand() should not error for specific job, got: %v", err)
	}
}

func TestRunCommand_ForceDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestGitRepo(t, tmpDir)

	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	workflowContent := `
name: Test Workflow
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Test Step
        run: echo "hello"
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Test forced dry-run mode
	err := runCommand([]string{"-w", workflowPath, "-C", tmpDir}, true)
	if err != nil {
		t.Errorf("runCommand() with forceDryRun should not error, got: %v", err)
	}
}

func TestRunCommand_InvalidJob(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestGitRepo(t, tmpDir)

	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	workflowContent := `
name: Test Workflow
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Test Step
        run: echo "hello"
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Test with non-existent job
	err := runCommand([]string{"-w", workflowPath, "-C", tmpDir, "-j", "nonexistent"}, false)
	if err == nil {
		t.Error("runCommand() should error for non-existent job")
	}
}

func TestRunCommand_NotGitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't initialize git repo

	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	workflowContent := `
name: Test Workflow
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Test Step
        run: echo "hello"
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Test without git repo
	err := runCommand([]string{"-w", workflowPath, "-C", tmpDir}, false)
	if err == nil {
		t.Error("runCommand() should error when not in git repo")
	}
}

func TestRunCommand_ParseError(t *testing.T) {
	// Test with invalid flag
	err := runCommand([]string{"--invalid-flag"}, false)
	if err == nil {
		t.Error("runCommand() should error for invalid flag")
	}
}
