package main

import (
	"fmt"
	"os"

	"github.com/aria-cli/aria/cmd/aria/commands"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "aria",
	Short: "ARIA — Autonomous Routing & Intelligence Agent",
	Long: `ARIA is a CLI/TUI orchestration layer that sits on top of existing coding
CLI tools (Claude Code, Codex, Gemini CLI, Crush, Kiro, Droids) and manages
up to 20 specialized agents autonomously with persistent memory, consistent
prompting, auto-recovery, and parallel git worktrees.`,
}

func init() {
	rootCmd.AddCommand(commands.InitCmd)
	rootCmd.AddCommand(commands.IngestCmd)
	rootCmd.AddCommand(commands.TaskCmd)
	rootCmd.AddCommand(commands.AgentCmd)
	rootCmd.AddCommand(commands.RunCmd)
	rootCmd.AddCommand(commands.MemoryCmd)
	rootCmd.AddCommand(commands.WorktreeCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
