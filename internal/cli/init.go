package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/user/ws/internal/workspace"
	"github.com/user/ws/pkg/types"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			name := initName
			if name == "" {
				fmt.Print("Workspace name: ")
				reader := bufio.NewReader(os.Stdin)
				input, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("failed to read input: %w", err)
				}
				name = strings.TrimSpace(input)
				if name == "" {
					return fmt.Errorf("workspace name is required")
				}
			}

			// Check if workspace.yaml already exists.
			if _, err := os.Stat("workspace.yaml"); err == nil {
				fmt.Print("workspace.yaml already exists. Overwrite? (y/N): ")
				reader := bufio.NewReader(os.Stdin)
				input, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("failed to read input: %w", err)
				}
				input = strings.TrimSpace(input)
				if input != "y" && input != "Y" {
					return fmt.Errorf("aborted by user")
				}
			}

			ws := &workspace.Workspace{
				WorkspaceName: name,
				Description:   "",
				Config: workspace.Config{
					DefaultConcurrency: 4,
					DefaultTimeout:     120 * time.Second,
					FailFast:           false,
					OutputFormat:       "tui",
				},
				Groups: make(map[string][]string),
				Repos:  make([]types.Repo, 0),
			}

			data, err := yaml.Marshal(ws)
			if err != nil {
				return fmt.Errorf("failed to marshal workspace: %w", err)
			}

			if err := os.WriteFile("workspace.yaml", data, 0644); err != nil {
				return fmt.Errorf("failed to write workspace.yaml: %w", err)
			}

			fmt.Printf("Initialized workspace '%s'\n", name)
			return nil
		},
	}
	cmd.Flags().StringVar(&initName, "name", "", "workspace name")
	return cmd
}
