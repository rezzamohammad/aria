package roles

import (
	"testing"
)

func TestAllRolesExist(t *testing.T) {
	expectedRoles := []Role{
		RoleOrchestrator, RoleWorker, RoleVerifier, RoleArchitect,
		RoleResearcher, RolePM, RoleQA, RoleSecurity,
		RoleLibrarian, RoleDocSystem, RoleReleaseOps,
	}

	if len(AllRoles) != len(expectedRoles) {
		t.Errorf("expected %d roles, got %d", len(expectedRoles), len(AllRoles))
	}

	for _, role := range expectedRoles {
		found := false
		for _, r := range AllRoles {
			if r == role {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing role %q in AllRoles", role)
		}
	}
}

func TestIsValidRole(t *testing.T) {
	valid := []string{
		"orchestrator", "worker", "verifier", "architect", "researcher",
		"pm", "qa", "security", "librarian", "doc_system", "release_ops",
	}
	for _, r := range valid {
		if !IsValidRole(r) {
			t.Errorf("expected %q to be valid", r)
		}
	}

	invalid := []string{"", "unknown", "admin", "root"}
	for _, r := range invalid {
		if IsValidRole(r) {
			t.Errorf("expected %q to be invalid", r)
		}
	}
}

func TestDefaultToolMapping(t *testing.T) {
	for _, role := range AllRoles {
		tools, ok := DefaultToolMapping[role]
		if !ok {
			t.Errorf("missing default tool mapping for role %q", role)
			continue
		}
		if len(tools) == 0 {
			t.Errorf("empty tool chain for role %q", role)
		}
	}
}

func TestSystemPrompt(t *testing.T) {
	for _, role := range AllRoles {
		prompt := SystemPrompt(role)
		if prompt == "" {
			t.Errorf("empty system prompt for role %q", role)
		}
		// All prompts should have reasonable length
		if len(prompt) < 100 {
			t.Errorf("system prompt for role %q seems too short (%d chars)", role, len(prompt))
		}
	}
}

func TestSystemPromptDefault(t *testing.T) {
	prompt := SystemPrompt(Role("nonexistent"))
	if prompt == "" {
		t.Error("expected fallback prompt for unknown role")
	}
	if prompt != WorkerSystemPrompt {
		t.Error("expected unknown role to fall back to WorkerSystemPrompt")
	}
}

func TestSystemPromptContent(t *testing.T) {
	// Verify specific prompts contain role-appropriate content
	tests := []struct {
		role     Role
		contains string
	}{
		{RoleOrchestrator, "orchestrator"},
		{RoleWorker, "worker"},
		{RoleVerifier, "verif"},
		{RoleArchitect, "architect"},
		{RoleResearcher, "research"},
		{RolePM, "product"},
		{RoleQA, "qa"},
		{RoleSecurity, "security"},
		{RoleLibrarian, "memory"},
		{RoleDocSystem, "document"},
		{RoleReleaseOps, "release"},
	}

	for _, tt := range tests {
		prompt := SystemPrompt(tt.role)
		lower := toLower(prompt)
		if !containsStr(lower, tt.contains) {
			t.Errorf("system prompt for %q should contain %q", tt.role, tt.contains)
		}
	}
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func containsStr(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
