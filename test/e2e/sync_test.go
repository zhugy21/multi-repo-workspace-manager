package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncAllSuccess(t *testing.T) {
	dir := t.TempDir()

	// Create two git repos with local bare remotes.
	repo1 := makeGitRepoWithRemote(t, dir, "repo1")
	repo2 := makeGitRepoWithRemote(t, dir, "repo2")

	wsContent := fmt.Sprintf(`workspace_name: test
config:
  default_concurrency: 2
repos:
  - name: repo1
    path: %s
  - name: repo2
    path: %s
`, repo1, repo2)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "workspace.yaml"), []byte(wsContent), 0644))

	cmd := wsCmd(t, "sync", "--output", "plain")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	t.Logf("output:\n%s", string(out))

	require.NoError(t, err, "sync should exit 0:\n%s", string(out))
	assert.Contains(t, string(out), "passed")
}

func TestSyncFailFast(t *testing.T) {
	dir := t.TempDir()

	// Repo1: a clean repo with remote — should sync successfully.
	repo1 := makeGitRepoWithRemote(t, dir, "repo1")
	// Repo2: a repo with no remote at all — git fetch origin will fail deterministically.
	repo2 := filepath.Join(dir, "repo2")
	require.NoError(t, os.MkdirAll(repo2, 0755))
	makeGitRepo(t, repo2)

	wsContent := fmt.Sprintf(`workspace_name: test
repos:
  - name: repo1
    path: %s
  - name: repo2
    path: %s
`, repo1, repo2)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "workspace.yaml"), []byte(wsContent), 0644))

	cmd := wsCmd(t, "sync", "--fail-fast", "--output", "plain")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	t.Logf("output:\n%s", string(out))

	// Should exit non-zero because repo2 fails (no remote) and fail-fast cancels other tasks.
	assert.Error(t, err, "expected non-zero exit for fail-fast")
	output := string(out)
	assert.Condition(t, func() bool {
		return assert.Contains(t, output, "cancelled") || assert.Contains(t, output, "skipped")
	}, "output should contain 'cancelled' or 'skipped'")
}

func TestSyncCheckoutDirtySkip(t *testing.T) {
	dir := t.TempDir()

	repoDir := filepath.Join(dir, "myrepo")
	require.NoError(t, os.MkdirAll(repoDir, 0755))
	makeGitRepo(t, repoDir)

	// Create a feature branch in the repo.
	cmd := exec.Command("git", "-C", repoDir, "checkout", "-b", "feature-x")
	require.NoError(t, cmd.Run())
	cmd = exec.Command("git", "-C", repoDir, "checkout", "-") // back to previous branch
	require.NoError(t, cmd.Run())

	// Make the repo dirty.
	err := os.WriteFile(filepath.Join(repoDir, "dirty.txt"), []byte("uncommitted"), 0644)
	require.NoError(t, err)

	wsContent := fmt.Sprintf(`workspace_name: test
repos:
  - name: myrepo
    path: %s
`, repoDir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "workspace.yaml"), []byte(wsContent), 0644))

	cmd = wsCmd(t, "sync", "--checkout", "feature-x", "--output", "plain")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	t.Logf("output:\n%s", string(out))

	require.NoError(t, err, "dirty-skip should exit 0")
	output := string(out)
	assert.Condition(t, func() bool {
		hasWarn := assert.Contains(t, output, "WARN")
		hasDirty := assert.Contains(t, output, "dirty")
		return hasWarn || hasDirty
	}, "output should contain 'WARN' or 'dirty'")
}

func TestSyncCheckoutForce(t *testing.T) {
	dir := t.TempDir()

	// Create a repo with a remote so feature-x can have an upstream.
	repoDir := makeGitRepoWithRemote(t, dir, "myrepo")

	// Get the current branch name so we can return to it.
	out, err := exec.Command("git", "-C", repoDir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	require.NoError(t, err)
	mainBranch := strings.TrimSpace(string(out))

	// Create feature-x, push it (sets upstream), then switch back to main.
	cmd := exec.Command("git", "-C", repoDir, "checkout", "-b", "feature-x")
	require.NoError(t, cmd.Run())
	cmd = exec.Command("git", "-C", repoDir, "push", "--set-upstream", "origin", "feature-x")
	require.NoError(t, cmd.Run())
	cmd = exec.Command("git", "-C", repoDir, "checkout", mainBranch)
	require.NoError(t, cmd.Run())

	// Make the repo dirty (add an untracked file).
	err = os.WriteFile(filepath.Join(repoDir, "dirty.txt"), []byte("uncommitted"), 0644)
	require.NoError(t, err)

	wsContent := fmt.Sprintf(`workspace_name: test
repos:
  - name: myrepo
    path: %s
`, repoDir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "workspace.yaml"), []byte(wsContent), 0644))

	cmd = wsCmd(t, "sync", "--checkout", "feature-x", "--force", "--concurrency", "1", "--output", "plain")
	cmd.Dir = dir
	out, err = cmd.CombinedOutput()
	t.Logf("output:\n%s", string(out))

	require.NoError(t, err, "force checkout should exit 0:\n%s", string(out))

	// Verify we are now on the feature-x branch.
	branchOut, err := exec.Command("git", "-C", repoDir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	require.NoError(t, err)
	assert.Equal(t, "feature-x", strings.TrimSpace(string(branchOut)),
		"should be on feature-x branch after force checkout")
}
