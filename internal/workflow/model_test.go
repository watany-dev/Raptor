package workflow

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestStringOrSlice_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    []string
		wantErr bool
	}{
		{
			name: "single string value",
			yaml: `needs: build`,
			want: []string{"build"},
		},
		{
			name: "array with single element",
			yaml: `needs: [build]`,
			want: []string{"build"},
		},
		{
			name: "array with multiple elements",
			yaml: `needs: [build, test]`,
			want: []string{"build", "test"},
		},
		{
			name: "array with multiple elements multiline",
			yaml: `needs:
  - build
  - test
  - lint`,
			want: []string{"build", "test", "lint"},
		},
		{
			name: "empty array",
			yaml: `needs: []`,
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result struct {
				Needs StringOrSlice `yaml:"needs"`
			}

			err := yaml.Unmarshal([]byte(tt.yaml), &result)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(result.Needs) != len(tt.want) {
				t.Errorf("UnmarshalYAML() got %v, want %v", result.Needs, tt.want)
				return
			}

			for i, v := range result.Needs {
				if v != tt.want[i] {
					t.Errorf("UnmarshalYAML() got[%d] = %v, want %v", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestJob_Needs_Integration(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string
		wantNeeds []string
	}{
		{
			name: "job with single dependency",
			yaml: `
name: test
jobs:
  deploy:
    runs-on: ubuntu-latest
    needs: build
    steps:
      - run: echo deploy`,
			wantNeeds: []string{"build"},
		},
		{
			name: "job with multiple dependencies",
			yaml: `
name: test
jobs:
  deploy:
    runs-on: ubuntu-latest
    needs: [build, test]
    steps:
      - run: echo deploy`,
			wantNeeds: []string{"build", "test"},
		},
		{
			name: "job without dependencies",
			yaml: `
name: test
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - run: echo build`,
			wantNeeds: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wf WorkflowFile
			err := yaml.Unmarshal([]byte(tt.yaml), &wf)
			if err != nil {
				t.Fatalf("failed to unmarshal workflow: %v", err)
			}

			// Get the first job
			var job Job
			for _, j := range wf.Jobs {
				job = j
				break
			}

			if len(job.Needs) != len(tt.wantNeeds) {
				t.Errorf("Job.Needs = %v, want %v", job.Needs, tt.wantNeeds)
				return
			}

			for i, v := range job.Needs {
				if v != tt.wantNeeds[i] {
					t.Errorf("Job.Needs[%d] = %v, want %v", i, v, tt.wantNeeds[i])
				}
			}
		})
	}
}

func TestStep_IsAction(t *testing.T) {
	t.Run("returns true when Uses is set", func(t *testing.T) {
		step := &Step{
			Name: "Checkout",
			Uses: "actions/checkout@v4",
		}

		if !step.IsAction() {
			t.Error("IsAction() should return true when Uses is set")
		}
	})

	t.Run("returns false when Uses is empty", func(t *testing.T) {
		step := &Step{
			Name: "Run tests",
			Run:  "go test ./...",
		}

		if step.IsAction() {
			t.Error("IsAction() should return false when Uses is empty")
		}
	})

	t.Run("returns true for composite action", func(t *testing.T) {
		step := &Step{
			Name: "Setup Go",
			Uses: "actions/setup-go@v5",
			With: map[string]string{
				"go-version": "1.21",
			},
		}

		if !step.IsAction() {
			t.Error("IsAction() should return true for action with With parameters")
		}
	})

	t.Run("returns false for step with only Run", func(t *testing.T) {
		step := &Step{
			Run: "echo hello",
		}

		if step.IsAction() {
			t.Error("IsAction() should return false for step with only Run")
		}
	})

	t.Run("returns true for local action", func(t *testing.T) {
		step := &Step{
			Uses: "./.github/actions/my-action",
		}

		if !step.IsAction() {
			t.Error("IsAction() should return true for local action")
		}
	})
}
