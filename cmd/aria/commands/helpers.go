package commands

import (
	"fmt"
	"os"

	"github.com/aria-cli/aria/internal/config"
)

// loadConfig finds and loads the aria.toml configuration.
func loadConfig() (*config.Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	cfg, err := config.Load(cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w\nRun 'aria init' to initialize a project", err)
	}

	return cfg, nil
}
