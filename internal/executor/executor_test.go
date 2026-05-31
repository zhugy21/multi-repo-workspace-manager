package executor

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/user/ws/pkg/types"
)

func TestExecutorAllSuccess(t *testing.T) {
	tasks := []types.Task{
		{ID: "1", Command: []string{"echo", "ok"}},
		{ID: "2", Command: []string{"echo", "ok"}},
	}

	e := &Executor{Concurrency: 2, FailFast: false}
	results := e.Run(context.Background(), tasks)

	require.Len(t, results, 2)
	assert.Equal(t, types.StatusSuccess, results[0].Status)
	assert.Equal(t, types.StatusSuccess, results[1].Status)
}

func TestExecutorFailFast(t *testing.T) {
	tasks := []types.Task{
		{ID: "1", Command: []string{"sh", "-c", "sleep 0.1; exit 0"}},
		{ID: "2", Command: []string{"sh", "-c", "exit 1"}},
		{ID: "3", Command: []string{"sh", "-c", "sleep 0.2; exit 0"}},
	}

	e := &Executor{Concurrency: 2, FailFast: true}
	results := e.Run(context.Background(), tasks)

	require.Len(t, results, 3)

	// Task 2 must have failed (exit 1).
	assert.Equal(t, types.StatusFailed, results[1].Status)

	// Due to fail-fast, at least one of task 1 or task 3 should be
	// cancelled (started but cancelled mid-flight) or skipped (never started).
	cancelledOrSkipped := results[0].Status == types.StatusCancelled || results[0].Status == types.StatusSkipped ||
		results[2].Status == types.StatusCancelled || results[2].Status == types.StatusSkipped
	assert.True(t, cancelledOrSkipped, "expected at least one of task 1 or task 3 to be cancelled or skipped")
}

func TestExecutorTimeout(t *testing.T) {
	task := types.Task{
		ID:      "1",
		Command: []string{"sleep", "10"},
		Timeout: 100 * time.Millisecond,
	}

	e := &Executor{Concurrency: 1, FailFast: false}
	results := e.Run(context.Background(), []types.Task{task})

	require.Len(t, results, 1)
	assert.Equal(t, types.StatusFailed, results[0].Status)
	assert.True(t, strings.Contains(strings.ToLower(results[0].ErrorStr), "timeout") ||
		strings.Contains(strings.ToLower(results[0].ErrorStr), "deadline"),
		"ErrorStr should contain 'timeout' or 'deadline', got: %q", results[0].ErrorStr)
}

func TestExecutorContinueOnError(t *testing.T) {
	tasks := []types.Task{
		{ID: "1", Command: []string{"echo", "ok"}},
		{ID: "2", Command: []string{"sh", "-c", "exit 1"}},
		{ID: "3", Command: []string{"echo", "ok"}},
	}

	e := &Executor{Concurrency: 2, FailFast: false}
	results := e.Run(context.Background(), tasks)

	require.Len(t, results, 3)

	summary := types.Summarize(results)
	assert.Equal(t, 3, summary.Total)
	assert.Equal(t, 2, summary.Success)
	assert.Equal(t, 1, summary.Failed)
}

func TestExecutorAllSkippedIfContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel context

	tasks := []types.Task{
		{ID: "1", Command: []string{"echo", "ok"}},
		{ID: "2", Command: []string{"echo", "ok"}},
		{ID: "3", Command: []string{"echo", "ok"}},
	}

	e := &Executor{Concurrency: 2, FailFast: false}
	results := e.Run(ctx, tasks)

	require.Len(t, results, 3)
	for i, r := range results {
		assert.Equal(t, types.StatusSkipped, r.Status, "task %d should be skipped", i)
	}
}
