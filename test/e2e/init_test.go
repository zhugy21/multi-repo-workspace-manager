package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitCreatesWorkspaceYAML(t *testing.T) {
	dir := t.TempDir()

	cmd := wsCmd(t, "init", "--name", "e2e-test")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "init should succeed:\n%s", string(out))

	yamlPath := filepath.Join(dir, "workspace.yaml")
	_, err = os.Stat(yamlPath)
	require.NoError(t, err, "workspace.yaml should exist")

	data, err := os.ReadFile(yamlPath)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "workspace_name: e2e-test")
	assert.Contains(t, content, "default_concurrency: 4")
}

func TestInitOverwritePromptNo(t *testing.T) {
	dir := t.TempDir()

	// Create an existing workspace.yaml.
	err := os.WriteFile(filepath.Join(dir, "workspace.yaml"), []byte("workspace_name: old\n"), 0644)
	require.NoError(t, err)

	cmd := wsCmd(t, "init", "--name", "e2e-test")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader("n\n")
	out, err := cmd.CombinedOutput()
	t.Logf("output:\n%s", string(out))

	// Should exit with error.
	assert.Error(t, err, "expected non-zero exit when refusing overwrite")
}

func TestSyncNoWorkspace(t *testing.T) {
	dir := t.TempDir()

	cmd := wsCmd(t, "sync")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	t.Logf("output:\n%s", string(out))

	// Should exit with code 2 (WorkspaceError).
	require.Error(t, err)
	if exitErr, ok := err.(*exec.ExitError); ok {
		assert.Equal(t, 2, exitErr.ExitCode(),
			"expected exit code 2 for missing workspace.yaml")
	} else {
		t.Fatalf("expected exec.ExitError, got %T: %v", err, err)
	}
}
