package task

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// DependencyGraph represents the task dependency DAG.
type DependencyGraph struct {
	nodes map[string]*Task
	edges map[string][]string // taskID -> list of tasks that depend on it
	deps  map[string][]string // taskID -> list of tasks it depends on
}

// BuildGraph constructs a dependency graph from all tasks in the database.
func BuildGraph(db *sql.DB) (*DependencyGraph, error) {
	rows, err := db.Query(`
		SELECT id, title, role, status, priority, dependencies
		FROM tasks ORDER BY priority ASC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer rows.Close()

	g := &DependencyGraph{
		nodes: make(map[string]*Task),
		edges: make(map[string][]string),
		deps:  make(map[string][]string),
	}

	for rows.Next() {
		t := &Task{}
		var depsStr sql.NullString
		if err := rows.Scan(&t.ID, &t.Title, &t.Role, &t.Status, &t.Priority, &depsStr); err != nil {
			return nil, err
		}

		if depsStr.Valid && depsStr.String != "" {
			json.Unmarshal([]byte(depsStr.String), &t.Dependencies)
		}

		g.nodes[t.ID] = t
		g.deps[t.ID] = t.Dependencies

		// Build reverse edges
		for _, dep := range t.Dependencies {
			g.edges[dep] = append(g.edges[dep], t.ID)
		}
	}

	return g, nil
}

// Ready returns all tasks whose dependencies are all done.
func (g *DependencyGraph) Ready() []*Task {
	var ready []*Task
	for id, t := range g.nodes {
		if t.Status != StatusPending {
			continue
		}
		allDone := true
		for _, depID := range g.deps[id] {
			dep, exists := g.nodes[depID]
			if !exists || dep.Status != StatusDone {
				allDone = false
				break
			}
		}
		if allDone {
			ready = append(ready, t)
		}
	}
	return ready
}

// Dependents returns all tasks that depend on the given task ID.
func (g *DependencyGraph) Dependents(taskID string) []*Task {
	var result []*Task
	for _, depID := range g.edges[taskID] {
		if t, exists := g.nodes[depID]; exists {
			result = append(result, t)
		}
	}
	return result
}

// HasCycle checks if the dependency graph contains any cycles.
func (g *DependencyGraph) HasCycle() bool {
	visited := make(map[string]bool)
	inStack := make(map[string]bool)

	var dfs func(id string) bool
	dfs = func(id string) bool {
		visited[id] = true
		inStack[id] = true

		for _, dep := range g.deps[id] {
			if !visited[dep] {
				if dfs(dep) {
					return true
				}
			} else if inStack[dep] {
				return true
			}
		}

		inStack[id] = false
		return false
	}

	for id := range g.nodes {
		if !visited[id] {
			if dfs(id) {
				return true
			}
		}
	}
	return false
}

// TopologicalOrder returns tasks in execution order (dependencies first).
func (g *DependencyGraph) TopologicalOrder() ([]*Task, error) {
	if g.HasCycle() {
		return nil, fmt.Errorf("dependency graph contains a cycle")
	}

	visited := make(map[string]bool)
	var order []*Task

	var visit func(id string)
	visit = func(id string) {
		if visited[id] {
			return
		}
		visited[id] = true
		for _, dep := range g.deps[id] {
			visit(dep)
		}
		if t, exists := g.nodes[id]; exists {
			order = append(order, t)
		}
	}

	for id := range g.nodes {
		visit(id)
	}

	return order, nil
}
