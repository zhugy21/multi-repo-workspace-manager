package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// TestBuildCommandNoExtraFlags verifies the build command has no
// command-specific flags (only persistent flags inherited from root).
func TestBuildCommandNoExtraFlags(t *testing.T) {
	cmd := newBuildCmd()

	count := 0
	cmd.Flags().VisitAll(func(_ *pflag.Flag) {
		count++
	})
	if count != 0 {
		t.Errorf("expected build command to have 0 command-specific flags, got %d", count)
	}
}

// TestBuildNoBuildCommand verifies that running build on a workspace with
// repos that have no build_command produces no error (task list is empty).
func TestBuildNoBuildCommand(t *testing.T) {
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
	err = runBuild(cmd, nil)
	if err != nil {
		t.Fatalf("expected no error when running build without build_command, got: %v", err)
	}
}
