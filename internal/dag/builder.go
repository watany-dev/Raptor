package dag

import (
	"fmt"

	"github.com/watany-dev/raptor/internal/workflow"
)

// BuildFromJobs constructs a dependency graph from workflow jobs.
// It validates that all dependencies exist and there are no cycles.
func BuildFromJobs(jobs map[string]workflow.Job) (*Graph, error) {
	g := New()

	// First pass: add all jobs as nodes
	for jobID := range jobs {
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

	// Get all transitive dependencies
	allDeps := g.GetAllDependencies(jobID)

	// Create a subgraph with only the relevant nodes
	subgraph := New()
	subgraph.AddNode(jobID)
	for _, dep := range allDeps {
		subgraph.AddNode(dep)
	}

	// Add edges for the subgraph
	for node := range subgraph.nodes {
		for _, dep := range g.edges[node] {
			if subgraph.nodes[dep] {
				subgraph.edges[node] = append(subgraph.edges[node], dep)
			}
		}
	}

	return subgraph.TopologicalSort()
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
