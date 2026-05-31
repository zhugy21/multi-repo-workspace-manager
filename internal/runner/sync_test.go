package runner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/user/ws/internal/workspace"
	"github.com/user/ws/pkg/types"
)

// gitInit initialises a git repository at dir with an initial empty commit.
func gitInit(t *testing.T, dir string) {
	t.Helper()

	cmd := exec.Command("git", "-C", dir, "init")
	require.NoError(t, cmd.Run(), "git init should succeed")

	cmd = exec.Command("git", "-C", dir, "config", "user.email", "test@test.com")
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "-C", dir, "config", "user.name", "test")
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "-C", dir, "commit", "--allow-empty", "-m", "initial")
	require.NoError(t, cmd.Run(), "initial commit should succeed")
}

func TestSyncRunnerPrepare(t *testing.T) {
	// Set up a real git repo in a temp directory.
	dir := t.TempDir()
	gitInit(t, dir)

	ws := &workspace.Workspace{
		Repos: []types.Repo{
			{Name: "myrepo", Path: dir, Group: "default"},
		},
	}

	r := &SyncRunner{}
	tasks, err := r.Prepare(context.Background(), ws, types.Filter{})
	require.NoError(t, err)
	require.Len(t, tasks, 2, "expected fetch + merge tasks")

	assert.Contains(t, tasks[0].ID, "/fetch", "first task should be a fetch")
	assert.Contains(t, tasks[1].ID, "/merge", "second task should be a merge")

	assert.Equal(t, "myrepo", tasks[0].RepoName)
	assert.Equal(t, "myrepo", tasks[1].RepoName)
	assert.Equal(t, "default", tasks[0].Group)
}

func TestSyncRunnerPrepareCheckout(t *testing.T) {
	dir := t.TempDir()
	gitInit(t, dir)

	ws := &workspace.Workspace{
		Repos: []types.Repo{
			{Name: "myrepo", Path: dir, Group: "default"},
		},
	}

	r := &SyncRunner{Checkout: "feature-x"}
	tasks, err := r.Prepare(context.Background(), ws, types.Filter{})
	require.NoError(t, err)
	require.Len(t, tasks, 2, "expected checkout + pull tasks")

	assert.Contains(t, tasks[0].Command, "checkout", "first task should be checkout")
	assert.Contains(t, tasks[0].Command, "feature-x", "checkout should target feature-x branch")
	assert.Contains(t, tasks[1].Command, "pull", "second task should be pull")
}

func TestSyncRunnerPrepareDirtySkip(t *testing.T) {
	dir := t.TempDir()
	gitInit(t, dir)

	// Create an uncommitted file to make the repo dirty.
	dirtyFile := filepath.Join(dir, "dirty.txt")
	err := os.WriteFile(dirtyFile, []byte("uncommitted content"), 0644)
	require.NoError(t, err)

	ws := &workspace.Workspace{
		Repos: []types.Repo{
			{Name: "myrepo", Path: dir, Group: "default"},
		},
	}

	r := &SyncRunner{Checkout: "feature-x", Force: false}
	tasks, err := r.Prepare(context.Background(), ws, types.Filter{})
	require.NoError(t, err)

	// With dirty && !Force, we expect a warning task — no checkout task.
	require.Equal(t, 1, len(tasks), "expected exactly 1 warning task for dirty repo")
	assert.Contains(t, tasks[0].ID, "/warning", "task should be a warning")
	assert.Equal(t, "echo", tasks[0].Command[0], "warning task should be an echo command")
	assert.Contains(t, tasks[0].Command[1], "WARNING", "warning message should contain WARNING")
}

func TestSyncRunnerPrepareClone(t *testing.T) {
	ws := &workspace.Workspace{
		Repos: []types.Repo{
			{
				Name:  "x",
				Path:  "/tmp/nonexistent-12345",
				URL:   "git@github.com:org/x.git",
				Group: "default",
			},
		},
	}

	r := &SyncRunner{}
	tasks, err := r.Prepare(context.Background(), ws, types.Filter{})
	require.NoError(t, err)
	require.Len(t, tasks, 1, "expected 1 clone task")

	assert.Equal(t, "x/clone", tasks[0].ID)
	assert.Equal(t, []string{"git", "clone", "git@github.com:org/x.git", "/tmp/nonexistent-12345"}, tasks[0].Command)
}

func TestSyncRunnerPrepareEmptyRepo(t *testing.T) {
	ws := &workspace.Workspace{
		Repos: []types.Repo{
			{
				Name:  "ghost",
				Path:  "/tmp/nonexistent-99999",
				URL:   "", // no URL
				Group: "default",
			},
		},
	}

	r := &SyncRunner{}
	tasks, err := r.Prepare(context.Background(), ws, types.Filter{})
	require.NoError(t, err)
	require.Len(t, tasks, 0, "expected 0 tasks for repo with no URL and non-existent path")
}

// Test helper — verify isDirty and pathExists work correctly.
func TestIsDirty(t *testing.T) {
	dir := t.TempDir()
	gitInit(t, dir)

	clean, err := isDirty(dir)
	require.NoError(t, err)
	assert.False(t, clean, "clean repo should not be dirty")

	dirtyFile := filepath.Join(dir, "new.txt")
	require.NoError(t, os.WriteFile(dirtyFile, []byte("data"), 0644))

	dirty, err := isDirty(dir)
	require.NoError(t, err)
	assert.True(t, dirty, "repo with uncommitted file should be dirty")
}

func TestPathExists(t *testing.T) {
	assert.True(t, pathExists("."), "current dir should exist")
	assert.False(t, pathExists("/tmp/nonexistent-path-abc-123"), "bogus path should not exist")
}
