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

func TestBuildRunnerPrepareWithCommand(t *testing.T) {
	ws := &workspace.Workspace{
		Repos: []types.Repo{
			{Name: "test", Path: "/tmp", BuildCommand: "echo hello world"},
		},
	}
	r := &BuildRunner{}
	tasks, err := r.Prepare(context.Background(), ws, types.Filter{All: true})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, []string{"echo", "hello", "world"}, tasks[0].Command)
	assert.Contains(t, tasks[0].ID, "/build")
	assert.Equal(t, "test", tasks[0].RepoName)
}

func TestBuildRunnerPrepareWithoutCommand(t *testing.T) {
	ws := &workspace.Workspace{
		Repos: []types.Repo{
			{Name: "test", Path: "/tmp", BuildCommand: ""},
		},
	}
	r := &BuildRunner{}
	tasks, err := r.Prepare(context.Background(), ws, types.Filter{All: true})
	require.NoError(t, err)
	require.Len(t, tasks, 0)
}

func TestBuildRunnerTimeout(t *testing.T) {
	ws := &workspace.Workspace{
		Repos: []types.Repo{
			{Name: "test", Path: "/tmp", BuildCommand: "make", BuildTimeout: 5 * time.Second},
		},
	}
	r := &BuildRunner{}
	tasks, err := r.Prepare(context.Background(), ws, types.Filter{All: true})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, 5*time.Second, tasks[0].Timeout)
	assert.Equal(t, []string{"make"}, tasks[0].Command)
	assert.Contains(t, tasks[0].ID, "/build")
}
