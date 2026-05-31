package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthWithCommand(t *testing.T) {
	dir := t.TempDir()

	repoDir := filepath.Join(dir, "myrepo")
	require.NoError(t, os.MkdirAll(repoDir, 0755))
	makeGitRepo(t, repoDir)

	wsContent := fmt.Sprintf(`workspace_name: test
repos:
  - name: myrepo
    path: %s
    health_command: "true"
`, repoDir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "workspace.yaml"), []byte(wsContent), 0644))

	cmd := wsCmd(t, "health", "--output", "plain")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	t.Logf("output:\n%s", string(out))

	require.NoError(t, err, "health should exit 0:\n%s", string(out))
	output := string(out)
	assert.Condition(t, func() bool {
		return strings.Contains(output, "OK") || strings.Contains(output, "passed")
	}, "output should contain 'OK' or 'passed'")
}

func TestHealthJSONOutput(t *testing.T) {
	dir := t.TempDir()

	repoDir := filepath.Join(dir, "myrepo")
	require.NoError(t, os.MkdirAll(repoDir, 0755))
	makeGitRepo(t, repoDir)

	wsContent := fmt.Sprintf(`workspace_name: test
repos:
  - name: myrepo
    path: %s
    health_command: "true"
`, repoDir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "workspace.yaml"), []byte(wsContent), 0644))

	cmd := wsCmd(t, "health", "--output", "json")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	t.Logf("output:\n%s", string(out))

	require.NoError(t, err, "health with json output should exit 0:\n%s", string(out))

	// Verify valid JSON with a "results" array.
	var result map[string]interface{}
	err = json.Unmarshal(out, &result)
	require.NoError(t, err, "output should be valid JSON")

	results, ok := result["results"]
	assert.True(t, ok, "JSON output should contain 'results' key")
	if ok {
		resultsArr, ok := results.([]interface{})
		assert.True(t, ok, "'results' should be an array")
		assert.NotEmpty(t, resultsArr, "'results' array should not be empty")
	}

	// Check summary is also present.
	_, hasSummary := result["summary"]
	assert.True(t, hasSummary, "JSON output should contain 'summary' key")
}
