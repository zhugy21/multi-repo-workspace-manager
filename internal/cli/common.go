package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/user/ws/internal/workspace"
	"github.com/user/ws/pkg/types"
)

// StatusIcon returns an emoji icon for the given status.
func StatusIcon(s types.Status) string {
	switch s {
	case types.StatusSuccess:
		return "✅" // ✅
	case types.StatusFailed:
		return "✗" // ✗
	case types.StatusCancelled:
		return "⏹" // ⏹
	case types.StatusSkipped:
		return "○" // ◌
	case types.StatusWarning:
		return "⚠" // ⚠
	default:
		return "?"
	}
}

// resolveWorkspacePath determines the workspace.yaml path to use.
// Priority:
//  1. --workspace flag value (must exist)
//  2. ./workspace.yaml in current directory
//  3. ~/.ws/config.yaml → read "active:" field → use that path
//  4. Fallback: return ./workspace.yaml (will fail naturally if not found)
func resolveWorkspacePath() (string, error) {
	if workspaceFlag != "" {
		// Check the explicitly specified path exists.
		if _, err := os.Stat(workspaceFlag); err != nil {
			return "", fmt.Errorf("workspace file not found: %s: %w", workspaceFlag, err)
		}
		return workspaceFlag, nil
	}

	// Check current directory.
	cwd, err := os.Getwd()
	if err == nil {
		candidate := filepath.Join(cwd, "workspace.yaml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	// Check ~/.ws/config.yaml for an active workspace reference.
	usr, err := user.Current()
	if err == nil {
		wsConfigPath := filepath.Join(usr.HomeDir, ".ws", "config.yaml")
		if data, err := os.ReadFile(wsConfigPath); err == nil {
			var ac workspace.ActiveConfig
			if err := unmarshalYAML(data, &ac); err == nil && ac.Active != "" {
				expanded := expandPath(ac.Active)
				if _, err := os.Stat(expanded); err == nil {
					return expanded, nil
				}
			}
		}
	}

	// Fallback — will fail naturally when the caller tries to read it.
	return filepath.Join(cwd, "workspace.yaml"), nil
}

// expandPath replaces a leading ~ with the user's home directory.
func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		usr, err := user.Current()
		if err == nil {
			return filepath.Join(usr.HomeDir, p[2:])
		}
	}
	return p
}

// unmarshalYAML is a minimal YAML unmarshaler for the ActiveConfig struct.
// It avoids importing a YAML library just for this helper.
func unmarshalYAML(data []byte, v *workspace.ActiveConfig) error {
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "active:") {
			val := strings.TrimSpace(line[7:])
			val = strings.Trim(val, `"'`)
			v.Active = val
			return nil
		}
	}
	return fmt.Errorf("active field not found")
}

// resolveFilter builds a Filter from the global flag values.
// --repo takes precedence over --group.
func resolveFilter() types.Filter {
	if repoFlag != "" {
		if groupFlag != "" {
			fmt.Fprintf(os.Stderr, "warning: --repo takes precedence over --group\n")
		}
		return types.Filter{Repo: repoFlag}
	}
	if groupFlag != "" {
		return types.Filter{Group: groupFlag}
	}
	return types.Filter{All: allFlag}
}

// resolveFailFast returns true when execution should stop at the first failure.
func resolveFailFast() bool {
	if failFast {
		return true
	}
	return !continueOnErr
}

// resolveConcurrency returns the configured concurrency level.
// A return value of 0 means "use config default".
func resolveConcurrency() int {
	return concurrency
}

// resolveTimeout parses the timeoutFlag string as a duration.
// Returns 0 if not set or the string is invalid.
func resolveTimeout() time.Duration {
	if timeoutFlag == "" {
		return 0
	}
	d, err := time.ParseDuration(timeoutFlag)
	if err != nil {
		return 0
	}
	return d
}

// resolveOutputFormat returns the output format, preferring the --output flag
// over the workspace config default.
func resolveOutputFormat(ws *workspace.Workspace) string {
	if outputFlag != "" {
		return outputFlag
	}
	return ws.Config.OutputFormat
}

// displayResults dispatches to the appropriate output format.
func displayResults(results []types.Result, format, command string) {
	if format == "json" {
		displayJSON(results, command)
	} else {
		displayPlain(results)
	}
}

// displayPlain prints one line per result followed by a summary line.
func displayPlain(results []types.Result) {
	for _, r := range results {
		icon := StatusIcon(r.Status)
		duration := r.Duration.Round(time.Millisecond)
		detail := r.Detail
		if detail == "" {
			detail = r.ErrorStr
		}
		// Trim trailing newlines for cleaner output.
		detail = strings.TrimRight(detail, "\n")
		if detail != "" {
			fmt.Printf("[%s] %s %s  %s  %v\n", r.RepoName, icon, r.Status, detail, duration)
		} else {
			fmt.Printf("[%s] %s %s  %v\n", r.RepoName, icon, r.Status, duration)
		}
	}

	summary := types.Summarize(results)
	total := summary.Total
	fmt.Printf("---\n%d repos: %d passed, %d failed, %d cancelled, %d skipped, %d warning\n",
		total, summary.Success, summary.Failed, summary.Cancelled, summary.Skipped, summary.Warning)
}

// displayJSON outputs the results as a JSON object to stdout.
func displayJSON(results []types.Result, command string) {
	summary := types.Summarize(results)
	out := map[string]interface{}{
		"workspace": "",
		"command":   command,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"results":   results,
		"summary":   summary,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}

// exitCodeFromResults returns an error if any repo failed, nil otherwise.
func exitCodeFromResults(results []types.Result) error {
	summary := types.Summarize(results)
	if summary.Failed > 0 {
		return fmt.Errorf("%d repos failed", summary.Failed)
	}
	return nil
}
