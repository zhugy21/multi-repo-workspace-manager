package workspace

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/user/ws/pkg/types"
)

// ---------------------------------------------------------------------------
// Parse tests
// ---------------------------------------------------------------------------

func TestParseValidConfig(t *testing.T) {
	ws, err := Parse("testdata/valid.yaml")
	require.NoError(t, err)
	require.NotNil(t, ws)

	assert.Equal(t, "fixtures", ws.WorkspaceName)
	assert.Len(t, ws.Repos, 2)
	assert.Equal(t, "svc-a", ws.Repos[0].Name)
	assert.Equal(t, "svc-b", ws.Repos[1].Name)

	backend, ok := ws.Groups["backend"]
	require.True(t, ok)
	assert.Equal(t, []string{"svc-a", "svc-b"}, backend)
}

func TestParseMissingName(t *testing.T) {
	// Also create via t.TempDir + os.WriteFile as required by the spec.
	content := []byte("config: {}\nrepos:\n  - path: ./no-name\n")
	dir := t.TempDir()
	p := filepath.Join(dir, "missing_name.yaml")
	require.NoError(t, os.WriteFile(p, content, 0644))

	ws, err := Parse(p)
	assert.Error(t, err)
	assert.Nil(t, ws) // or could be non-nil with error

	var cfgErr *types.ConfigError
	assert.ErrorAs(t, err, &cfgErr)
}

func TestParseInvalidGroupRef(t *testing.T) {
	ws, err := Parse("testdata/invalid_group_ref.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "group")
	_ = ws // ws may be returned with error from validate
}

func TestParseNonexistentFile(t *testing.T) {
	ws, err := Parse("/nonexistent")
	assert.Error(t, err)
	assert.Nil(t, ws)

	var wsErr *types.WorkspaceError
	assert.ErrorAs(t, err, &wsErr)
}

func TestParseDefaults(t *testing.T) {
	content := []byte("config: {}\nrepos: []\n")
	dir := t.TempDir()
	p := filepath.Join(dir, "minimal.yaml")
	require.NoError(t, os.WriteFile(p, content, 0644))

	ws, err := Parse(p)
	require.NoError(t, err)
	require.NotNil(t, ws)

	assert.Equal(t, 4, ws.Config.DefaultConcurrency)
	assert.Equal(t, 120*time.Second, ws.Config.DefaultTimeout)
	assert.Equal(t, "tui", ws.Config.OutputFormat)
	assert.False(t, ws.Config.FailFast) // zero-value should remain false
}

// ---------------------------------------------------------------------------
// Filter tests
// ---------------------------------------------------------------------------

func TestFilterByRepo(t *testing.T) {
	ws := &Workspace{
		Repos: []types.Repo{
			{Name: "svc-a", Path: "./svc-a"},
			{Name: "svc-b", Path: "./svc-b"},
			{Name: "svc-c", Path: "./svc-c"},
		},
	}

	result := ws.Filter(types.Filter{Repo: "svc-a"})
	require.Len(t, result, 1)
	assert.Equal(t, "svc-a", result[0].Name)
}

func TestFilterByGroup(t *testing.T) {
	ws := &Workspace{
		Groups: map[string][]string{
			"backend": {"svc-a", "svc-b"},
		},
		Repos: []types.Repo{
			{Name: "svc-a", Path: "./svc-a"},
			{Name: "svc-b", Path: "./svc-b"},
			{Name: "svc-c", Path: "./svc-c"},
		},
	}

	result := ws.Filter(types.Filter{Group: "backend"})
	require.Len(t, result, 2)
	names := []string{result[0].Name, result[1].Name}
	assert.ElementsMatch(t, []string{"svc-a", "svc-b"}, names)
}

func TestFilterByAll(t *testing.T) {
	ws := &Workspace{
		Repos: []types.Repo{
			{Name: "svc-a", Path: "./svc-a"},
			{Name: "svc-b", Path: "./svc-b"},
		},
	}

	result := ws.Filter(types.Filter{All: true})
	require.Len(t, result, 2)
}

func TestFilterRepoOverGroup(t *testing.T) {
	ws := &Workspace{
		Groups: map[string][]string{
			"backend": {"svc-b", "svc-c"},
		},
		Repos: []types.Repo{
			{Name: "svc-a", Path: "./svc-a"},
			{Name: "svc-b", Path: "./svc-b"},
			{Name: "svc-c", Path: "./svc-c"},
		},
	}

	// Capture stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	oldStderr := os.Stderr
	os.Stderr = w

	result := ws.Filter(types.Filter{Repo: "svc-a", Group: "backend"})

	// Close writer and restore stderr
	require.NoError(t, w.Close())
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	require.NoError(t, r.Close())

	// Verify warning was printed
	assert.Contains(t, buf.String(), "warning")
	assert.Contains(t, buf.String(), "--repo")
	assert.Contains(t, buf.String(), "--group")

	// Verify only the repo-filtered result is returned
	require.Len(t, result, 1)
	assert.Equal(t, "svc-a", result[0].Name)
}
