package dag

import (
	"errors"
	"fmt"
	"testing"

	"github.com/watany-dev/raptor/internal/workflow"
)

func TestBuildFromJobs(t *testing.T) {
	tests := []struct {
		name       string
		jobs       map[string]workflow.Job
		wantErr    error
		wantNodes  int
		validateFn func(*Graph) error
	}{
		{
			name: "simple linear dependency",
			jobs: map[string]workflow.Job{
				"build":  {},
				"test":   {Needs: []string{"build"}},
				"deploy": {Needs: []string{"test"}},
			},
			wantNodes: 3,
			validateFn: func(g *Graph) error {
				order, err := g.TopologicalSort()
				if err != nil {
					return err
				}
				// build should come before test, test before deploy
				pos := make(map[string]int)
				for i, n := range order {
					pos[n] = i
				}
				if pos["build"] >= pos["test"] {
					return errors.New("build should come before test")
				}
				if pos["test"] >= pos["deploy"] {
					return errors.New("test should come before deploy")
				}
				return nil
			},
		},
		{
			name: "diamond dependency",
			jobs: map[string]workflow.Job{
				"build":  {},
				"lint":   {Needs: []string{"build"}},
				"test":   {Needs: []string{"build"}},
				"deploy": {Needs: []string{"lint", "test"}},
			},
			wantNodes: 4,
			validateFn: func(g *Graph) error {
				order, err := g.TopologicalSort()
				if err != nil {
					return err
				}
				pos := make(map[string]int)
				for i, n := range order {
					pos[n] = i
				}
				// build should come before lint and test
				if pos["build"] >= pos["lint"] || pos["build"] >= pos["test"] {
					return errors.New("build should come before lint and test")
				}
				// lint and test should come before deploy
				if pos["lint"] >= pos["deploy"] || pos["test"] >= pos["deploy"] {
					return errors.New("lint and test should come before deploy")
				}
				return nil
			},
		},
		{
			name: "no dependencies",
			jobs: map[string]workflow.Job{
				"job1": {},
				"job2": {},
				"job3": {},
			},
			wantNodes: 3,
		},
		{
			name:      "empty jobs",
			jobs:      map[string]workflow.Job{},
			wantNodes: 0,
		},
		{
			name: "unknown dependency",
			jobs: map[string]workflow.Job{
				"build": {},
				"test":  {Needs: []string{"nonexistent"}},
			},
			wantErr: ErrUnknownDependency,
		},
		{
			name: "self dependency",
			jobs: map[string]workflow.Job{
				"build": {Needs: []string{"build"}},
			},
			wantErr: ErrSelfDependency,
		},
		{
			name: "cyclic dependency - simple",
			jobs: map[string]workflow.Job{
				"a": {Needs: []string{"b"}},
				"b": {Needs: []string{"a"}},
			},
			wantErr: ErrCyclicDependency,
		},
		{
			name: "cyclic dependency - three nodes",
			jobs: map[string]workflow.Job{
				"a": {Needs: []string{"c"}},
				"b": {Needs: []string{"a"}},
				"c": {Needs: []string{"b"}},
			},
			wantErr: ErrCyclicDependency,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := BuildFromJobs(tt.jobs)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(g.nodes) != tt.wantNodes {
				t.Errorf("expected %d nodes, got %d", tt.wantNodes, len(g.nodes))
			}

			if tt.validateFn != nil {
				if err := tt.validateFn(g); err != nil {
					t.Errorf("validation failed: %v", err)
				}
			}
		})
	}
}

