package commands

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/aria-cli/aria/internal/agent"
	"github.com/aria-cli/aria/internal/agent/roles"
	"github.com/aria-cli/aria/internal/db"
	"github.com/aria-cli/aria/internal/db/queries"
	"github.com/spf13/cobra"
)

// AgentCmd is the parent command for agent management.
var AgentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Agent management commands",
}

var agentSpawnRole string
var agentSpawnTool string

func init() {
	// agent spawn
	spawnCmd := &cobra.Command{
		Use:   "spawn",
		Short: "Spawn a new agent",
		RunE:  runAgentSpawn,
	}
	spawnCmd.Flags().StringVar(&agentSpawnRole, "role", "worker", "Agent role")
	spawnCmd.Flags().StringVar(&agentSpawnTool, "tool", "", "CLI tool override (default: auto-resolve)")
	AgentCmd.AddCommand(spawnCmd)

	// agent list
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all agents",
		RunE:  runAgentList,
	}
	AgentCmd.AddCommand(listCmd)

	// agent kill
	killCmd := &cobra.Command{
		Use:   "kill [agent-id]",
		Short: "Kill an agent",
		Args:  cobra.ExactArgs(1),
		RunE:  runAgentKill,
	}
	AgentCmd.AddCommand(killCmd)
}

func runAgentSpawn(cmd *cobra.Command, args []string) error {
	if !roles.IsValidRole(agentSpawnRole) {
		return fmt.Errorf("invalid role %q — valid roles: %v", agentSpawnRole, roles.AllRoles)
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	database, err := db.Open(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	router := agent.NewRouter(cfg.ToolMapping)
	pool := agent.NewPool(database, router, cfg.Agents.MaxPoolSize)

	ap, err := pool.SpawnAgent(roles.Role(agentSpawnRole), agentSpawnTool)
	if err != nil {
		return fmt.Errorf("failed to spawn agent: %w", err)
	}

	fmt.Printf("✓ Spawned agent %s (role: %s, tool: %s)\n", ap.ID[:8], ap.Role, ap.CLITool)
	return nil
}

func runAgentList(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	database, err := db.Open(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	agents, err := queries.ListAgents(database, "")
	if err != nil {
		return fmt.Errorf("failed to list agents: %w", err)
	}

	if len(agents) == 0 {
		fmt.Println("No agents registered.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tROLE\tSTATUS\tTOOL\tTASK\tTOTAL\tAVG SCORE\n")
	fmt.Fprintf(w, "──\t────\t──────\t────\t────\t─────\t─────────\n")

	for _, a := range agents {
		shortID := a.ID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}

		taskStr := "—"
		if a.CurrentTaskID.Valid && a.CurrentTaskID.String != "" {
			taskStr = a.CurrentTaskID.String
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%.1f\n",
			shortID, a.Role, a.Status, a.CLITool, taskStr,
			a.TotalTasks, a.AvgScore)
	}

	w.Flush()

	// Summary
	counts, _ := queries.CountAgentsByStatus(database)
	fmt.Printf("\nTotal: %d", len(agents))
	for status, count := range counts {
		fmt.Printf(" | %s: %d", status, count)
	}
	fmt.Println()

	return nil
}

func runAgentKill(cmd *cobra.Command, args []string) error {
	agentID := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	database, err := db.Open(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	// Update status to offline
	if err := queries.UpdateAgentStatus(database, agentID, "offline", "", ""); err != nil {
		return fmt.Errorf("failed to kill agent: %w", err)
	}

	fmt.Printf("✓ Agent %s marked as offline\n", agentID)
	return nil
}
