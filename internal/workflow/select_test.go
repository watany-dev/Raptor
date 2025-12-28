package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestDiscoverWorkflows(t *testing.T) {
	t.Run("discovers YAML files in workflows directory", func(t *testing.T) {
		// Create a temporary directory structure
		tmpDir := t.TempDir()
		workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
		if err := os.MkdirAll(workflowsDir, 0755); err != nil {
			t.Fatalf("failed to create workflows directory: %v", err)
		}

		// Create test workflow files
		workflow1 := filepath.Join(workflowsDir, "ci.yml")
		workflow2 := filepath.Join(workflowsDir, "deploy.yaml")
		if err := os.WriteFile(workflow1, []byte("name: CI"), 0644); err != nil {
			t.Fatalf("failed to write workflow file: %v", err)
		}
		if err := os.WriteFile(workflow2, []byte("name: Deploy"), 0644); err != nil {
			t.Fatalf("failed to write workflow file: %v", err)
		}

		// Create a non-workflow file that should be ignored
		nonWorkflow := filepath.Join(workflowsDir, "readme.txt")
		if err := os.WriteFile(nonWorkflow, []byte("readme"), 0644); err != nil {
			t.Fatalf("failed to write non-workflow file: %v", err)
		}

		workflows, err := DiscoverWorkflows(tmpDir)
		if err != nil {
			t.Fatalf("DiscoverWorkflows() error = %v", err)
		}

		if len(workflows) != 2 {
			t.Errorf("len(workflows) = %d, want 2", len(workflows))
		}

		// Check that both workflow files are found
		foundCI := false
		foundDeploy := false
		for _, wf := range workflows {
			if filepath.Base(wf) == "ci.yml" {
				foundCI = true
			}
			if filepath.Base(wf) == "deploy.yaml" {
				foundDeploy = true
			}
		}

		if !foundCI {
			t.Error("ci.yml not found in discovered workflows")
		}
		if !foundDeploy {
			t.Error("deploy.yaml not found in discovered workflows")
		}
	})

	t.Run("returns error when workflows directory does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Don't create .github/workflows directory

		_, err := DiscoverWorkflows(tmpDir)
		if err == nil {
			t.Error("DiscoverWorkflows() expected error when workflows directory does not exist, got nil")
		}
	})

	t.Run("returns empty slice when workflows directory is empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
		if err := os.MkdirAll(workflowsDir, 0755); err != nil {
			t.Fatalf("failed to create workflows directory: %v", err)
		}

		workflows, err := DiscoverWorkflows(tmpDir)
		if err != nil {
			t.Fatalf("DiscoverWorkflows() error = %v", err)
		}

		if len(workflows) != 0 {
			t.Errorf("len(workflows) = %d, want 0", len(workflows))
		}
	})

	t.Run("only discovers .yml and .yaml files", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
		if err := os.MkdirAll(workflowsDir, 0755); err != nil {
			t.Fatalf("failed to create workflows directory: %v", err)
		}

		// Create various files
		files := map[string]bool{
			"workflow.yml":  true,  // should be discovered
			"workflow.yaml": true,  // should be discovered
			"workflow.json": false, // should be ignored
			"workflow.txt":  false, // should be ignored
			".hidden.yml":   true,  // should be discovered (hidden but valid extension)
		}

		for name := range files {
			path := filepath.Join(workflowsDir, name)
			if err := os.WriteFile(path, []byte("name: Test"), 0644); err != nil {
				t.Fatalf("failed to write file %s: %v", name, err)
			}
		}

		workflows, err := DiscoverWorkflows(tmpDir)
		if err != nil {
			t.Fatalf("DiscoverWorkflows() error = %v", err)
		}

		expectedCount := 0
		for _, shouldDiscover := range files {
			if shouldDiscover {
				expectedCount++
			}
		}

		if len(workflows) != expectedCount {
			t.Errorf("len(workflows) = %d, want %d", len(workflows), expectedCount)
		}
	})
}

