package workflow

import (
	"os"
	"path/filepath"
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
