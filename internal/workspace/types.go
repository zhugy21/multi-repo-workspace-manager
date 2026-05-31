package workspace

import (
	"time"

	"github.com/user/ws/pkg/types"
)

// Config holds workspace-level configuration.
type Config struct {
	DefaultConcurrency int           `yaml:"default_concurrency"`
	DefaultTimeout     time.Duration `yaml:"default_timeout"`
	FailFast           bool          `yaml:"fail_fast"`
	OutputFormat       string        `yaml:"output_format"`
}

// Workspace represents a parsed workspace.yaml file.
type Workspace struct {
	WorkspaceName string              `yaml:"workspace_name"`
	Description   string              `yaml:"description"`
	Config        Config              `yaml:"config"`
	Groups        map[string][]string `yaml:"groups"`
	Repos         []types.Repo        `yaml:"repos"`
}

// ActiveConfig represents a file that contains a reference to an active workspace.
type ActiveConfig struct {
	Active string `yaml:"active"`
}