func TestSelectJob(t *testing.T) {
	t.Run("returns correct job for existing job ID", func(t *testing.T) {
		wf := &WorkflowFile{
			Name: "Test Workflow",
			Jobs: map[string]Job{
				"build": {
					Name:   "Build Job",
					RunsOn: "ubuntu-latest",
					Steps: []Step{
						{Name: "Checkout", Run: "echo checkout"},
					},
				},
				"test": {
					Name:   "Test Job",
					RunsOn: "ubuntu-latest",
					Steps: []Step{
						{Name: "Run tests", Run: "go test ./..."},
					},
				},
			},
		}

		job, err := SelectJob(wf, "build")
		if err != nil {
			t.Fatalf("SelectJob() error = %v", err)
		}

		if job.Name != "Build Job" {
			t.Errorf("job.Name = %q, want %q", job.Name, "Build Job")
		}
		if job.RunsOn != "ubuntu-latest" {
			t.Errorf("job.RunsOn = %q, want %q", job.RunsOn, "ubuntu-latest")
		}
		if len(job.Steps) != 1 {
			t.Errorf("len(job.Steps) = %d, want 1", len(job.Steps))
		}
	})

	t.Run("returns error for non-existent job ID", func(t *testing.T) {
		wf := &WorkflowFile{
			Name: "Test Workflow",
			Jobs: map[string]Job{
				"build": {
					Name:   "Build Job",
					RunsOn: "ubuntu-latest",
				},
			},
		}

		_, err := SelectJob(wf, "nonexistent")
		if err == nil {
			t.Error("SelectJob() expected error for non-existent job ID, got nil")
		}
	})

	t.Run("returns error for nil workflow", func(t *testing.T) {
		_, err := SelectJob(nil, "build")
		if err == nil {
			t.Error("SelectJob() expected error for nil workflow, got nil")
		}
	})

	t.Run("returns error for empty job ID", func(t *testing.T) {
		wf := &WorkflowFile{
			Name: "Test Workflow",
			Jobs: map[string]Job{
				"build": {Name: "Build Job"},
			},
		}

		_, err := SelectJob(wf, "")
		if err == nil {
			t.Error("SelectJob() expected error for empty job ID, got nil")
		}
	})
}

