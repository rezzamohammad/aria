package agent

import (
	"fmt"
	"os/exec"

	"github.com/aria-cli/aria/internal/agent/roles"
)

// Router maps roles to CLI tools and resolves which tool to use.
type Router struct {
	mapping map[roles.Role][]string
}

// NewRouter creates a router with the given tool mapping.
// If mapping is nil, uses DefaultToolMapping.
func NewRouter(mapping map[string][]string) *Router {
	r := &Router{
		mapping: make(map[roles.Role][]string),
	}

	if mapping == nil {
		r.mapping = roles.DefaultToolMapping
	} else {
		for k, v := range mapping {
			r.mapping[roles.Role(k)] = v
		}
	}

	return r
}

// ResolveTool returns the best available CLI tool for a given role.
// It checks if each tool in the fallback chain is installed on the system.
func (r *Router) ResolveTool(role roles.Role) (string, error) {
	tools, ok := r.mapping[role]
	if !ok {
		return "", fmt.Errorf("no tool mapping for role %q", role)
	}

	for _, tool := range tools {
		if isToolAvailable(tool) {
			return tool, nil
		}
	}

	// Return first tool even if not found — let the spawn fail with a clear error
	return tools[0], fmt.Errorf("no installed CLI tool found for role %q (tried: %v)", role, tools)
}

// ResolveToolOrDefault returns the tool for a role, or the first in the chain if none available.
func (r *Router) ResolveToolOrDefault(role roles.Role) string {
	tool, err := r.ResolveTool(role)
	if err != nil {
		tools := r.mapping[role]
		if len(tools) > 0 {
			return tools[0]
		}
		return "claude-code" // ultimate fallback
	}
	return tool
}

// GetToolChain returns the full fallback chain for a role.
func (r *Router) GetToolChain(role roles.Role) []string {
	if tools, ok := r.mapping[role]; ok {
		return tools
	}
	return nil
}

// SetToolMapping overrides the tool chain for a specific role.
func (r *Router) SetToolMapping(role roles.Role, tools []string) {
	r.mapping[role] = tools
}

// isToolAvailable checks if a CLI tool is installed and in PATH.
func isToolAvailable(tool string) bool {
	_, err := exec.LookPath(tool)
	return err == nil
}
