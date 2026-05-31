package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/user/ws/internal/cli"
	"github.com/user/ws/pkg/types"
)

func main() {
	cmd := cli.NewRootCommand()
	if err := cmd.Execute(); err != nil {
		var cfgErr *types.ConfigError
		var wsErr *types.WorkspaceError
		if errors.As(err, &cfgErr) || errors.As(err, &wsErr) {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