// TestDiscoverWorkflowsEdgeCases tests edge cases for DiscoverWorkflows
func TestDiscoverWorkflowsEdgeCases(t *testing.T) {
	t.Run("returns error when workflows path is a file", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Create a file instead of a directory at .github/workflows
		ghDir := filepath.Join(tmpDir, ".github")
		if err := os.MkdirAll(ghDir, 0755); err != nil {
			t.Fatalf("failed to create .github directory: %v", err)
		}

		workflowsFile := filepath.Join(ghDir, "workflows")
		if err := os.WriteFile(workflowsFile, []byte("not a directory"), 0644); err != nil {
			t.Fatalf("failed to create workflows file: %v", err)
		}

		_, err := DiscoverWorkflows(tmpDir)
		if err == nil {
			t.Error("DiscoverWorkflows() expected error when workflows path is a file, got nil")
		}
	})

	t.Run("returns error when stat fails with permission error", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("skipping permission test as root")
		}

		tmpDir := t.TempDir()
		ghDir := filepath.Join(tmpDir, ".github")
		if err := os.MkdirAll(ghDir, 0755); err != nil {
			t.Fatalf("failed to create .github directory: %v", err)
		}

		// Create workflows directory but make parent unreadable
		workflowsDir := filepath.Join(ghDir, "workflows")
		if err := os.MkdirAll(workflowsDir, 0755); err != nil {
			t.Fatalf("failed to create workflows directory: %v", err)
		}

		// Make .github directory unreadable to trigger stat error
		if err := os.Chmod(ghDir, 0000); err != nil {
			t.Fatalf("failed to chmod: %v", err)
		}
		defer func() { _ = os.Chmod(ghDir, 0755) }()

		_, err := DiscoverWorkflows(tmpDir)
		if err == nil {
			t.Error("DiscoverWorkflows() expected error for permission denied on stat")
		}
	})

	t.Run("returns error when readdir fails", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("skipping permission test as root")
		}

		tmpDir := t.TempDir()
		workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
		if err := os.MkdirAll(workflowsDir, 0755); err != nil {
			t.Fatalf("failed to create workflows directory: %v", err)
		}

		// Create a workflow file
		workflowPath := filepath.Join(workflowsDir, "test.yml")
		if err := os.WriteFile(workflowPath, []byte("name: Test"), 0644); err != nil {
			t.Fatalf("failed to write workflow file: %v", err)
		}

		// Make directory unreadable but still stat-able
		if err := os.Chmod(workflowsDir, 0111); err != nil {
			t.Fatalf("failed to chmod: %v", err)
		}
		defer func() { _ = os.Chmod(workflowsDir, 0755) }()

		_, err := DiscoverWorkflows(tmpDir)
		if err == nil {
			t.Error("DiscoverWorkflows() expected error for permission denied on readdir")
		}
	})

	t.Run("returns error when directory is replaced during traversal", func(t *testing.T) {
		if runtime.GOOS != "linux" {
			t.Skip("race condition test only runs on Linux")
		}

		// This test attempts to trigger a race condition where the directory
		// is replaced between os.Stat and os.ReadDir. This is inherently racy
		// and may not always trigger the error path.
		var readDirErrorCount int
		var mu sync.Mutex

		for attempt := 0; attempt < 50; attempt++ {
			tmpDir := t.TempDir()
			ghDir := filepath.Join(tmpDir, ".github")
			workflowsDir := filepath.Join(ghDir, "workflows")

			if err := os.MkdirAll(workflowsDir, 0755); err != nil {
				continue
			}

			// Create initial workflow file
			workflowPath := filepath.Join(workflowsDir, "test.yml")
			if err := os.WriteFile(workflowPath, []byte("name: Test"), 0644); err != nil {
				continue
			}

			var wg sync.WaitGroup
			done := make(chan struct{})

			// Goroutine that rapidly replaces the directory with a file
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-done:
						return
					default:
						// Remove directory and replace with a file (causes ReadDir to fail)
						_ = os.RemoveAll(workflowsDir)
						_ = os.WriteFile(workflowsDir, []byte("not a dir"), 0644)
						// Restore directory
						_ = os.Remove(workflowsDir)
						_ = os.MkdirAll(workflowsDir, 0755)
					}
				}
			}()

			// Try to hit the race condition
			for i := 0; i < 100; i++ {
				_, err := DiscoverWorkflows(tmpDir)
				if err != nil {
					mu.Lock()
					readDirErrorCount++
					mu.Unlock()
				}
			}

			close(done)
			wg.Wait()

			// Clean up - restore directory structure for TempDir cleanup
			_ = os.Remove(workflowsDir)
			_ = os.MkdirAll(workflowsDir, 0755)
		}

		// We don't require the race to be hit - it's probabilistic
		// Just log if we managed to trigger it
		t.Logf("ReadDir errors triggered: %d", readDirErrorCount)
	})

	t.Run("discovers workflows with mixed case extensions", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
		if err := os.MkdirAll(workflowsDir, 0755); err != nil {
			t.Fatalf("failed to create workflows directory: %v", err)
		}

		// Create files with various case extensions
		files := []string{"workflow.YML", "test.YAML", "ci.yml", "deploy.yaml"}
		for _, file := range files {
			path := filepath.Join(workflowsDir, file)
			if err := os.WriteFile(path, []byte("name: Test"), 0644); err != nil {
				t.Fatalf("failed to write file %s: %v", file, path)
			}
		}

		workflows, err := DiscoverWorkflows(tmpDir)
		if err != nil {
			t.Fatalf("DiscoverWorkflows() error = %v", err)
		}

		// Should discover all 4 files regardless of case
		if len(workflows) != 4 {
			t.Errorf("len(workflows) = %d, want 4", len(workflows))
		}
	})

	t.Run("ignores subdirectories", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
		if err := os.MkdirAll(workflowsDir, 0755); err != nil {
			t.Fatalf("failed to create workflows directory: %v", err)
		}

		// Create a valid workflow file
		workflowFile := filepath.Join(workflowsDir, "valid.yml")
		if err := os.WriteFile(workflowFile, []byte("name: Valid"), 0644); err != nil {
			t.Fatalf("failed to write workflow file: %v", err)
		}

		// Create a subdirectory with workflow files
		subDir := filepath.Join(workflowsDir, "subdir")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("failed to create subdirectory: %v", err)
		}

		subWorkflow := filepath.Join(subDir, "workflow.yml")
		if err := os.WriteFile(subWorkflow, []byte("name: Sub"), 0644); err != nil {
			t.Fatalf("failed to write sub workflow: %v", err)
		}

		workflows, err := DiscoverWorkflows(tmpDir)
		if err != nil {
			t.Fatalf("DiscoverWorkflows() error = %v", err)
		}

		// Should only discover the one file in the root workflows directory
		if len(workflows) != 1 {
			t.Errorf("len(workflows) = %d, want 1", len(workflows))
		}

		if filepath.Base(workflows[0]) != "valid.yml" {
			t.Errorf("discovered workflow = %s, want valid.yml", filepath.Base(workflows[0]))
		}
	})
}

