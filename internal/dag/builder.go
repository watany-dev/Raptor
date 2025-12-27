package dag

import (
	"errors"
	"fmt"

	"github.com/watany-dev/raptor/internal/workflow"
)

// Limits matching GitHub Actions constraints.
const (
	// MaxJobs is the maximum number of jobs per workflow (matches GitHub Actions limit).
	MaxJobs = 256
	// MaxJobNameLength is the maximum length of a job identifier.
	MaxJobNameLength = 256
)

// ErrTooManyJobs is returned when a workflow exceeds the maximum job limit.
var ErrTooManyJobs = errors.New("workflow exceeds maximum job limit")

// ErrJobNameTooLong is returned when a job name exceeds the maximum length.
var ErrJobNameTooLong = errors.New("job name exceeds maximum length")

// BuildFromJobs constructs a dependency graph from workflow jobs.
// It validates that all dependencies exist and there are no cycles.
func BuildFromJobs(jobs map[string]workflow.Job) (*Graph, error) {
	// Validate job count (matches GitHub Actions limit of 256)
	if len(jobs) > MaxJobs {
		return nil, fmt.Errorf("%w: %d jobs (max %d)", ErrTooManyJobs, len(jobs), MaxJobs)
	}

	g := New()

	// First pass: add all jobs as nodes with name validation
	for jobID := range jobs {
		if len(jobID) > MaxJobNameLength {
			return nil, fmt.Errorf("%w: '%s' (%d chars, max %d)",
				ErrJobNameTooLong, jobID[:50]+"...", len(jobID), MaxJobNameLength)
		}
		g.AddNode(jobID)
	}

	// Second pass: add dependency edges
	for jobID, job := range jobs {
		for _, dep := range job.Needs {
			// Check for self-dependency
			if dep == jobID {
				return nil, fmt.Errorf("%w: '%s'", ErrSelfDependency, jobID)
			}

			// Check if dependency exists
			if !g.nodes[dep] {
				return nil, fmt.Errorf("%w: job '%s' depends on non-existent job '%s'",
					ErrUnknownDependency, jobID, dep)
			}

			// Add edge: jobID depends on dep
			// We bypass AddEdge validation since we already checked
			g.edges[jobID] = append(g.edges[jobID], dep)
		}
	}

	// Validate no cycles exist
	if cycle := g.FindCycle(); cycle != nil {
		return nil, fmt.Errorf("%w: %v", ErrCyclicDependency, formatCycle(cycle))
	}

	return g, nil
}

// ResolveWithDependencies returns the given job and all its dependencies
// in topological order (dependencies first).
func ResolveWithDependencies(g *Graph, jobID string) ([]string, error) {
	if !g.nodes[jobID] {
		return nil, fmt.Errorf("%w: '%s'", ErrUnknownDependency, jobID)
	}

	// Build set of target nodes (job + all dependencies)
	allDeps := g.GetAllDependencies(jobID)
	targetNodes := make(map[string]bool, len(allDeps)+1)
	targetNodes[jobID] = true
	for _, dep := range allDeps {
		targetNodes[dep] = true
	}

	// Get full topological order and filter to target nodes
	fullOrder, err := g.TopologicalSort()
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, len(targetNodes))
	for _, node := range fullOrder {
		if targetNodes[node] {
			result = append(result, node)
		}
	}
	return result, nil
}

// formatCycle formats a cycle path for error messages.
func formatCycle(cycle []string) string {
	if len(cycle) == 0 {
		return ""
	}

	result := cycle[0]
	for i := 1; i < len(cycle); i++ {
		result += " -> " + cycle[i]
	}
	return result
}
