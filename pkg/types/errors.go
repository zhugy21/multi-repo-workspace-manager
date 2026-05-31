package types

import "fmt"

// ConfigError represents a configuration-related error.
type ConfigError struct {
	File    string
	Message string
	Err     error
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("config error in %s: %s: %s", e.File, e.Message, e.Err)
}

// RepoError represents a repository-related error.
type RepoError struct {
	Repo    string
	Message string
	Err     error
}

func (e *RepoError) Error() string {
	return fmt.Sprintf("repo %s: %s: %s", e.Repo, e.Message, e.Err)
}

// CommandError represents a command execution error.
type CommandError struct {
	Repo      string
	Command   string
	ExitCode  int
	Stderr    string
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("repo %s: command '%s' exited with code %d: %s", e.Repo, e.Command, e.ExitCode, e.Stderr)
}

// TimeoutError represents a timeout error.
type TimeoutError struct {
	Repo    string
	Timeout string
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("repo %s: timed out after %s", e.Repo, e.Timeout)
}

// CancelError represents a cancellation error.
type CancelError struct {
	Repo   string
	Reason string
}

func (e *CancelError) Error() string {
	return fmt.Sprintf("repo %s: cancelled: %s", e.Repo, e.Reason)
}

// WorkspaceError represents a workspace-related error.
type WorkspaceError struct {
	Name    string
	Message string
	Err     error
}

func (e *WorkspaceError) Error() string {
	return fmt.Sprintf("workspace %s: %s: %s", e.Name, e.Message, e.Err)
}
