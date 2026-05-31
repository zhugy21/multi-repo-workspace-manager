package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/user/ws/internal/workspace"
	"github.com/user/ws/pkg/types"
)

func newAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a repository to the workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			path, _ := cmd.Flags().GetString("path")
			url, _ := cmd.Flags().GetString("url")
			group, _ := cmd.Flags().GetString("group")

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

			repo := types.Repo{
				Name:  name,
				Path:  path,
				URL:   url,
				Group: group,
			}
			ws.Repos = append(ws.Repos, repo)

			out, err := yaml.Marshal(&ws)
			if err != nil {
				return fmt.Errorf("failed to marshal workspace: %w", err)
			}

			if err := os.WriteFile(wsPath, out, 0644); err != nil {
				return fmt.Errorf("failed to write workspace: %w", err)
			}

			fmt.Printf("Added repo '%s' to workspace\n", name)
			return nil
		},
	}
	cmd.Flags().String("name", "", "repository name")
	cmd.Flags().String("path", "", "local path to the repository")
	cmd.Flags().String("url", "", "repository URL")
	cmd.Flags().String("group", "", "group to assign the repository to")
	return cmd
}
