package types

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusConstants(t *testing.T) {
	assert.Equal(t, Status("pending"), StatusPending)
	assert.Equal(t, Status("running"), StatusRunning)
	assert.Equal(t, Status("success"), StatusSuccess)
	assert.Equal(t, Status("failed"), StatusFailed)
	assert.Equal(t, Status("cancelled"), StatusCancelled)
	assert.Equal(t, Status("skipped"), StatusSkipped)
	assert.Equal(t, Status("warning"), StatusWarning)
}

func TestRepoJSONRoundTrip(t *testing.T) {
	r := Repo{
		Name:          "auth-service",
		Path:          "./auth-service",
		URL:           "git@github.com:org/auth-service.git",
		Type:          "go",
		Group:         "backend",
		DefaultBranch: "main",
		BuildCommand:  "make build",
		BuildTimeout:  5 * time.Minute,
		HealthCommand: "curl -sf http://localhost:8080/health",
		HealthTimeout: 30 * time.Second,
		EnvFile:       ".env.auth",
	}
	data, err := json.Marshal(r)
	require.NoError(t, err)
	var r2 Repo
	err = json.Unmarshal(data, &r2)
	require.NoError(t, err)
	assert.Equal(t, r.Name, r2.Name)
	assert.Equal(t, r.Path, r2.Path)
	assert.Equal(t, r.URL, r2.URL)
	assert.Equal(t, "go", r2.Type)
	assert.Equal(t, "backend", r2.Group)
	assert.Equal(t, "main", r2.DefaultBranch)
	assert.Equal(t, "make build", r2.BuildCommand)
	assert.Equal(t, r.BuildTimeout, r2.BuildTimeout)
	assert.Equal(t, r.HealthTimeout, r2.HealthTimeout)
}

func TestResultJSONOmitEmpty(t *testing.T) {
	r := Result{
		TaskID:   "a/sync",
		RepoName: "a",
		Status:   StatusSuccess,
		Detail:   "ok",
	}
	data, err := json.Marshal(r)
	require.NoError(t, err)
	assert.NotContains(t, string(data), `"group"`)
	assert.NotContains(t, string(data), `"error"`)
}

func TestResultSummary(t *testing.T) {
	results := []Result{
		{RepoName: "a", Status: StatusSuccess},
		{RepoName: "b", Status: StatusFailed},
		{RepoName: "c", Status: StatusCancelled},
		{RepoName: "d", Status: StatusSkipped},
		{RepoName: "e", Status: StatusSuccess},
	}
	s := Summarize(results)
	assert.Equal(t, 5, s.Total)
	assert.Equal(t, 2, s.Success)
	assert.Equal(t, 1, s.Failed)
	assert.Equal(t, 1, s.Cancelled)
	assert.Equal(t, 1, s.Skipped)
	assert.Equal(t, 0, s.Warning)
	assert.True(t, s.HasFailures())
}

func TestSummarizeNoFailures(t *testing.T) {
	results := []Result{
		{RepoName: "a", Status: StatusSuccess},
		{RepoName: "b", Status: StatusWarning},
	}
	s := Summarize(results)
	assert.Equal(t, 2, s.Total)
	assert.Equal(t, 1, s.Success)
	assert.Equal(t, 1, s.Warning)
	assert.False(t, s.HasFailures())
}

func TestSummarizeNilInput(t *testing.T) {
	s := Summarize(nil)
	assert.Equal(t, 0, s.Total)
	assert.Equal(t, Summary{}, s)
}

func TestFilterDefaults(t *testing.T) {
	f := Filter{}
	assert.False(t, f.All)
	assert.Empty(t, f.Group)
	assert.Empty(t, f.Repo)
}

func TestConfigError(t *testing.T) {
	err := &ConfigError{File: "workspace.yaml", Message: "missing name"}
	assert.Contains(t, err.Error(), "workspace.yaml")
	assert.Contains(t, err.Error(), "missing name")
}

func TestRepoError(t *testing.T) {
	err := &RepoError{Repo: "auth", Message: "git pull failed", Err: fmt.Errorf("network")}
	assert.Contains(t, err.Error(), "auth")
	assert.Contains(t, err.Error(), "git pull failed")
	assert.Contains(t, err.Error(), "network")
}

func TestCommandError(t *testing.T) {
	err := &CommandError{Repo: "auth", Command: "make build", ExitCode: 1, Stderr: "compile error"}
	assert.Contains(t, err.Error(), "auth")
	assert.Contains(t, err.Error(), "make build")
	assert.Contains(t, err.Error(), "exit code 1")
	assert.Contains(t, err.Error(), "compile error")
}

func TestTimeoutError(t *testing.T) {
	err := &TimeoutError{Repo: "auth", Timeout: "30s"}
	assert.Contains(t, err.Error(), "auth")
	assert.Contains(t, err.Error(), "timed out after 30s")
}

func TestCancelError(t *testing.T) {
	err := &CancelError{Repo: "auth", Reason: "fail-fast triggered by payment-svc"}
	assert.Contains(t, err.Error(), "auth")
	assert.Contains(t, err.Error(), "cancelled")
	assert.Contains(t, err.Error(), "payment-svc")
}

func TestWorkspaceError(t *testing.T) {
	err := &WorkspaceError{Name: "myproject", Message: "not found"}
	assert.Contains(t, err.Error(), "myproject")
	assert.Contains(t, err.Error(), "not found")
}

func TestErrorUnwrap(t *testing.T) {
	inner := fmt.Errorf("inner")
	cfgErr := &ConfigError{File: "f", Message: "m", Err: inner}
	assert.ErrorIs(t, cfgErr, inner)

	wsErr := &WorkspaceError{Name: "w", Message: "m", Err: inner}
	assert.ErrorIs(t, wsErr, inner)
}
