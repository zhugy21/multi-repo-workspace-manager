package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/user/ws/internal/workspace"
)

func newSwitchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "switch <workspace_name>",
		Short: "Switch to a different workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("cannot get home directory: %w", err)
			}

			symlinkPath := filepath.Join(home, ".ws", "workspaces", name+".yaml")
			resolvedPath, err := os.Readlink(symlinkPath)
			if err != nil {
				return fmt.Errorf("failed to resolve workspace symlink: %w", err)
			}

			wsDir := filepath.Join(home, ".ws")
			if err := os.MkdirAll(wsDir, 0755); err != nil {
				return fmt.Errorf("failed to create .ws directory: %w", err)
			}

			ac := workspace.ActiveConfig{Active: resolvedPath}
			data, err := yaml.Marshal(&ac)
			if err != nil {
				return fmt.Errorf("failed to marshal active config: %w", err)
			}

			configPath := filepath.Join(wsDir, "config.yaml")
			if err := os.WriteFile(configPath, data, 0644); err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}

			fmt.Printf("Switched to workspace '%s' (%s)\n", name, resolvedPath)
			return nil
		},
	}
	return cmd
}
