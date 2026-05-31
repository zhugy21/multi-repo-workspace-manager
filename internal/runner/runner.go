package runner

import (
	"context"

	"github.com/user/ws/internal/workspace"
	"github.com/user/ws/pkg/types"
)

// Runner defines the interface for all task runners.
type Runner interface {
	// Name returns the runner's identifier (e.g. "sync", "build", "health").
	Name() string

	// Prepare generates the list of tasks for this runner based on the workspace
	// and filter. The tasks are not executed here; they are returned for the
	// executor to run.
	Prepare(ctx context.Context, ws *workspace.Workspace, filter types.Filter) ([]types.Task, error)
}