func TestResolveWithDependencies(t *testing.T) {
	tests := []struct {
		name      string
		jobs      map[string]workflow.Job
		targetJob string
		wantJobs  []string // jobs that should be in result (order may vary)
		wantErr   error
	}{
		{
			name: "job with no dependencies",
			jobs: map[string]workflow.Job{
				"build": {},
				"test":  {Needs: []string{"build"}},
			},
			targetJob: "build",
			wantJobs:  []string{"build"},
		},
		{
			name: "job with one dependency",
			jobs: map[string]workflow.Job{
				"build": {},
				"test":  {Needs: []string{"build"}},
			},
			targetJob: "test",
			wantJobs:  []string{"build", "test"},
		},
		{
			name: "job with transitive dependencies",
			jobs: map[string]workflow.Job{
				"build":  {},
				"test":   {Needs: []string{"build"}},
				"deploy": {Needs: []string{"test"}},
			},
			targetJob: "deploy",
			wantJobs:  []string{"build", "test", "deploy"},
		},
		{
			name: "job with multiple dependencies",
			jobs: map[string]workflow.Job{
				"build":  {},
				"lint":   {Needs: []string{"build"}},
				"test":   {Needs: []string{"build"}},
				"deploy": {Needs: []string{"lint", "test"}},
			},
			targetJob: "deploy",
			wantJobs:  []string{"build", "lint", "test", "deploy"},
		},
		{
			name: "unknown job",
			jobs: map[string]workflow.Job{
				"build": {},
			},
			targetJob: "nonexistent",
			wantErr:   ErrUnknownDependency,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := BuildFromJobs(tt.jobs)
			if err != nil {
				t.Fatalf("failed to build graph: %v", err)
			}

			result, err := ResolveWithDependencies(g, tt.targetJob)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Check all expected jobs are present
			resultSet := make(map[string]bool)
			for _, j := range result {
				resultSet[j] = true
			}

			for _, wantJob := range tt.wantJobs {
				if !resultSet[wantJob] {
					t.Errorf("expected job %s in result, got %v", wantJob, result)
				}
			}

			if len(result) != len(tt.wantJobs) {
				t.Errorf("expected %d jobs, got %d: %v", len(tt.wantJobs), len(result), result)
			}

			// Verify topological order
			pos := make(map[string]int)
			for i, j := range result {
				pos[j] = i
			}

			for _, job := range result {
				for _, dep := range g.edges[job] {
					if resultSet[dep] && pos[dep] >= pos[job] {
						t.Errorf("dependency %s should come before %s", dep, job)
					}
				}
			}
		})
	}
}

func TestFormatCycle(t *testing.T) {
	tests := []struct {
		name  string
		cycle []string
		want  string
	}{
		{
			name:  "empty cycle",
			cycle: []string{},
			want:  "",
		},
		{
			name:  "single node",
			cycle: []string{"a"},
			want:  "a",
		},
		{
			name:  "two nodes",
			cycle: []string{"a", "b"},
			want:  "a -> b",
		},
		{
			name:  "three nodes",
			cycle: []string{"a", "b", "c"},
			want:  "a -> b -> c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCycle(tt.cycle)
			if got != tt.want {
				t.Errorf("formatCycle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildFromJobs_TooManyJobs(t *testing.T) {
	// Create a workflow with more than MaxJobs
	jobs := make(map[string]workflow.Job)
	for i := 0; i <= MaxJobs; i++ {
		jobs[fmt.Sprintf("job%d", i)] = workflow.Job{}
	}

	_, err := BuildFromJobs(jobs)
	if err == nil {
		t.Fatal("expected error for too many jobs, got nil")
	}
	if !errors.Is(err, ErrTooManyJobs) {
		t.Errorf("expected ErrTooManyJobs, got %v", err)
	}
}

func TestBuildFromJobs_JobNameTooLong(t *testing.T) {
	// Create a job with a name longer than MaxJobNameLength
	longName := make([]byte, MaxJobNameLength+1)
	for i := range longName {
		longName[i] = 'a'
	}

	jobs := map[string]workflow.Job{
		string(longName): {},
	}

	_, err := BuildFromJobs(jobs)
	if err == nil {
		t.Fatal("expected error for job name too long, got nil")
	}
	if !errors.Is(err, ErrJobNameTooLong) {
		t.Errorf("expected ErrJobNameTooLong, got %v", err)
	}
}

func TestBuildFromJobs_AtMaxLimit(t *testing.T) {
	// Create exactly MaxJobs jobs - should succeed
	jobs := make(map[string]workflow.Job)
	for i := 0; i < MaxJobs; i++ {
		jobs[fmt.Sprintf("job%d", i)] = workflow.Job{}
	}

	g, err := BuildFromJobs(jobs)
	if err != nil {
		t.Fatalf("expected success for exactly %d jobs, got %v", MaxJobs, err)
	}
	if len(g.nodes) != MaxJobs {
		t.Errorf("expected %d nodes, got %d", MaxJobs, len(g.nodes))
	}
}

func TestBuildFromJobs_JobNameAtMaxLength(t *testing.T) {
	// Create a job with exactly MaxJobNameLength - should succeed
	exactName := make([]byte, MaxJobNameLength)
	for i := range exactName {
		exactName[i] = 'a'
	}

	jobs := map[string]workflow.Job{
		string(exactName): {},
	}

	g, err := BuildFromJobs(jobs)
	if err != nil {
		t.Fatalf("expected success for job name at max length, got %v", err)
	}
	if len(g.nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(g.nodes))
	}
}
