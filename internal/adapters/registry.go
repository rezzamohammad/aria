package adapters

import "fmt"

// Registry holds all discovered adapters and routes jobs to the best one.
type Registry struct {
	adapters map[string]AgentAdapter
}

// NewRegistry auto-discovers all CLI tools available in PATH.
func NewRegistry() *Registry {
	r := &Registry{adapters: make(map[string]AgentAdapter)}

	candidates := []AgentAdapter{
		&ClaudeAdapter{},
		&CodexAdapter{},
		&GeminiAdapter{},
		&CrushAdapter{},
		&DroidsAdapter{},
	}

	for _, a := range candidates {
		if a.IsAvailable() {
			r.adapters[a.Name()] = a
		}
	}

	return r
}

// capabilityPriority defines the preferred adapter order per capability.
// Mirrors DefaultToolMapping in roles package.
var capabilityPriority = map[Capability][]string{
	CapCode:     {"claude-code", "codex", "crush", "gemini-cli"},
	CapResearch: {"gemini-cli", "claude-code"},
	CapTest:     {"codex", "claude-code"},
	CapReview:   {"claude-code", "codex"},
	CapRefactor: {"claude-code", "codex", "crush"},
	CapDocs:     {"gemini-cli", "claude-code"},
	CapSecurity: {"claude-code"},
	CapRelease:  {"droids", "claude-code"},
}

// BestFor returns the highest-priority available adapter for a capability.
func (r *Registry) BestFor(cap Capability) (AgentAdapter, error) {
	for _, name := range capabilityPriority[cap] {
		if a, ok := r.adapters[name]; ok {
			return a, nil
		}
	}
	return nil, fmt.Errorf("no installed adapter for capability %q", cap)
}

// ByName returns a specific adapter by tool name.
func (r *Registry) ByName(name string) (AgentAdapter, bool) {
	a, ok := r.adapters[name]
	return a, ok
}

// Available returns all adapters that passed IsAvailable() during init.
func (r *Registry) Available() []AgentAdapter {
	out := make([]AgentAdapter, 0, len(r.adapters))
	for _, a := range r.adapters {
		out = append(out, a)
	}
	return out
}

// Count returns the number of available adapters.
func (r *Registry) Count() int {
	return len(r.adapters)
}
