package roles

// Role represents an agent's specialized role.
type Role string

const (
	RoleOrchestrator Role = "orchestrator"
	RoleWorker       Role = "worker"
	RoleVerifier     Role = "verifier"
	RoleArchitect    Role = "architect"
	RoleResearcher   Role = "researcher"
	RolePM           Role = "pm"
	RoleQA           Role = "qa"
	RoleSecurity     Role = "security"
	RoleLibrarian    Role = "librarian"
	RoleDocSystem    Role = "doc_system"
	RoleReleaseOps   Role = "release_ops"
)

// AllRoles lists every defined role.
var AllRoles = []Role{
	RoleOrchestrator, RoleWorker, RoleVerifier, RoleArchitect,
	RoleResearcher, RolePM, RoleQA, RoleSecurity,
	RoleLibrarian, RoleDocSystem, RoleReleaseOps,
}

// IsValidRole checks if a string is a valid role.
func IsValidRole(s string) bool {
	for _, r := range AllRoles {
		if string(r) == s {
			return true
		}
	}
	return false
}

// DefaultToolMapping provides the default CLI tool chain per role.
// First tool is preferred; subsequent are fallbacks.
var DefaultToolMapping = map[Role][]string{
	RoleOrchestrator: {"claude-code"},
	RoleWorker:       {"claude-code", "codex", "crush"},
	RoleVerifier:     {"claude-code"},
	RoleArchitect:    {"claude-code", "gemini-cli"},
	RoleResearcher:   {"gemini-cli", "claude-code"},
	RolePM:           {"claude-code"},
	RoleQA:           {"codex", "claude-code"},
	RoleSecurity:     {"claude-code"},
	RoleLibrarian:    {"claude-code"},
	RoleDocSystem:    {"claude-code"},
	RoleReleaseOps:   {"droids", "claude-code"},
}

// RoleDescription returns a human-readable description of the role.
func RoleDescription(r Role) string {
	switch r {
	case RoleOrchestrator:
		return "Decomposes, delegates, reviews, decides"
	case RoleWorker:
		return "Implementation, code generation"
	case RoleVerifier:
		return "Score-gated verification (>=80 threshold)"
	case RoleArchitect:
		return "System design, API contracts"
	case RoleResearcher:
		return "Tech evaluation, best practices"
	case RolePM:
		return "PRD validation, acceptance criteria"
	case RoleQA:
		return "Testing, quality assurance"
	case RoleSecurity:
		return "Security review, vulnerability detection"
	case RoleLibrarian:
		return "Memory indexing, context routing"
	case RoleDocSystem:
		return "Documentation generation"
	case RoleReleaseOps:
		return "Deployment, release management"
	default:
		return "Unknown role"
	}
}
