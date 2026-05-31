package cli

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/user/ws/internal/executor"
	"github.com/user/ws/internal/runner"
	"github.com/user/ws/internal/workspace"
)

func newHealthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check health of repositories",
		RunE:  runHealth,
	}
}

func runHealth(cmd *cobra.Command, args []string) error {
	wsPath, err := resolveWorkspacePath()
	if err != nil {
		return err
	}
	ws, err := workspace.Parse(wsPath)
	if err != nil {
		return err
	}
	r := &runner.HealthRunner{}
	tasks, err := r.Prepare(context.Background(), ws, resolveFilter())
	if err != nil {
		return err
	}
	exec := &executor.Executor{Concurrency: resolveConcurrency(), FailFast: resolveFailFast()}
	if exec.Concurrency <= 0 {
		exec.Concurrency = ws.Config.DefaultConcurrency
	}
	results := exec.Run(context.Background(), tasks)
	displayResults(results, resolveOutputFormat(ws), r.Name())
	return exitCodeFromResults(results)
}
