package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// TestSyncCommandHasCheckoutAndForceFlags verifies the sync command registers
// the --checkout and --force flags as described in the CLI spec.
func TestSyncCommandHasCheckoutAndForceFlags(t *testing.T) {
	cmd := newSyncCmd()

	// Check --checkout flag exists and has the correct default.
	checkoutFlag := cmd.Flags().Lookup("checkout")
	if checkoutFlag == nil {
		t.Fatal("expected --checkout flag to be registered on sync command")
	}
	if checkoutFlag.DefValue != "" {
		t.Errorf("expected --checkout default to be empty string, got %q", checkoutFlag.DefValue)
	}

	// Check --force flag exists and has the correct default.
	forceFlag := cmd.Flags().Lookup("force")
	if forceFlag == nil {
		t.Fatal("expected --force flag to be registered on sync command")
	}
	if forceFlag.DefValue != "false" {
		t.Errorf("expected --force default to be false, got %q", forceFlag.DefValue)
	}

	// Check there are no other flags registered on sync (only checkout and force).
	expected := 2
	count := 0
	cmd.Flags().VisitAll(func(_ *pflag.Flag) {
		count++
	})
	if count != expected {
		t.Errorf("expected sync command to have %d flags, got %d", expected, count)
	}
}

// TestSyncNoWorkspace verifies that running sync in a directory without a
// workspace.yaml returns an error.
func TestSyncNoWorkspace(t *testing.T) {
	// Create a temporary directory with no workspace.yaml.
	dir := t.TempDir()
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

	// Ensure the directory doesn't have a workspace.yaml.
	if _, err := os.Stat(filepath.Join(dir, "workspace.yaml")); err == nil {
		t.Fatal("expected no workspace.yaml in temp dir")
	}

	// Run sync — it should error because no workspace exists.
	cmd := &cobra.Command{}
	err = runSync(cmd, nil)
	if err == nil {
		t.Fatal("expected error when running sync without workspace.yaml, got nil")
	}
}
