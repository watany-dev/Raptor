package dag

import (
	"errors"
	"fmt"
	"sort"
)

// Common errors returned by the DAG package.
var (
	ErrCyclicDependency  = errors.New("cyclic dependency detected")
	ErrUnknownDependency = errors.New("unknown dependency")
	ErrSelfDependency    = errors.New("job cannot depend on itself")
)

// Graph represents a directed acyclic graph for job dependencies.
type Graph struct {
	nodes map[string]bool     // Set of all node IDs
	edges map[string][]string // node -> list of nodes it depends on
}

// New creates a new empty Graph.
func New() *Graph {
	return &Graph{
		nodes: make(map[string]bool),
		edges: make(map[string][]string),
	}
}

// AddNode adds a node to the graph.
func (g *Graph) AddNode(id string) {
	g.nodes[id] = true
	if _, exists := g.edges[id]; !exists {
		g.edges[id] = nil
	}
}

// AddEdge adds a dependency edge: "from" depends on "to".
// Returns an error if either node doesn't exist or if it would create a self-dependency.
func (g *Graph) AddEdge(from, to string) error {
	if from == to {
		return fmt.Errorf("%w: '%s'", ErrSelfDependency, from)
	}
	if !g.nodes[from] {
		return fmt.Errorf("%w: '%s' (referenced by '%s')", ErrUnknownDependency, from, to)
	}
	if !g.nodes[to] {
		return fmt.Errorf("%w: '%s' (referenced by '%s')", ErrUnknownDependency, to, from)
	}

	g.edges[from] = append(g.edges[from], to)
	return nil
}

// GetDependencies returns the direct dependencies of a node.
func (g *Graph) GetDependencies(id string) []string {
	return g.edges[id]
}

// GetAllDependencies returns all dependencies of a node (transitive closure).
func (g *Graph) GetAllDependencies(id string) []string {
	visited := make(map[string]bool)
	var result []string

	var visit func(nodeID string)
	visit = func(nodeID string) {
		for _, dep := range g.edges[nodeID] {
			if !visited[dep] {
				visited[dep] = true
				result = append(result, dep)
				visit(dep)
			}
		}
	}

	visit(id)
	return result
}

// HasCycle returns true if the graph contains a cycle.
func (g *Graph) HasCycle() bool {
	return len(g.FindCycle()) > 0
}

// FindCycle returns the nodes involved in a cycle, or nil if no cycle exists.
// Uses DFS with three-color marking (white/gray/black).
func (g *Graph) FindCycle() []string {
	const (
		white = 0 // unvisited
		gray  = 1 // visiting (in current path)
		black = 2 // visited (completed)
	)

	color := make(map[string]int)
	parent := make(map[string]string)

	var cyclePath []string

	var dfs func(node string) bool
	dfs = func(node string) bool {
		color[node] = gray

		for _, dep := range g.edges[node] {
			if color[dep] == gray {
				// Found cycle - reconstruct path
				cyclePath = []string{dep}
				for n := node; n != dep; n = parent[n] {
					cyclePath = append([]string{n}, cyclePath...)
				}
				cyclePath = append([]string{dep}, cyclePath...)
				return true
			}
			if color[dep] == white {
				parent[dep] = node
				if dfs(dep) {
					return true
				}
			}
		}

		color[node] = black
		return false
	}

	// Get sorted node list for deterministic output
	nodes := g.sortedNodes()
	for _, node := range nodes {
		if color[node] == white {
			if dfs(node) {
				return cyclePath
			}
		}
	}

	return nil
}

// TopologicalSort returns nodes in topological order (dependencies first).
// Uses Kahn's algorithm. Returns an error if the graph contains a cycle.
func (g *Graph) TopologicalSort() ([]string, error) {
	if len(g.nodes) == 0 {
		return nil, nil
	}

	// Calculate in-degree for each node
	inDegree := make(map[string]int)
	for node := range g.nodes {
		inDegree[node] = 0
	}
	for _, deps := range g.edges {
		for _, dep := range deps {
			inDegree[dep]++
		}
	}

	// Find all nodes with no incoming edges (in-degree 0)
	// These are the "leaf" nodes that depend on others but nothing depends on them
	var queue []string
	for node := range g.nodes {
		if inDegree[node] == 0 {
			queue = append(queue, node)
		}
	}
	// Sort for deterministic order
	sort.Strings(queue)

	var result []string

	for len(queue) > 0 {
		// Sort once per iteration for deterministic order
		sort.Strings(queue)

		// Pop from queue
		node := queue[0]
		queue = queue[1:]

		// This node depends on others, so it comes later
		// We'll reverse at the end
		result = append(result, node)

		// For each dependency of this node, reduce its in-degree
		for _, dep := range g.edges[node] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	// Check if all nodes were processed (cycle exists if not)
	if len(result) != len(g.nodes) {
		return nil, ErrCyclicDependency
	}

	// Reverse the result so dependencies come first
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result, nil
}

// Nodes returns all node IDs in the graph.
func (g *Graph) Nodes() []string {
	return g.sortedNodes()
}

// sortedNodes returns nodes in sorted order for deterministic operations.
func (g *Graph) sortedNodes() []string {
	nodes := make([]string, 0, len(g.nodes))
	for node := range g.nodes {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)
	return nodes
}
