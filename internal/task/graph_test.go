package task

import (
	"testing"
)

func buildTestGraph() *DependencyGraph {
	g := &DependencyGraph{
		nodes: make(map[string]*Task),
		edges: make(map[string][]string),
		deps:  make(map[string][]string),
	}

	// T-001 -> T-002 -> T-003
	// T-001 -> T-004
	tasks := []*Task{
		{ID: "T-001", Title: "First", Role: "worker", Status: StatusDone, Priority: 10},
		{ID: "T-002", Title: "Second", Role: "worker", Status: StatusPending, Priority: 20, Dependencies: []string{"T-001"}},
		{ID: "T-003", Title: "Third", Role: "worker", Status: StatusPending, Priority: 30, Dependencies: []string{"T-002"}},
		{ID: "T-004", Title: "Fourth", Role: "worker", Status: StatusPending, Priority: 15, Dependencies: []string{"T-001"}},
	}

	for _, t := range tasks {
		g.nodes[t.ID] = t
		g.deps[t.ID] = t.Dependencies
		for _, dep := range t.Dependencies {
			g.edges[dep] = append(g.edges[dep], t.ID)
		}
	}

	return g
}

func TestReady(t *testing.T) {
	g := buildTestGraph()

	ready := g.Ready()

	// T-001 is done, T-002 and T-004 depend on T-001 (done), so both are ready
	// T-003 depends on T-002 (pending), so not ready
	if len(ready) != 2 {
		t.Fatalf("expected 2 ready tasks, got %d", len(ready))
	}

	ids := map[string]bool{}
	for _, task := range ready {
		ids[task.ID] = true
	}

	if !ids["T-002"] {
		t.Error("expected T-002 to be ready")
	}
	if !ids["T-004"] {
		t.Error("expected T-004 to be ready")
	}
}

func TestDependents(t *testing.T) {
	g := buildTestGraph()

	deps := g.Dependents("T-001")
	if len(deps) != 2 {
		t.Fatalf("expected 2 dependents of T-001, got %d", len(deps))
	}

	ids := map[string]bool{}
	for _, task := range deps {
		ids[task.ID] = true
	}

	if !ids["T-002"] || !ids["T-004"] {
		t.Error("expected T-002 and T-004 as dependents of T-001")
	}
}

func TestHasCycleNoCycle(t *testing.T) {
	g := buildTestGraph()
	if g.HasCycle() {
		t.Error("expected no cycle in valid graph")
	}
}

func TestHasCycleWithCycle(t *testing.T) {
	g := &DependencyGraph{
		nodes: make(map[string]*Task),
		edges: make(map[string][]string),
		deps:  make(map[string][]string),
	}

	// A -> B -> C -> A (cycle)
	tasks := []*Task{
		{ID: "A", Title: "A", Role: "worker", Status: StatusPending, Dependencies: []string{"C"}},
		{ID: "B", Title: "B", Role: "worker", Status: StatusPending, Dependencies: []string{"A"}},
		{ID: "C", Title: "C", Role: "worker", Status: StatusPending, Dependencies: []string{"B"}},
	}

	for _, t := range tasks {
		g.nodes[t.ID] = t
		g.deps[t.ID] = t.Dependencies
		for _, dep := range t.Dependencies {
			g.edges[dep] = append(g.edges[dep], t.ID)
		}
	}

	if !g.HasCycle() {
		t.Error("expected cycle detection")
	}
}

func TestTopologicalOrder(t *testing.T) {
	g := buildTestGraph()

	order, err := g.TopologicalOrder()
	if err != nil {
		t.Fatalf("TopologicalOrder error: %v", err)
	}

	if len(order) != 4 {
		t.Fatalf("expected 4 tasks in order, got %d", len(order))
	}

	// T-001 should come before T-002 and T-004
	// T-002 should come before T-003
	positions := map[string]int{}
	for i, task := range order {
		positions[task.ID] = i
	}

	if positions["T-001"] > positions["T-002"] {
		t.Error("T-001 should come before T-002")
	}
	if positions["T-001"] > positions["T-004"] {
		t.Error("T-001 should come before T-004")
	}
	if positions["T-002"] > positions["T-003"] {
		t.Error("T-002 should come before T-003")
	}
}

func TestTopologicalOrderWithCycle(t *testing.T) {
	g := &DependencyGraph{
		nodes: make(map[string]*Task),
		edges: make(map[string][]string),
		deps:  make(map[string][]string),
	}

	g.nodes["A"] = &Task{ID: "A", Dependencies: []string{"B"}}
	g.nodes["B"] = &Task{ID: "B", Dependencies: []string{"A"}}
	g.deps["A"] = []string{"B"}
	g.deps["B"] = []string{"A"}

	_, err := g.TopologicalOrder()
	if err == nil {
		t.Error("expected error for cyclic graph")
	}
}

func TestReadyNoTasks(t *testing.T) {
	g := &DependencyGraph{
		nodes: make(map[string]*Task),
		edges: make(map[string][]string),
		deps:  make(map[string][]string),
	}

	ready := g.Ready()
	if len(ready) != 0 {
		t.Errorf("expected 0 ready tasks, got %d", len(ready))
	}
}

func TestReadyAllDone(t *testing.T) {
	g := &DependencyGraph{
		nodes: make(map[string]*Task),
		edges: make(map[string][]string),
		deps:  make(map[string][]string),
	}

	g.nodes["T-001"] = &Task{ID: "T-001", Status: StatusDone}
	g.deps["T-001"] = nil

	ready := g.Ready()
	if len(ready) != 0 {
		t.Errorf("expected 0 ready tasks (all done), got %d", len(ready))
	}
}
