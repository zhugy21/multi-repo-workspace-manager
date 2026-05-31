package cli

import (
	"github.com/spf13/cobra"
)

// Package-level flag variables used across CLI commands.
var (
	workspaceFlag string
	allFlag       bool
	groupFlag     string
	repoFlag      string
	concurrency   int
	timeoutFlag   string
	failFast      bool
	continueOnErr bool
	outputFlag    string
	formatFlag    string
	checkoutFlag  string
	forceFlag     bool
	initName      string
)

// NewRootCommand creates the root cobra command for the ws CLI.
func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ws",
		Short:         "Multi-repo workspace manager",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().StringVar(&workspaceFlag, "workspace", "", "path to workspace configuration file")
	cmd.PersistentFlags().BoolVar(&allFlag, "all", true, "operate on all repositories")
	cmd.PersistentFlags().StringVar(&groupFlag, "group", "", "filter repositories by group")
	cmd.PersistentFlags().StringVar(&repoFlag, "repo", "", "filter repositories by name")
	cmd.PersistentFlags().IntVar(&concurrency, "concurrency", 0, "max concurrent tasks (0 = use config default)")
	cmd.PersistentFlags().StringVar(&timeoutFlag, "timeout", "", "global timeout (e.g. 120s)")
	cmd.PersistentFlags().BoolVar(&failFast, "fail-fast", false, "stop on first failure")
	cmd.PersistentFlags().BoolVar(&continueOnErr, "continue-on-error", true, "continue running on error")
	cmd.PersistentFlags().StringVar(&outputFlag, "output", "", "output mode: tui|plain|json")
	cmd.PersistentFlags().StringVar(&formatFlag, "format", "table", "output format")

	cmd.AddCommand(newSyncCmd())
	cmd.AddCommand(newBuildCmd())
	cmd.AddCommand(newHealthCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newMonitorCmd())
	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newAddCmd())
	cmd.AddCommand(newRemoveCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newSwitchCmd())
	cmd.AddCommand(newConfigCmd())

	return cmd
}
