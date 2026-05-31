package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/user/ws/internal/workspace"
)

type workspaceEntry struct {
	Name     string    `json:"name"`
	Repos    int       `json:"repos"`
	LastUsed time.Time `json:"last_used"`
}

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workspaces",
		RunE: func(cmd *cobra.Command, args []string) error {
			format, _ := cmd.Flags().GetString("format")

			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("cannot get home directory: %w", err)
			}

			workspacesDir := filepath.Join(home, ".ws", "workspaces")
			entries, err := os.ReadDir(workspacesDir)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("No workspaces found")
					return nil
				}
				return fmt.Errorf("failed to read workspaces directory: %w", err)
			}

			var workspaces []workspaceEntry
			for _, entry := range entries {
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
					continue
				}

				wsPath := filepath.Join(workspacesDir, entry.Name())
				data, err := os.ReadFile(wsPath)
				if err != nil {
					continue
				}

				var ws workspace.Workspace
				if err := yaml.Unmarshal(data, &ws); err != nil {
					continue
				}

				info, err := entry.Info()
				if err != nil {
					continue
				}

				workspaces = append(workspaces, workspaceEntry{
					Name:     ws.WorkspaceName,
					Repos:    len(ws.Repos),
					LastUsed: info.ModTime(),
				})
			}

			sort.Slice(workspaces, func(i, j int) bool {
				return workspaces[i].Name < workspaces[j].Name
			})

			if format == "json" {
				if len(workspaces) == 0 {
					fmt.Println("[]")
					return nil
				}
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(workspaces)
			}

			// Table format.
			fmt.Printf("%-20s  %-5s  %s\n", "NAME", "REPOS", "LAST USED")
			for _, w := range workspaces {
				fmt.Printf("%-20s  %-5d  %s\n", w.Name, w.Repos, w.LastUsed.Format("2006-01-02 15:04"))
			}

			if len(workspaces) == 0 {
				fmt.Println("No workspaces found")
			}

			return nil
		},
	}
	cmd.Flags().String("format", "table", "output format: table|json")
	return cmd
}
