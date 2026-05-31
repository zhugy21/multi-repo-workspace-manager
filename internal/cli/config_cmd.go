package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage workspace configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			edit, _ := cmd.Flags().GetBool("edit")

			wsPath, err := resolveWorkspacePath()
			if err != nil {
				return fmt.Errorf("failed to resolve workspace: %w", err)
			}

			if edit {
				editor := os.Getenv("EDITOR")
				if editor == "" {
					editor = "vim"
				}
				execCmd := exec.Command(editor, wsPath)
				execCmd.Stdin = os.Stdin
				execCmd.Stdout = os.Stdout
				execCmd.Stderr = os.Stderr
				if err := execCmd.Run(); err != nil {
					return fmt.Errorf("editor failed: %w", err)
				}
				return nil
			}

			// Default: --show
			data, err := os.ReadFile(wsPath)
			if err != nil {
				return fmt.Errorf("failed to read workspace: %w", err)
			}
			fmt.Print(string(data))
			return nil
		},
	}
	cmd.Flags().Bool("show", false, "show the current configuration")
	cmd.Flags().Bool("edit", false, "edit the configuration in an editor")
	return cmd
}
