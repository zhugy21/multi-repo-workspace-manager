package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/user/ws/internal/executor"
	"github.com/user/ws/internal/workspace"
	"github.com/user/ws/pkg/types"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show repository status",
		RunE:  runStatus,
	}
}

func runStatus(cmd *cobra.Command, args []string) error {
	wsPath, err := resolveWorkspacePath()
	if err != nil {
		return err
	}
	ws, err := workspace.Parse(wsPath)
	if err != nil {
		return err
	}

	repos := ws.Filter(resolveFilter())
	var tasks []types.Task
	for _, repo := range repos {
		if _, err := os.Stat(repo.Path); err != nil {
			// Path does not exist.
			tasks = append(tasks, types.Task{
				ID:       fmt.Sprintf("%s/status", repo.Name),
				RepoName: repo.Name,
				Group:    repo.Group,
				Command:  []string{"sh", "-c", `echo "MISSING"`},
			})
			continue
		}

		tasks = append(tasks, types.Task{
			ID:       fmt.Sprintf("%s/status", repo.Name),
			RepoName: repo.Name,
			Group:    repo.Group,
			Command:  []string{"sh", "-c", statusScript(repo.Path)},
		})
	}

	exec := &executor.Executor{
		Concurrency: resolveConcurrency(),
		FailFast:    resolveFailFast(),
	}
	if exec.Concurrency <= 0 {
		exec.Concurrency = ws.Config.DefaultConcurrency
	}
	results := exec.Run(context.Background(), tasks)
	displayResults(results, resolveOutputFormat(ws), "status")
	return exitCodeFromResults(results)
}

// statusScript returns a shell script that queries git metadata for the given path.
// The script handles three cases:
//   - path is not a directory (cd fails) → prints "NOT_A_REPO"
//   - path is not a git repository   → prints "NOT_A_REPO"
//   - path is a valid git repository → prints branch, dirty, last_commit
func statusScript(path string) string {
	return fmt.Sprintf(`cd "%s" 2>/dev/null || { echo "NOT_A_REPO"; exit; }
if ! git rev-parse --git-dir >/dev/null 2>&1; then echo "NOT_A_REPO"; exit; fi
echo "branch:$(git branch --show-current 2>/dev/null)"
echo "dirty:$(git status --porcelain 2>/dev/null)"
echo "last_commit:$(git log -1 --format='%%h %%s' 2>/dev/null)"`, path)
}
