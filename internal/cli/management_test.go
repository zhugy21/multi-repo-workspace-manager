package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Init tests
// ---------------------------------------------------------------------------

func TestInitWithName(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(cwd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cmd := newInitCmd()
	cmd.SetArgs([]string{"--name", "test"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	data, err := os.ReadFile("workspace.yaml")
	if err != nil {
		t.Fatalf("expected workspace.yaml to exist, got: %v", err)
	}
	if !strings.Contains(string(data), "workspace_name: test") {
		t.Fatalf("expected workspace_name: test in file, got: %s", string(data))
	}
}

func TestInitOverwriteNo(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(cwd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Create an existing workspace.yaml.
	if err := os.WriteFile("workspace.yaml", []byte("workspace_name: old\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Mock stdin to respond "n" to the overwrite prompt.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("n\n")); err != nil {
		t.Fatal(err)
	}
	w.Close()
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	cmd := newInitCmd()
	cmd.SetArgs([]string{"--name", "test"})
	err = cmd.Execute()
	if err == nil {
		t.Fatal("expected error when refusing overwrite, got nil")
	}
}

// ---------------------------------------------------------------------------
// Add / Remove tests
// ---------------------------------------------------------------------------

func TestAddRepo(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(cwd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// First init a workspace.
	initCmd := newInitCmd()
	initCmd.SetArgs([]string{"--name", "test"})
	if err := initCmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Add a repo.
	addCmd := newAddCmd()
	addCmd.SetArgs([]string{"--name", "my-repo", "--path", "/some/path"})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	// Verify the repo was added.
	data, err := os.ReadFile("workspace.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "my-repo") {
		t.Fatalf("expected repo name 'my-repo' in workspace.yaml, got: %s", string(data))
	}
}

func TestRemoveRepo(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(cwd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Init workspace.
	initCmd := newInitCmd()
	initCmd.SetArgs([]string{"--name", "test"})
	if err := initCmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Add a repo.
	addCmd := newAddCmd()
	addCmd.SetArgs([]string{"--name", "my-repo", "--path", "/some/path"})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	// Remove the repo.
	removeCmd := newRemoveCmd()
	removeCmd.SetArgs([]string{"my-repo"})
	if err := removeCmd.Execute(); err != nil {
		t.Fatalf("remove failed: %v", err)
	}

	// Verify the repo is gone.
	data, err := os.ReadFile("workspace.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "my-repo") {
		t.Fatalf("expected repo to be removed, but found in: %s", string(data))
	}
}

func TestRemoveRepoNotFound(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(cwd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Init workspace.
	initCmd := newInitCmd()
	initCmd.SetArgs([]string{"--name", "test"})
	if err := initCmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Try to remove a nonexistent repo.
	removeCmd := newRemoveCmd()
	removeCmd.SetArgs([]string{"nonexistent"})
	err = removeCmd.Execute()
	if err == nil {
		t.Fatal("expected error when removing nonexistent repo, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected error message about 'not found', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Switch test
// ---------------------------------------------------------------------------

func TestSwitchWorkspace(t *testing.T) {
	homeDir := t.TempDir()
	oldEnv := os.Getenv("HOME")
	os.Setenv("HOME", homeDir)
	defer func() { _ = os.Setenv("HOME", oldEnv) }()

	// Create a workspace.yaml in a known location.
	realWorkspaceDir := t.TempDir()
	realWorkspacePath := filepath.Join(realWorkspaceDir, "workspace.yaml")
	wsContent := []byte("workspace_name: my-workspace\ndescription: \"\"\nconfig: {}\ngroups: {}\nrepos: []\n")
	if err := os.WriteFile(realWorkspacePath, wsContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Create the symlink at ~/.ws/workspaces/my-workspace.yaml -> realWorkspacePath.
	workspacesDir := filepath.Join(homeDir, ".ws", "workspaces")
	if err := os.MkdirAll(workspacesDir, 0755); err != nil {
		t.Fatal(err)
	}
	symlinkPath := filepath.Join(workspacesDir, "my-workspace.yaml")
	if err := os.Symlink(realWorkspacePath, symlinkPath); err != nil {
		t.Fatal(err)
	}

	// Run switch.
	switchCmd := newSwitchCmd()
	switchCmd.SetArgs([]string{"my-workspace"})
	if err := switchCmd.Execute(); err != nil {
		t.Fatalf("switch failed: %v", err)
	}

	// Check config.yaml contains the resolved path.
	configPath := filepath.Join(homeDir, ".ws", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected config.yaml to exist: %v", err)
	}
	if !strings.Contains(string(data), realWorkspacePath) {
		t.Fatalf("expected config.yaml to contain resolved path %s, got: %s", realWorkspacePath, string(data))
	}
}

// ---------------------------------------------------------------------------
// Config test
// ---------------------------------------------------------------------------

func TestConfigShow(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(cwd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Init a workspace so we have something to show.
	initCmd := newInitCmd()
	initCmd.SetArgs([]string{"--name", "test"})
	if err := initCmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Capture stdout.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	configCmd := newConfigCmd()
	configCmd.SetArgs([]string{"--show"})
	err = configCmd.Execute()

	_ = w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	output := buf.String()

	if err != nil {
		t.Fatalf("config --show failed: %v", err)
	}
	if !strings.Contains(output, "workspace_name: test") {
		t.Fatalf("expected output to contain workspace info, got: %s", output)
	}
}
