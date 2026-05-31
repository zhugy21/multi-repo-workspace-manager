package workspace

import (
	"os"
	"strings"

	"github.com/user/ws/pkg/types"
)

// Filter selects repos from the workspace based on the provided filter criteria.
func (ws *Workspace) Filter(f types.Filter) []types.Repo {
	// 1. Repo filter takes precedence
	if f.Repo != "" {
		if f.Group != "" {
			// Print warning to stderr
			_, _ = os.Stderr.WriteString("warning: --repo takes precedence over --group\n")
		}
		names := strings.Split(f.Repo, ",")
		nameSet := make(map[string]bool, len(names))
		for _, n := range names {
			nameSet[strings.TrimSpace(n)] = true
		}
		result := make([]types.Repo, 0, len(names))
		for _, r := range ws.Repos {
			if nameSet[r.Name] {
				result = append(result, r)
			}
		}
		return result
	}

	// 2. Group filter
	if f.Group != "" {
		groupNames := strings.Split(f.Group, ",")
		seen := make(map[string]bool)
		result := make([]types.Repo, 0)
		for _, g := range groupNames {
			g = strings.TrimSpace(g)
			members, ok := ws.Groups[g]
			if !ok {
				continue
			}
			for _, m := range members {
				if seen[m] {
					continue
				}
				seen[m] = true
				for _, r := range ws.Repos {
					if r.Name == m {
						result = append(result, r)
						break
					}
				}
			}
		}
		return result
	}

	// 3. All (default)
	return ws.Repos
}
