package runner

import (
	"context"
	"fmt"
	"strings"

	"github.com/user/ws/internal/workspace"
	"github.com/user/ws/pkg/types"
)

type BuildRunner struct{}

func (r *BuildRunner) Name() string { return "build" }

func (r *BuildRunner) Prepare(ctx context.Context, ws *workspace.Workspace, filter types.Filter) ([]types.Task, error) {
	repos := ws.Filter(filter)
	var tasks []types.Task
	for _, repo := range repos {
		if repo.BuildCommand == "" {
			continue // v1: no default command by type, just skip
		}
		parts := strings.Fields(repo.BuildCommand)
		timeout := repo.BuildTimeout // 0 means use global default (handled by executor)
		tasks = append(tasks, types.Task{
			ID:       fmt.Sprintf("%s/build", repo.Name),
			RepoName: repo.Name,
			Group:    repo.Group,
			Command:  parts,
			Timeout:  timeout,
		})
	}
	return tasks, nil
}