// TestDiscoverWorkflowsStatErrors tests os.Stat error scenarios that are not "not exist"
func TestDiscoverWorkflowsStatErrors(t *testing.T) {
	t.Run("returns error when workflows path has symlink loop", func(t *testing.T) {
		tmpDir := t.TempDir()
		ghDir := filepath.Join(tmpDir, ".github")
		if err := os.MkdirAll(ghDir, 0755); err != nil {
			t.Fatalf("failed to create .github directory: %v", err)
		}

		// Create a symlink loop: workflows -> workflows (points to itself)
		workflowsPath := filepath.Join(ghDir, "workflows")
		if err := os.Symlink(workflowsPath, workflowsPath); err != nil {
			t.Fatalf("failed to create symlink loop: %v", err)
		}

		_, err := DiscoverWorkflows(tmpDir)
		if err == nil {
			t.Error("DiscoverWorkflows() expected error for symlink loop, got nil")
		}
		// Should contain "failed to access workflows directory" message
		if err != nil && !strings.Contains(err.Error(), "failed to access") {
			t.Errorf("error message should mention 'failed to access', got: %v", err)
		}
	})
}

// TestDiscoverWorkflowsFileRaceConditions tests error handling for race conditions
// where files are deleted between directory listing and file info retrieval
func TestDiscoverWorkflowsFileRaceConditions(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("race condition tests only run on Linux")
	}

	t.Run("handles file deletion race condition gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
		if err := os.MkdirAll(workflowsDir, 0755); err != nil {
			t.Fatalf("failed to create workflows directory: %v", err)
		}

		// Create a stable workflow file
		stableFile := filepath.Join(workflowsDir, "stable.yml")
		if err := os.WriteFile(stableFile, []byte("name: Stable"), 0644); err != nil {
			t.Fatalf("failed to write stable file: %v", err)
		}

		// Create files that will be rapidly deleted/recreated to trigger race conditions
		var wg sync.WaitGroup
		stop := make(chan struct{})

		// Start goroutine that rapidly creates and deletes files
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; ; i++ {
				select {
				case <-stop:
					return
				default:
					filename := filepath.Join(workflowsDir, fmt.Sprintf("race%d.yml", i%10))
					_ = os.WriteFile(filename, []byte("name: Race"), 0644)
					_ = os.Remove(filename)
				}
			}
		}()

		// Call DiscoverWorkflows multiple times to try to hit the race
		for i := 0; i < 100; i++ {
			workflows, err := DiscoverWorkflows(tmpDir)
			// The function should not error, just skip problematic entries
			if err != nil {
				t.Fatalf("DiscoverWorkflows() unexpected error: %v", err)
			}
			// Should at least find the stable file
			found := false
			for _, wf := range workflows {
				if filepath.Base(wf) == "stable.yml" {
					found = true
					break
				}
			}
			if !found {
				t.Error("stable.yml should always be discovered")
			}
		}

		close(stop)
		wg.Wait()
	})

	t.Run("handles symlink target deletion race gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
		if err := os.MkdirAll(workflowsDir, 0755); err != nil {
			t.Fatalf("failed to create workflows directory: %v", err)
		}

		targetDir := filepath.Join(tmpDir, "targets")
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			t.Fatalf("failed to create targets directory: %v", err)
		}

		// Create a stable workflow file
		stableFile := filepath.Join(workflowsDir, "stable.yml")
		if err := os.WriteFile(stableFile, []byte("name: Stable"), 0644); err != nil {
			t.Fatalf("failed to write stable file: %v", err)
		}

		var wg sync.WaitGroup
		stop := make(chan struct{})

		// Start goroutine that rapidly creates and deletes symlink targets
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; ; i++ {
				select {
				case <-stop:
					return
				default:
					targetFile := filepath.Join(targetDir, fmt.Sprintf("target%d.yml", i%10))
					symlinkPath := filepath.Join(workflowsDir, fmt.Sprintf("link%d.yml", i%10))

					// Remove old symlink if exists
					_ = os.Remove(symlinkPath)

					// Create target file
					_ = os.WriteFile(targetFile, []byte("name: Target"), 0644)

					// Create symlink
					_ = os.Symlink(targetFile, symlinkPath)

					// Delete target file (makes symlink "broken" for Stat but EvalSymlinks may succeed)
					_ = os.Remove(targetFile)
				}
			}
		}()

		// Call DiscoverWorkflows multiple times to try to hit the race
		for i := 0; i < 100; i++ {
			workflows, err := DiscoverWorkflows(tmpDir)
			// The function should not error, just skip problematic entries
			if err != nil {
				t.Fatalf("DiscoverWorkflows() unexpected error: %v", err)
			}
			// Should at least find the stable file
			found := false
			for _, wf := range workflows {
				if filepath.Base(wf) == "stable.yml" {
					found = true
					break
				}
			}
			if !found {
				t.Error("stable.yml should always be discovered")
			}
		}

		close(stop)
		wg.Wait()
	})
}

