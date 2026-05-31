package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildWithCommand(t *testing.T) {
	dir := t.TempDir()

	// Create a git repo (required so workspace.yaml validates).
	repoDir := filepath.Join(dir, "myrepo")
	require.NoError(t, os.MkdirAll(repoDir, 0755))
	makeGitRepo(t, repoDir)

	wsContent := fmt.Sprintf(`workspace_name: test
repos:
  - name: myrepo
    path: %s
    build_command: "echo built"
`, repoDir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "workspace.yaml"), []byte(wsContent), 0644))

	cmd := wsCmd(t, "build", "--output", "plain")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	t.Logf("output:\n%s", string(out))

	require.NoError(t, err, "build should exit 0:\n%s", string(out))
	output := string(out)
	assert.Condition(t, func() bool {
		return strings.Contains(output, "OK") || strings.Contains(output, "passed")
	}, "output should contain 'OK' or 'passed'")
}

func TestBuildNoCommand(t *testing.T) {
	dir := t.TempDir()

	repoDir := filepath.Join(dir, "myrepo")
	require.NoError(t, os.MkdirAll(repoDir, 0755))
	makeGitRepo(t, repoDir)

	// No build_command set.
	wsContent := fmt.Sprintf(`workspace_name: test
repos:
  - name: myrepo
    path: %s
`, repoDir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "workspace.yaml"), []byte(wsContent), 0644))

	cmd := wsCmd(t, "build", "--output", "plain")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	t.Logf("output:\n%s", string(out))

	require.NoError(t, err, "build with no command should exit 0:\n%s", string(out))
}
