package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestRootCommandHasAllSubcommands(t *testing.T) {
	root := NewRootCommand()

	expected := []string{
		"sync", "build", "health", "status", "monitor",
		"init", "add", "remove", "list", "switch", "config",
	}

	for _, name := range expected {
		sub := findSubcommand(root, name)
		if assert.NotNil(t, sub, "expected subcommand %q to exist", name) {
			assert.NotEmpty(t, sub.Use, "subcommand %q should have a Use field", name)
			assert.NotEmpty(t, sub.Short, "subcommand %q should have a Short field", name)
		}
	}

	// Ensure no extra subcommands exist.
	assert.Len(t, root.Commands(), len(expected), "unexpected number of subcommands")
}

func TestRootCommandHasGlobalFlags(t *testing.T) {
	root := NewRootCommand()

	expectedFlags := []string{
		"workspace", "all", "group", "repo",
		"concurrency", "timeout", "fail-fast", "continue-on-error",
		"output", "format",
	}

	for _, name := range expectedFlags {
		f := root.PersistentFlags().Lookup(name)
		assert.NotNil(t, f, "expected global flag --%s to exist", name)
	}
}

func TestSyncStubHasCheckoutFlag(t *testing.T) {
	syncCmd := newSyncCmd()

	checkoutFlag := syncCmd.Flags().Lookup("checkout")
	assert.NotNil(t, checkoutFlag, "sync command should have --checkout flag")
	assert.Equal(t, "string", checkoutFlag.Value.Type(), "--checkout flag should be a string")

	forceFlag := syncCmd.Flags().Lookup("force")
	assert.NotNil(t, forceFlag, "sync command should have --force flag")
	assert.Equal(t, "bool", forceFlag.Value.Type(), "--force flag should be a bool")
}

// findSubcommand searches for a subcommand by name.
func findSubcommand(cmd *cobra.Command, name string) *cobra.Command {
	for _, sub := range cmd.Commands() {
		if sub.Name() == name {
			return sub
		}
	}
	return nil
}
