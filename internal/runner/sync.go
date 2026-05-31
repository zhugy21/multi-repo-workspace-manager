package runner

import (
	"context"
	"os"
	"os/exec"

	"github.com/user/ws/internal/workspace"
	"github.com/user/ws/pkg/types"
)

// SyncRunner generates tasks for syncing (fetching/merging or cloning) repositories.
// When Checkout is set, it additionally checks out a branch (with optional stashing).
type SyncRunner struct {
	Checkout string // branch name to checkout (empty = sync only)
	Force    bool   // force checkout even if dirty
}

// Name returns "sync".
func (r *SyncRunner) Name() string { return "sync" }

// Prepare generates sync tasks for each repo in the filtered workspace.
func (r *SyncRunner) Prepare(ctx context.Context, ws *workspace.Workspace, filter types.Filter) ([]types.Task, error) {
	repos := ws.Filter(filter)
	var tasks []types.Task

	for _, repo := range repos {
		exists := pathExists(repo.Path)

		if r.Checkout == "" {
			// --- No checkout: sync only ---
			if !exists {
				if repo.URL != "" {
					tasks = append(tasks, types.Task{
						ID:       repo.Name + "/clone",
						RepoName: repo.Name,
						Group:    repo.Group,
						Command:  []string{"git", "clone", repo.URL, repo.Path},
					})
				}
				// URL is empty: nothing to clone, skip this repo.
			} else {
				tasks = append(tasks, types.Task{
					ID:       repo.Name + "/fetch",
					RepoName: repo.Name,
					Group:    repo.Group,
					Command:  []string{"git", "-C", repo.Path, "fetch", "origin"},
				})
				tasks = append(tasks, types.Task{
					ID:       repo.Name + "/merge",
					RepoName: repo.Name,
					Group:    repo.Group,
					Command:  []string{"git", "-C", repo.Path, "merge", "--ff-only"},
				})
			}
		} else {
			// --- With checkout ---
			if !exists {
				if repo.URL != "" {
					// Clone first, then checkout and pull.
					tasks = append(tasks, types.Task{
						ID:       repo.Name + "/clone",
						RepoName: repo.Name,
						Group:    repo.Group,
						Command:  []string{"git", "clone", repo.URL, repo.Path},
					})
					tasks = append(tasks, types.Task{
						ID:       repo.Name + "/checkout",
						RepoName: repo.Name,
						Group:    repo.Group,
						Command:  []string{"git", "-C", repo.Path, "checkout", r.Checkout},
					})
					tasks = append(tasks, types.Task{
						ID:       repo.Name + "/pull",
						RepoName: repo.Name,
						Group:    repo.Group,
						Command:  []string{"git", "-C", repo.Path, "pull"},
					})
				}
				// URL is empty: nothing to clone, skip this repo.
			} else {
				dirty, err := isDirty(repo.Path)
				if err != nil {
					return nil, err
				}

				if dirty && !r.Force {
					// Warn and skip checkout.
					tasks = append(tasks, types.Task{
						ID:       repo.Name + "/warning",
						RepoName: repo.Name,
						Group:    repo.Group,
						Command:  []string{"echo", "WARNING: dirty repo, skipping checkout"},
					})
				} else if dirty && r.Force {
					// Stash, then checkout, then pull.
					tasks = append(tasks, types.Task{
						ID:       repo.Name + "/stash",
						RepoName: repo.Name,
						Group:    repo.Group,
						Command:  []string{"git", "-C", repo.Path, "stash"},
					})
					tasks = append(tasks, types.Task{
						ID:       repo.Name + "/checkout",
						RepoName: repo.Name,
						Group:    repo.Group,
						Command:  []string{"git", "-C", repo.Path, "checkout", r.Checkout},
					})
					tasks = append(tasks, types.Task{
						ID:       repo.Name + "/pull",
						RepoName: repo.Name,
						Group:    repo.Group,
						Command:  []string{"git", "-C", repo.Path, "pull"},
					})
				} else {
					// Clean: checkout then pull.
					tasks = append(tasks, types.Task{
						ID:       repo.Name + "/checkout",
						RepoName: repo.Name,
						Group:    repo.Group,
						Command:  []string{"git", "-C", repo.Path, "checkout", r.Checkout},
					})
					tasks = append(tasks, types.Task{
						ID:       repo.Name + "/pull",
						RepoName: repo.Name,
						Group:    repo.Group,
						Command:  []string{"git", "-C", repo.Path, "pull"},
					})
				}
			}
		}
	}

	return tasks, nil
}

// pathExists reports whether the given filesystem path exists.
func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// isDirty checks whether a git repository at the given path has uncommitted changes.
func isDirty(path string) (bool, error) {
	cmd := exec.Command("git", "-C", path, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(out) > 0, nil
}
