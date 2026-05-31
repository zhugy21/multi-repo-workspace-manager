package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/user/ws/internal/workspace"
	"github.com/user/ws/pkg/types"
)

func newRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a repository from the workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			wsPath, err := resolveWorkspacePath()
			if err != nil {
				return fmt.Errorf("failed to resolve workspace: %w", err)
			}

			data, err := os.ReadFile(wsPath)
			if err != nil {
				return fmt.Errorf("failed to read workspace: %w", err)
			}

			var ws workspace.Workspace
			if err := yaml.Unmarshal(data, &ws); err != nil {
				return fmt.Errorf("failed to parse workspace: %w", err)
			}

			found := false
			filtered := make([]types.Repo, 0, len(ws.Repos))
			for _, r := range ws.Repos {
				if r.Name == name {
					found = true
					continue
				}
				filtered = append(filtered, r)
			}

			if !found {
				return fmt.Errorf("repo '%s' not found in workspace", name)
			}

			ws.Repos = filtered

			out, err := yaml.Marshal(&ws)
			if err != nil {
				return fmt.Errorf("failed to marshal workspace: %w", err)
			}

			if err := os.WriteFile(wsPath, out, 0644); err != nil {
				return fmt.Errorf("failed to write workspace: %w", err)
			}

			fmt.Printf("Removed repo '%s' from workspace\n", name)
			return nil
		},
	}
	return cmd
}