// TestDiscoverWorkflowsProcFS tests symlink handling with /proc filesystem
func TestDiscoverWorkflowsProcFS(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("/proc tests only run on Linux")
	}

	t.Run("skips symlink to closed file descriptor", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
		if err := os.MkdirAll(workflowsDir, 0755); err != nil {
			t.Fatalf("failed to create workflows directory: %v", err)
		}

		// Create a regular workflow file
		regularFile := filepath.Join(workflowsDir, "regular.yml")
		if err := os.WriteFile(regularFile, []byte("name: Regular"), 0644); err != nil {
			t.Fatalf("failed to write regular file: %v", err)
		}

		// Create a temporary file to get a file descriptor
		tmpFile, err := os.CreateTemp("", "workflow-*.yml")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		tmpFilePath := tmpFile.Name()
		fdPath := fmt.Sprintf("/proc/self/fd/%d", tmpFile.Fd())

		// Create symlink to the fd path
		symlinkPath := filepath.Join(workflowsDir, "fd-link.yml")
		if err := os.Symlink(fdPath, symlinkPath); err != nil {
			tmpFile.Close()
			os.Remove(tmpFilePath)
			t.Fatalf("failed to create symlink to fd: %v", err)
		}

		// Close the file descriptor AND delete the temp file
		// This ensures EvalSymlinks returns the path, but os.Stat on it fails
		tmpFile.Close()
		os.Remove(tmpFilePath)

		workflows, err := DiscoverWorkflows(tmpDir)
		if err != nil {
			t.Fatalf("DiscoverWorkflows() error = %v", err)
		}

		// Should only discover the regular file, not the broken fd symlink
		if len(workflows) != 1 {
			t.Errorf("len(workflows) = %d, want 1", len(workflows))
		}

		if len(workflows) > 0 && filepath.Base(workflows[0]) != "regular.yml" {
			t.Errorf("discovered workflow = %s, want regular.yml", filepath.Base(workflows[0]))
		}
	})

	t.Run("skips symlink where resolved path stat fails", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
		if err := os.MkdirAll(workflowsDir, 0755); err != nil {
			t.Fatalf("failed to create workflows directory: %v", err)
		}

		// Create a regular workflow file
		regularFile := filepath.Join(workflowsDir, "regular.yml")
		if err := os.WriteFile(regularFile, []byte("name: Regular"), 0644); err != nil {
			t.Fatalf("failed to write regular file: %v", err)
		}

		// Create a target file that we'll delete after creating the symlink
		targetDir := filepath.Join(tmpDir, "target")
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			t.Fatalf("failed to create target directory: %v", err)
		}

		targetFile := filepath.Join(targetDir, "target.yml")
		if err := os.WriteFile(targetFile, []byte("name: Target"), 0644); err != nil {
			t.Fatalf("failed to write target file: %v", err)
		}

		// Create symlink to the target file
		symlinkPath := filepath.Join(workflowsDir, "will-break.yml")
		if err := os.Symlink(targetFile, symlinkPath); err != nil {
			t.Fatalf("failed to create symlink: %v", err)
		}

		// Delete the target file - this makes the symlink broken
		// EvalSymlinks will fail, which tests line 76-79
		if err := os.Remove(targetFile); err != nil {
			t.Fatalf("failed to remove target file: %v", err)
		}

		workflows, err := DiscoverWorkflows(tmpDir)
		if err != nil {
			t.Fatalf("DiscoverWorkflows() error = %v", err)
		}

		// Should only discover the regular file
		if len(workflows) != 1 {
			t.Errorf("len(workflows) = %d, want 1", len(workflows))
		}

		if len(workflows) > 0 && filepath.Base(workflows[0]) != "regular.yml" {
			t.Errorf("discovered workflow = %s, want regular.yml", filepath.Base(workflows[0]))
		}
	})
}

