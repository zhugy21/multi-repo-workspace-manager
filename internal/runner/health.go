package runner

import (
	"context"
	"fmt"
	"strings"

	"github.com/user/ws/internal/workspace"
	"github.com/user/ws/pkg/types"
)

type HealthRunner struct{}

func (r *HealthRunner) Name() string { return "health" }

func (r *HealthRunner) Prepare(ctx context.Context, ws *workspace.Workspace, filter types.Filter) ([]types.Task, error) {
	repos := ws.Filter(filter)
	var tasks []types.Task
	for _, repo := range repos {
		if repo.HealthCommand == "" {
			continue
		}
		parts := strings.Fields(repo.HealthCommand)
		timeout := repo.HealthTimeout
		tasks = append(tasks, types.Task{
			ID:       fmt.Sprintf("%s/health", repo.Name),
			RepoName: repo.Name,
			Group:    repo.Group,
			Command:  parts,
			Timeout:  timeout,
		})
	}
	return tasks, nil
}
