package workspace

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/user/ws/pkg/types"
)

// applyDefaults sets sensible defaults for fields that are zero-valued.
func (ws *Workspace) applyDefaults() {
	if ws.Config.DefaultConcurrency == 0 {
		ws.Config.DefaultConcurrency = 4
	}
	if ws.Config.DefaultTimeout == 0 {
		ws.Config.DefaultTimeout = 120 * time.Second
	}
	if ws.Config.OutputFormat == "" {
		ws.Config.OutputFormat = "tui"
	}
}

// validate checks workspace invariants and returns a ConfigError on failure.
func (ws *Workspace) validate() error {
	seen := make(map[string]bool, len(ws.Repos))
	for i := range ws.Repos {
		r := &ws.Repos[i]
		if r.Name == "" {
			return &types.ConfigError{
				File:    "",
				Message: fmt.Sprintf("repo at index %d is missing required field 'name'", i),
			}
		}
		if r.Path == "" {
			return &types.ConfigError{
				File:    "",
				Message: fmt.Sprintf("repo %q is missing required field 'path'", r.Name),
			}
		}
		if seen[r.Name] {
			return &types.ConfigError{
				File:    "",
				Message: fmt.Sprintf("duplicate repo name %q", r.Name),
			}
		}
		seen[r.Name] = true

		if r.Group != "" {
			members, ok := ws.Groups[r.Group]
			if !ok {
				return &types.ConfigError{
					File:    "",
					Message: fmt.Sprintf("repo %q references non-existent group %q", r.Name, r.Group),
				}
			}
			found := false
			for _, m := range members {
				if m == r.Name {
					found = true
					break
				}
			}
			if !found {
				return &types.ConfigError{
					File:    "",
					Message: fmt.Sprintf("repo %q is not listed in group %q members", r.Name, r.Group),
				}
			}
		}
	}
	return nil
}

// Parse reads a workspace YAML file, validates it, and applies defaults.
func Parse(path string) (*Workspace, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &types.WorkspaceError{
			Name:    path,
			Message: "file not found",
			Err:     err,
		}
	}

	var ws Workspace
	if err := yaml.Unmarshal(data, &ws); err != nil {
		return nil, &types.ConfigError{
			File:    path,
			Message: "failed to parse YAML",
			Err:     err,
		}
	}

	if err := ws.validate(); err != nil {
		return nil, err
	}

	ws.applyDefaults()
	return &ws, nil
}

// ParseActiveConfig reads a file containing an "active" field pointing to a workspace path.
func ParseActiveConfig(path string) (*ActiveConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &types.WorkspaceError{
			Name:    path,
			Message: "file not found",
			Err:     err,
		}
	}

	var ac ActiveConfig
	if err := yaml.Unmarshal(data, &ac); err != nil {
		return nil, &types.ConfigError{
			File:    path,
			Message: "failed to parse active config YAML",
			Err:     err,
		}
	}
	return &ac, nil
}
