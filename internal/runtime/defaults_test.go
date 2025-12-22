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
