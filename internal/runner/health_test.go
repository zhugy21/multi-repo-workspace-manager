package runner

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/user/ws/internal/workspace"
	"github.com/user/ws/pkg/types"
)

func TestHealthRunnerPrepareWithCommand(t *testing.T) {
	ws := &workspace.Workspace{
		Repos: []types.Repo{
			{Name: "test", Path: "/tmp", HealthCommand: "curl localhost/health"},
		},
	}
	r := &HealthRunner{}
	tasks, err := r.Prepare(context.Background(), ws, types.Filter{All: true})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, []string{"curl", "localhost/health"}, tasks[0].Command)
	assert.Contains(t, tasks[0].ID, "/health")
	assert.Equal(t, "test", tasks[0].RepoName)
}

func TestHealthRunnerPrepareWithoutCommand(t *testing.T) {
	ws := &workspace.Workspace{
		Repos: []types.Repo{
			{Name: "test", Path: "/tmp", HealthCommand: ""},
		},
	}
	r := &HealthRunner{}
	tasks, err := r.Prepare(context.Background(), ws, types.Filter{All: true})
	require.NoError(t, err)
	require.Len(t, tasks, 0)
}

func TestHealthRunnerTimeout(t *testing.T) {
	ws := &workspace.Workspace{
		Repos: []types.Repo{
			{Name: "test", Path: "/tmp", HealthCommand: "curl localhost/health", HealthTimeout: 10 * time.Second},
		},
	}
	r := &HealthRunner{}
	tasks, err := r.Prepare(context.Background(), ws, types.Filter{All: true})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, 10*time.Second, tasks[0].Timeout)
	assert.Equal(t, []string{"curl", "localhost/health"}, tasks[0].Command)
	assert.Contains(t, tasks[0].ID, "/health")
}
