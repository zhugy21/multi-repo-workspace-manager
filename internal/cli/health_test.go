package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

// TestHealthCommandJSONOutput verifies that --output json flag works with the health command.
func TestHealthCommandJSONOutput(t *testing.T) {
	dir := t.TempDir()
	wsPath := filepath.Join(dir, "workspace.yaml")
	wsContent := []byte(`workspace_name: "test"
repos:
  - name: "test-repo"
    path: "/tmp"
    health_command: "echo ok"
`)
	if err := os.WriteFile(wsPath, wsContent, 0644); err != nil {
		t.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(cwd)
	}()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Set output flag to json.
	outputFlag = "json"
	defer func() { outputFlag = "" }()

	cmd := &cobra.Command{}
	err = runHealth(cmd, nil)
	if err != nil {
		t.Fatalf("expected no error when running health with --output json, got: %v", err)
	}
}

// TestHealthNoHealthCommand verifies that running health on a workspace with
// repos that have no health_command produces no error (task list is empty).
func TestHealthNoHealthCommand(t *testing.T) {
	dir := t.TempDir()
	wsPath := filepath.Join(dir, "workspace.yaml")
	wsContent := []byte(`workspace_name: "test"
repos:
  - name: "test-repo"
    path: "/tmp"
`)
	if err := os.WriteFile(wsPath, wsContent, 0644); err != nil {
		t.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(cwd)
	}()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	err = runHealth(cmd, nil)
	if err != nil {
		t.Fatalf("expected no error when running health without health_command, got: %v", err)
	}
}
