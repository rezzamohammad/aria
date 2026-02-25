package claudemd

import (
	"fmt"
	"strings"

	"github.com/aria-cli/aria/internal/task"
)

// Generate produces a CLAUDE.md context file for a specific task.
// This is injected into the agent's working directory before execution.
func Generate(t *task.Task, projectName string, priorWork string) string {
	fileScope := ""
	if len(t.ContextRefs) > 0 {
		fileScope = strings.Join(t.ContextRefs, "\n- ")
		fileScope = "- " + fileScope
	} else {
		fileScope = "- (no explicit scope — use best judgment)"
	}

	deps := t.DependenciesStr()

	retryNote := ""
	if t.RecoveryCount > 0 {
		retryNote = fmt.Sprintf("\n> ⚠️  This is retry #%d. Previous attempt failed: %s\n",
			t.RecoveryCount, t.FailureReason)
	}

	return fmt.Sprintf(`# ARIA Task Context

## Project
%s

## Your Task
- **ID**: %s
- **Title**: %s
- **Role**: %s
- **Priority**: %s
- **Dependencies completed**: %s
%s
## Goal
%s

## Files You Are Allowed to Touch
%s

## Prior Work Summary
%s

## Rules
- Complete ONLY the goal described above
- Do NOT modify files outside your scope list
- Commit your work with message: aria(%s): <short description>
- Add tests for any new public functions
- If blocked, write a BLOCKED.md in the worktree root explaining why

## Scoring (for Verifier)
Your output will be scored 0-100 on: correctness, test coverage, code style, scope adherence.
Minimum passing score is 80.
`, projectName,
		t.ID, t.Title, t.Role, t.PriorityLabel(), deps,
		retryNote, t.Description,
		fileScope, priorWork, t.ID)
}
