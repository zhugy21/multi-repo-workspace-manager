package cli

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/user/ws/internal/executor"
	"github.com/user/ws/internal/runner"
	"github.com/user/ws/internal/workspace"
)

func newSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync git repositories (fetch + merge --ff-only)",
		Long: `Sync all or selected repositories. Without --checkout, runs git fetch + merge --ff-only.
With --checkout, switches repositories to the specified branch.`,
		RunE: runSync,
	}
	cmd.Flags().StringVar(&checkoutFlag, "checkout", "", "Checkout branch on all matching repos")
	cmd.Flags().BoolVar(&forceFlag, "force", false, "Force checkout (stash dirty changes)")
	return cmd
}

func runSync(cmd *cobra.Command, args []string) error {
	wsPath, err := resolveWorkspacePath()
	if err != nil {
		return err
	}
	ws, err := workspace.Parse(wsPath)
	if err != nil {
		return err
	}
	r := &runner.SyncRunner{
		Checkout: checkoutFlag,
		Force:    forceFlag,
	}
	tasks, err := r.Prepare(context.Background(), ws, resolveFilter())
	if err != nil {
		return err
	}
	exec := &executor.Executor{
		Concurrency: resolveConcurrency(),
		FailFast:    resolveFailFast(),
	}
	if exec.Concurrency <= 0 {
		exec.Concurrency = ws.Config.DefaultConcurrency
	}
	results := exec.Run(context.Background(), tasks)
	displayResults(results, resolveOutputFormat(ws), r.Name())
	return exitCodeFromResults(results)
}
