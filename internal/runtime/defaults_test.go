package runtime

import (
	"reflect"
	"testing"
)

func TestMergeEnv(t *testing.T) {
	tests := []struct {
		name     string
		maps     []map[string]string
		expected map[string]string
	}{
		{
			name:     "empty input",
			maps:     nil,
			expected: map[string]string{},
		},
		{
			name: "single map",
			maps: []map[string]string{
				{"KEY1": "value1", "KEY2": "value2"},
			},
			expected: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
			},
		},
		{
			name: "multiple maps without overlap",
			maps: []map[string]string{
				{"KEY1": "value1"},
				{"KEY2": "value2"},
				{"KEY3": "value3"},
			},
			expected: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
				"KEY3": "value3",
			},
		},
		{
			name: "later maps override earlier maps",
			maps: []map[string]string{
				{"KEY1": "workflow_value"},
				{"KEY1": "job_value"},
				{"KEY1": "step_value"},
			},
			expected: map[string]string{
				"KEY1": "step_value",
			},
		},
		{
			name: "workflow -> job -> step env merge",
			maps: []map[string]string{
				// Workflow env
				{"SHARED": "workflow", "WORKFLOW_ONLY": "w1"},
				// Job env
				{"SHARED": "job", "JOB_ONLY": "j1"},
				// Step env
				{"SHARED": "step", "STEP_ONLY": "s1"},
			},
			expected: map[string]string{
				"SHARED":        "step",
				"WORKFLOW_ONLY": "w1",
				"JOB_ONLY":      "j1",
				"STEP_ONLY":     "s1",
			},
		},
		{
			name: "nil maps in input are skipped",
			maps: []map[string]string{
				{"KEY1": "value1"},
				nil,
				{"KEY2": "value2"},
			},
			expected: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
			},
		},
		{
			name: "empty maps are handled",
			maps: []map[string]string{
				{"KEY1": "value1"},
				{},
				{"KEY2": "value2"},
			},
			expected: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeEnv(tt.maps...)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("MergeEnv() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestMergeEnv_OrderPreservation(t *testing.T) {
	// Test that later values definitely override earlier ones
	workflowEnv := map[string]string{
		"DEBUG": "false",
		"MODE":  "production",
	}
	jobEnv := map[string]string{
		"DEBUG": "true",
	}
	stepEnv := map[string]string{
		"MODE": "development",
	}

	result := MergeEnv(workflowEnv, jobEnv, stepEnv)

	if result["DEBUG"] != "true" {
		t.Errorf("DEBUG should be 'true' from job env, got %q", result["DEBUG"])
	}
	if result["MODE"] != "development" {
		t.Errorf("MODE should be 'development' from step env, got %q", result["MODE"])
	}
}

func TestDefaultBaseEnv(t *testing.T) {
	workspacePath := "/workspace/project"
	sha := "abc123def456"
	ref := "refs/heads/main"

	result := DefaultBaseEnv(workspacePath, sha, ref)

	// Verify all expected keys are present
	expectedKeys := []string{"CI", "GITHUB_ACTIONS", "GITHUB_WORKSPACE", "GITHUB_SHA", "GITHUB_REF"}
	for _, key := range expectedKeys {
		if _, ok := result[key]; !ok {
			t.Errorf("DefaultBaseEnv() missing key %q", key)
		}
	}

	// Verify values
	if result["CI"] != "true" {
		t.Errorf("CI = %q, want %q", result["CI"], "true")
	}
	if result["GITHUB_ACTIONS"] != "true" {
		t.Errorf("GITHUB_ACTIONS = %q, want %q", result["GITHUB_ACTIONS"], "true")
	}
	if result["GITHUB_WORKSPACE"] != workspacePath {
		t.Errorf("GITHUB_WORKSPACE = %q, want %q", result["GITHUB_WORKSPACE"], workspacePath)
	}
	if result["GITHUB_SHA"] != sha {
		t.Errorf("GITHUB_SHA = %q, want %q", result["GITHUB_SHA"], sha)
	}
	if result["GITHUB_REF"] != ref {
		t.Errorf("GITHUB_REF = %q, want %q", result["GITHUB_REF"], ref)
	}
}

func TestDefaultBaseEnv_EmptyValues(t *testing.T) {
	// Test with empty values (e.g., detached HEAD)
	result := DefaultBaseEnv("/workspace", "", "")

	if result["CI"] != "true" {
		t.Errorf("CI should always be 'true'")
	}
	if result["GITHUB_SHA"] != "" {
		t.Errorf("GITHUB_SHA should be empty, got %q", result["GITHUB_SHA"])
	}
	if result["GITHUB_REF"] != "" {
		t.Errorf("GITHUB_REF should be empty, got %q", result["GITHUB_REF"])
	}
}
