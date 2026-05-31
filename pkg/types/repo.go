package types

import "time"

// Status represents the execution status of a task.
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusSuccess   Status = "success"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
	StatusSkipped   Status = "skipped"
	StatusWarning   Status = "warning"
)

// Repo represents a single repository in the workspace config.
type Repo struct {
	Name              string        `yaml:"name" json:"name"`
	Path              string        `yaml:"path" json:"path"`
	URL               string        `yaml:"url,omitempty" json:"url,omitempty"`
	Type              string        `yaml:"type,omitempty" json:"type,omitempty"`
	Group             string        `yaml:"group,omitempty" json:"group,omitempty"`
	DefaultBranch     string        `yaml:"default_branch,omitempty" json:"default_branch,omitempty"`
	BuildCommand      string        `yaml:"build_command,omitempty" json:"build_command,omitempty"`
	BuildTimeout      time.Duration `yaml:"build_timeout,omitempty" json:"build_timeout_ns,omitempty"`
	HealthCommand     string        `yaml:"health_command,omitempty" json:"health_command,omitempty"`
	HealthTimeout     time.Duration `yaml:"health_timeout,omitempty" json:"health_timeout_ns,omitempty"`
	EnvFile           string        `yaml:"env_file,omitempty" json:"env_file,omitempty"`
	DockerComposeFile string        `yaml:"docker_compose_file,omitempty" json:"docker_compose_file,omitempty"`
	SyncCommand       string        `yaml:"sync_command,omitempty" json:"sync_command,omitempty"`
	SmartBuild        bool          `yaml:"smart_build,omitempty" json:"smart_build,omitempty"`
}

// Task is the input to the executor engine.
type Task struct {
	ID       string        `json:"id"`
	RepoName string        `json:"repo_name"`
	Group    string        `json:"group,omitempty"`
	Command  []string      `json:"command"`
	Timeout  time.Duration `json:"timeout_ns"` // 0 means use global default
	EnvFiles []string      `json:"env_files,omitempty"`
}

// Result is the output from executing a single task.
type Result struct {
	TaskID   string        `json:"task_id"`
	RepoName string        `json:"repo_name"`
	Group    string        `json:"group,omitempty"`
	Status   Status        `json:"status"`
	Detail   string        `json:"detail,omitempty"`
	Error    error         `json:"-"`
	ErrorStr string        `json:"error,omitempty"`
	Duration time.Duration `json:"duration_ns"`
	ExitCode int           `json:"exit_code"`
}

// Summary aggregates results from a batch execution.
type Summary struct {
	Total     int `json:"total"`
	Success   int `json:"success"`
	Failed    int `json:"failed"`
	Cancelled int `json:"cancelled"`
	Skipped   int `json:"skipped"`
	Warning   int `json:"warning"`
}

// Filter selects which repos to operate on.
type Filter struct {
	All   bool   `json:"all"`
	Group string `json:"group,omitempty"`
	Repo  string `json:"repo,omitempty"`
}

// HasFailures returns true if any task failed.
func (s Summary) HasFailures() bool {
	return s.Failed > 0
}

// Summarize counts results by status and returns a Summary.
// pending and running statuses are counted in Total but not in any category.
func Summarize(results []Result) Summary {
	s := Summary{}
	for _, r := range results {
		s.Total++
		switch r.Status {
		case StatusSuccess:
			s.Success++
		case StatusFailed:
			s.Failed++
		case StatusCancelled:
			s.Cancelled++
		case StatusSkipped:
			s.Skipped++
		case StatusWarning:
			s.Warning++
		default:
			// pending, running, or unknown — counted in Total only
		}
	}
	return s
}
