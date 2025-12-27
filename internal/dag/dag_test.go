package dag

import (
	"errors"
	"testing"
)

func TestNew(t *testing.T) {
	g := New()
	if g == nil {
		t.Fatal("New() returned nil")
	}
	if g.nodes == nil {
		t.Error("nodes map is nil")
	}
	if g.edges == nil {
		t.Error("edges map is nil")
	}
}

func TestGraph_AddNode(t *testing.T) {
	g := New()
	g.AddNode("build")
	g.AddNode("test")

	if !g.nodes["build"] {
		t.Error("build node not added")
	}
	if !g.nodes["test"] {
		t.Error("test node not added")
	}

	nodes := g.Nodes()
	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(nodes))
	}
}

func TestGraph_AddEdge(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Graph)
		from    string
		to      string
		wantErr error
	}{
		{
			name: "valid edge",
			setup: func(g *Graph) {
				g.AddNode("build")
				g.AddNode("test")
			},
			from:    "test",
			to:      "build",
			wantErr: nil,
		},
		{
			name: "self dependency",
			setup: func(g *Graph) {
				g.AddNode("build")
			},
			from:    "build",
			to:      "build",
			wantErr: ErrSelfDependency,
		},
		{
			name: "unknown from node",
			setup: func(g *Graph) {
				g.AddNode("build")
			},
			from:    "unknown",
			to:      "build",
			wantErr: ErrUnknownDependency,
		},
		{
			name: "unknown to node",
			setup: func(g *Graph) {
				g.AddNode("build")
			},
			from:    "build",
			to:      "unknown",
			wantErr: ErrUnknownDependency,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := New()
			tt.setup(g)

			err := g.AddEdge(tt.from, tt.to)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestGraph_GetDependencies(t *testing.T) {
	g := New()
	g.AddNode("build")
	g.AddNode("test")
	g.AddNode("deploy")
	_ = g.AddEdge("test", "build")
	_ = g.AddEdge("deploy", "test")
	_ = g.AddEdge("deploy", "build")

	deps := g.GetDependencies("deploy")
	if len(deps) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(deps))
	}

	deps = g.GetDependencies("build")
	if len(deps) != 0 {
		t.Errorf("expected 0 dependencies for build, got %d", len(deps))
	}
}

func TestGraph_GetAllDependencies(t *testing.T) {
	g := New()
	g.AddNode("build")
	g.AddNode("test")
	g.AddNode("deploy")
	_ = g.AddEdge("test", "build")
	_ = g.AddEdge("deploy", "test")

	deps := g.GetAllDependencies("deploy")
	if len(deps) != 2 {
		t.Errorf("expected 2 transitive dependencies, got %d: %v", len(deps), deps)
	}

	// Check both build and test are in deps
	hasTest := false
	hasBuild := false
	for _, d := range deps {
		if d == "test" {
			hasTest = true
		}
		if d == "build" {
			hasBuild = true
		}
	}
	if !hasTest || !hasBuild {
		t.Errorf("expected both test and build in dependencies, got %v", deps)
	}
}

func TestGraph_HasCycle(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*Graph)
		wantCycle bool
	}{
		{
			name: "no cycle - linear",
			setup: func(g *Graph) {
				g.AddNode("a")
				g.AddNode("b")
				g.AddNode("c")
				_ = g.AddEdge("b", "a")
				_ = g.AddEdge("c", "b")
			},
			wantCycle: false,
		},
		{
			name: "no cycle - diamond",
			setup: func(g *Graph) {
				g.AddNode("a")
				g.AddNode("b")
				g.AddNode("c")
				g.AddNode("d")
				_ = g.AddEdge("b", "a")
				_ = g.AddEdge("c", "a")
				_ = g.AddEdge("d", "b")
				_ = g.AddEdge("d", "c")
			},
			wantCycle: false,
		},
		{
			name: "simple cycle",
			setup: func(g *Graph) {
				g.AddNode("a")
				g.AddNode("b")
				// Manually create cycle by bypassing AddEdge
				g.edges["a"] = []string{"b"}
				g.edges["b"] = []string{"a"}
			},
			wantCycle: true,
		},
		{
			name: "three node cycle",
			setup: func(g *Graph) {
				g.AddNode("a")
				g.AddNode("b")
				g.AddNode("c")
				g.edges["a"] = []string{"b"}
				g.edges["b"] = []string{"c"}
				g.edges["c"] = []string{"a"}
			},
			wantCycle: true,
		},
		{
			name: "empty graph",
			setup: func(g *Graph) {
				// empty
			},
			wantCycle: false,
		},
		{
			name: "single node no edges",
			setup: func(g *Graph) {
				g.AddNode("a")
			},
			wantCycle: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := New()
			tt.setup(g)

			if got := g.HasCycle(); got != tt.wantCycle {
				t.Errorf("HasCycle() = %v, want %v", got, tt.wantCycle)
			}
		})
	}
}

