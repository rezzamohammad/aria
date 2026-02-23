# ARIA — Autonomous Routing & Intelligence Agent

A CLI/TUI orchestration layer that sits on top of existing coding CLI tools (Claude Code, Codex, Gemini CLI, Crush, Kiro, Droids) and manages up to 20 specialized agents autonomously — with persistent state, consistent prompting, auto-recovery, and parallel git worktrees.

## Phase 1 (This Release)

- `aria init` — Bootstrap project (creates `aria.toml` + SQLite DB)
- `aria ingest <file>` — Extract tasks from any markdown document (PRD, CLAUDE.md, spec)
- `aria task list|add|show|retry|cancel` — Full task CRUD
- `aria agent spawn|list|kill` — Agent lifecycle management
- `aria run` — Launch TUI dashboard (Bubble Tea kanban board)
- `aria run --headless` — Log-only mode for overnight runs
- `aria memory show|add` — Context/memory inspection
- `aria worktree list|clean` — Git worktree management

## Architecture

```
ARIA (orchestration layer)
  ├── Task Queue (SQLite + dependency graph)
  ├── Agent Pool (up to 20, role-based)
  ├── CLI Router (role → tool mapping)
  ├── Handoff Generator (YAML envelopes)
  └── TUI Dashboard (Bubble Tea)
        ├── Kanban board
        ├── Agent status panel
        └── Real-time session log
```

ARIA does **not** build its own AI — it routes to existing CLI tools:
- Claude Code, Codex, Crush, Gemini CLI, Kiro, Droids

## Agent Roles

| Role | Responsibility |
|------|---------------|
| orchestrator | Decomposes, delegates, reviews |
| worker | Implementation, code generation |
| verifier | Score-gated verification (>=80) |
| architect | System design, API contracts |
| researcher | Tech evaluation, best practices |
| pm | PRD validation, acceptance criteria |
| qa | Testing, quality assurance |
| security | Security review |
| librarian | Memory indexing, context routing |
| doc_system | Documentation generation |
| release_ops | Deployment, release management |

## Quick Start

```bash
# Build
go build -o aria ./cmd/aria/

# Initialize a project
./aria init

# Add tasks
./aria task add --title "Implement auth" --role worker --priority 0
./aria ingest PRD.md

# Spawn agents
./aria agent spawn --role worker --tool claude-code
./aria agent spawn --role verifier

# Launch dashboard
./aria run
```

## Tech Stack

- **Go** + Bubble Tea + Lip Gloss (TUI)
- **SQLite** (task queue, agent state, sessions)
- **Cobra** (CLI framework)
- **TOML** (configuration)

## What's Next (Phase 2+)

- Mem0 integration for on-demand context loading
- Auto-recovery + failure classifier
- CLAUDE.md splitter (bootstrap vs cold archive)
- Task Master integration for PRD ingestion
- Smart consideration engine (proactive suggestions)
