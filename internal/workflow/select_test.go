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