func TestGraph_FindCycle(t *testing.T) {
	g := New()
	g.AddNode("a")
	g.AddNode("b")
	g.AddNode("c")
	g.edges["a"] = []string{"b"}
	g.edges["b"] = []string{"c"}
	g.edges["c"] = []string{"a"}

	cycle := g.FindCycle()
	if cycle == nil {
		t.Fatal("expected cycle to be found")
	}
	if len(cycle) < 3 {
		t.Errorf("cycle should contain at least 3 nodes, got %v", cycle)
	}
}

func TestGraph_TopologicalSort(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*Graph)
		wantLen   int
		wantErr   bool
		checkFunc func([]string, *Graph) error // custom validation
	}{
		{
			name: "linear dependency",
			setup: func(g *Graph) {
				g.AddNode("build")
				g.AddNode("test")
				g.AddNode("deploy")
				_ = g.AddEdge("test", "build")
				_ = g.AddEdge("deploy", "test")
			},
			wantLen: 3,
			wantErr: false,
			checkFunc: func(result []string, g *Graph) error {
				return validateTopologicalOrder(result, g)
			},
		},
		{
			name: "diamond dependency",
			setup: func(g *Graph) {
				g.AddNode("build")
				g.AddNode("lint")
				g.AddNode("test")
				g.AddNode("deploy")
				_ = g.AddEdge("lint", "build")
				_ = g.AddEdge("test", "build")
				_ = g.AddEdge("deploy", "lint")
				_ = g.AddEdge("deploy", "test")
			},
			wantLen: 4,
			wantErr: false,
			checkFunc: func(result []string, g *Graph) error {
				return validateTopologicalOrder(result, g)
			},
		},
		{
			name: "no dependencies",
			setup: func(g *Graph) {
				g.AddNode("a")
				g.AddNode("b")
				g.AddNode("c")
			},
			wantLen: 3,
			wantErr: false,
			checkFunc: func(result []string, g *Graph) error {
				// All nodes should be present
				if len(result) != 3 {
					return errors.New("expected 3 nodes")
				}
				return nil
			},
		},
		{
			name: "empty graph",
			setup: func(g *Graph) {
				// empty
			},
			wantLen: 0,
			wantErr: false,
		},
		{
			name: "cycle detection",
			setup: func(g *Graph) {
				g.AddNode("a")
				g.AddNode("b")
				g.edges["a"] = []string{"b"}
				g.edges["b"] = []string{"a"}
			},
			wantLen: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := New()
			tt.setup(g)

			got, err := g.TopologicalSort()
			if (err != nil) != tt.wantErr {
				t.Errorf("TopologicalSort() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != tt.wantLen {
					t.Errorf("TopologicalSort() returned %d nodes, want %d", len(got), tt.wantLen)
				}
				if tt.checkFunc != nil {
					if err := tt.checkFunc(got, g); err != nil {
						t.Errorf("TopologicalSort() order validation failed: %v, got %v", err, got)
					}
				}
			}
		})
	}
}

// validateTopologicalOrder checks that dependencies come before dependents.
func validateTopologicalOrder(result []string, g *Graph) error {
	position := make(map[string]int)
	for i, node := range result {
		position[node] = i
	}

	for node, deps := range g.edges {
		for _, dep := range deps {
			if position[dep] >= position[node] {
				return errors.New("dependency " + dep + " should come before " + node)
			}
		}
	}
	return nil
}

func TestGraph_TopologicalSort_ValidatesOrder(t *testing.T) {
	// Create a complex graph and verify the order is valid
	g := New()
	g.AddNode("a")
	g.AddNode("b")
	g.AddNode("c")
	g.AddNode("d")
	g.AddNode("e")

	// e -> d -> b -> a
	//        \-> c -> a
	_ = g.AddEdge("b", "a")
	_ = g.AddEdge("c", "a")
	_ = g.AddEdge("d", "b")
	_ = g.AddEdge("d", "c")
	_ = g.AddEdge("e", "d")

	result, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify order: for each edge from->to, "to" must come before "from"
	position := make(map[string]int)
	for i, node := range result {
		position[node] = i
	}

	edges := map[string][]string{
		"b": {"a"},
		"c": {"a"},
		"d": {"b", "c"},
		"e": {"d"},
	}

	for from, tos := range edges {
		for _, to := range tos {
			if position[to] >= position[from] {
				t.Errorf("invalid order: %s (pos %d) should come before %s (pos %d)",
					to, position[to], from, position[from])
			}
		}
	}
}
