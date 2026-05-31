package types

import "fmt"

// ConfigError is returned when workspace.yaml parsing or validation fails.
type ConfigError struct {
	File    string
	Message string
	Err     error
}

func (e *ConfigError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("config error in %s: %s: %v", e.File, e.Message, e.Err)
	}
	return fmt.Sprintf("config error in %s: %s", e.File, e.Message)
}

func (e *ConfigError) Unwrap() error { return e.Err }

// RepoError is returned when a git operation on a repository fails.
type RepoError struct {
	Repo    string
	Message string
	Err     error
}

func (e *RepoError) Error() string {
	return fmt.Sprintf("repo %s: %s: %v", e.Repo, e.Message, e.Err)
}

func (e *RepoError) Unwrap() error { return e.Err }

// CommandError is returned when a build/health command exits non-zero.
type CommandError struct {
	Repo     string
	Command  string
	ExitCode int
	Stderr   string
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("repo %s: command '%s' exit code %d: %s",
		e.Repo, e.Command, e.ExitCode, e.Stderr)
}

// TimeoutError is returned when a task exceeds its timeout.
type TimeoutError struct {
	Repo    string
	Timeout string
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("repo %s: timed out after %s", e.Repo, e.Timeout)
}

// CancelError is returned when a task is cancelled due to fail-fast.
type CancelError struct {
	Repo   string
	Reason string
}

func (e *CancelError) Error() string {
	return fmt.Sprintf("repo %s: cancelled: %s", e.Repo, e.Reason)
}

// WorkspaceError is returned when workspace switching/lookup fails.
type WorkspaceError struct {
	Name    string
	Message string
	Err     error
}

func (e *WorkspaceError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("workspace %s: %s: %v", e.Name, e.Message, e.Err)
	}
	return fmt.Sprintf("workspace %s: %s", e.Name, e.Message)
}

func (e *WorkspaceError) Unwrap() error { return e.Err }