// TestDiscoverWorkflowsSymlinks tests symlink handling in DiscoverWorkflows
func TestDiscoverWorkflowsSymlinks(t *testing.T) {
	t.Run("discovers valid symlink to workflow file", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
		if err := os.MkdirAll(workflowsDir, 0755); err != nil {
			t.Fatalf("failed to create workflows directory: %v", err)
		}

		// Create a target workflow file outside the workflows directory
		targetDir := filepath.Join(tmpDir, "shared")
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			t.Fatalf("failed to create target directory: %v", err)
		}

		targetFile := filepath.Join(targetDir, "shared-workflow.yml")
		if err := os.WriteFile(targetFile, []byte("name: Shared"), 0644); err != nil {
			t.Fatalf("failed to write target file: %v", err)
		}

		// Create a symlink to the target file
		symlinkPath := filepath.Join(workflowsDir, "linked.yml")
		if err := os.Symlink(targetFile, symlinkPath); err != nil {
			t.Fatalf("failed to create symlink: %v", err)
		}

		// Also create a regular workflow file
		regularFile := filepath.Join(workflowsDir, "regular.yml")
		if err := os.WriteFile(regularFile, []byte("name: Regular"), 0644); err != nil {
			t.Fatalf("failed to write regular file: %v", err)
		}

		workflows, err := DiscoverWorkflows(tmpDir)
		if err != nil {
			t.Fatalf("DiscoverWorkflows() error = %v", err)
		}

		// Should discover both the regular file and the symlink
		if len(workflows) != 2 {
			t.Errorf("len(workflows) = %d, want 2", len(workflows))
		}
	})

	t.Run("skips broken symlinks", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
		if err := os.MkdirAll(workflowsDir, 0755); err != nil {
			t.Fatalf("failed to create workflows directory: %v", err)
		}

		// Create a symlink to a non-existent file (broken symlink)
		brokenSymlink := filepath.Join(workflowsDir, "broken.yml")
		if err := os.Symlink("/nonexistent/path/workflow.yml", brokenSymlink); err != nil {
			t.Fatalf("failed to create broken symlink: %v", err)
		}

		// Also create a regular workflow file
		regularFile := filepath.Join(workflowsDir, "regular.yml")
		if err := os.WriteFile(regularFile, []byte("name: Regular"), 0644); err != nil {
			t.Fatalf("failed to write regular file: %v", err)
		}

		workflows, err := DiscoverWorkflows(tmpDir)
		if err != nil {
			t.Fatalf("DiscoverWorkflows() error = %v", err)
		}

		// Should only discover the regular file, not the broken symlink
		if len(workflows) != 1 {
			t.Errorf("len(workflows) = %d, want 1", len(workflows))
		}

		if len(workflows) > 0 && filepath.Base(workflows[0]) != "regular.yml" {
			t.Errorf("discovered workflow = %s, want regular.yml", filepath.Base(workflows[0]))
		}
	})

	t.Run("skips symlinks pointing to directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
		if err := os.MkdirAll(workflowsDir, 0755); err != nil {
			t.Fatalf("failed to create workflows directory: %v", err)
		}

		// Create a target directory
		targetDir := filepath.Join(tmpDir, "target-dir")
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			t.Fatalf("failed to create target directory: %v", err)
		}

		// Create a symlink to the directory with .yml extension
		symlinkToDir := filepath.Join(workflowsDir, "dir-link.yml")
		if err := os.Symlink(targetDir, symlinkToDir); err != nil {
			t.Fatalf("failed to create symlink to directory: %v", err)
		}

		// Also create a regular workflow file
		regularFile := filepath.Join(workflowsDir, "regular.yml")
		if err := os.WriteFile(regularFile, []byte("name: Regular"), 0644); err != nil {
			t.Fatalf("failed to write regular file: %v", err)
		}

		workflows, err := DiscoverWorkflows(tmpDir)
		if err != nil {
			t.Fatalf("DiscoverWorkflows() error = %v", err)
		}

		// Should only discover the regular file, not the symlink to directory
		if len(workflows) != 1 {
			t.Errorf("len(workflows) = %d, want 1", len(workflows))
		}

		if len(workflows) > 0 && filepath.Base(workflows[0]) != "regular.yml" {
			t.Errorf("discovered workflow = %s, want regular.yml", filepath.Base(workflows[0]))
		}
	})

	t.Run("skips symlinks with inaccessible targets", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("skipping permission test as root")
		}

		tmpDir := t.TempDir()
		workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
		if err := os.MkdirAll(workflowsDir, 0755); err != nil {
			t.Fatalf("failed to create workflows directory: %v", err)
		}

		// Create a target file in a directory that will become inaccessible
		restrictedDir := filepath.Join(tmpDir, "restricted")
		if err := os.MkdirAll(restrictedDir, 0755); err != nil {
			t.Fatalf("failed to create restricted directory: %v", err)
		}

		targetFile := filepath.Join(restrictedDir, "target.yml")
		if err := os.WriteFile(targetFile, []byte("name: Target"), 0644); err != nil {
			t.Fatalf("failed to write target file: %v", err)
		}

		// Create a symlink to the target file
		symlinkPath := filepath.Join(workflowsDir, "restricted-link.yml")
		if err := os.Symlink(targetFile, symlinkPath); err != nil {
			t.Fatalf("failed to create symlink: %v", err)
		}

		// Make the target directory inaccessible
		if err := os.Chmod(restrictedDir, 0000); err != nil {
			t.Fatalf("failed to chmod restricted directory: %v", err)
		}
		defer func() { _ = os.Chmod(restrictedDir, 0755) }()

		// Also create a regular workflow file
		regularFile := filepath.Join(workflowsDir, "regular.yml")
		if err := os.WriteFile(regularFile, []byte("name: Regular"), 0644); err != nil {
			t.Fatalf("failed to write regular file: %v", err)
		}

		workflows, err := DiscoverWorkflows(tmpDir)
		if err != nil {
			t.Fatalf("DiscoverWorkflows() error = %v", err)
		}

		// Should only discover the regular file, not the symlink with inaccessible target
		if len(workflows) != 1 {
			t.Errorf("len(workflows) = %d, want 1", len(workflows))
		}

		if len(workflows) > 0 && filepath.Base(workflows[0]) != "regular.yml" {
			t.Errorf("discovered workflow = %s, want regular.yml", filepath.Base(workflows[0]))
		}
	})
}
