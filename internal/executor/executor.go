package executor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/user/ws/pkg/types"
)

// Executor runs tasks concurrently with optional fail-fast behavior.
type Executor struct {
	Concurrency int
	FailFast    bool
}

// Run executes all tasks concurrently, throttled by the semaphore.
// Returns results in the same order as the input tasks.
func (e *Executor) Run(ctx context.Context, tasks []types.Task) []types.Result {
	concurrency := e.Concurrency
	if concurrency <= 0 {
		concurrency = 4
	}

	// Create a cancellable child context so we can propagate fail-fast.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make([]types.Result, len(tasks))
	sem := make(chan struct{}, concurrency)

	var (
		mu         sync.Mutex
		failedOnce bool
		wg         sync.WaitGroup
	)

	for i, task := range tasks {
		// If the context is already cancelled, skip this task.
		if ctx.Err() != nil {
			results[i] = types.Result{
				TaskID:   task.ID,
				RepoName: task.RepoName,
				Group:    task.Group,
				Status:   types.StatusSkipped,
				Detail:   "Skipped (context cancelled)",
			}
			continue
		}

		// Acquire semaphore slot.
		sem <- struct{}{}
		wg.Add(1)

		go func(idx int, t types.Task) {
			defer func() {
				<-sem // release semaphore slot
				wg.Done()
			}()

			result := e.runTask(ctx, t)
			mu.Lock()
			results[idx] = result

			// Fail-fast: cancel the context on the first non-cancellation failure.
			if e.FailFast && result.Status == types.StatusFailed && !failedOnce {
				failedOnce = true
				cancel()
			}
			mu.Unlock()
		}(i, task)
	}

	wg.Wait()
	return results
}

// runTask executes a single task with its own timeout context.
func (e *Executor) runTask(parentCtx context.Context, t types.Task) types.Result {
	timeout := t.Timeout
	if timeout == 0 {
		timeout = 2 * time.Minute
	}

	taskCtx, taskCancel := context.WithTimeout(parentCtx, timeout)
	defer taskCancel()

	cmd := exec.CommandContext(taskCtx, t.Command[0], t.Command[1:]...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	elapsed := time.Since(start)

	result := types.Result{
		TaskID:   t.ID,
		RepoName: t.RepoName,
		Group:    t.Group,
		Duration: elapsed,
	}

	if err != nil {
		if taskCtx.Err() == context.DeadlineExceeded {
			result.Status = types.StatusFailed
			msg := fmt.Sprintf("Timeout after %v", elapsed.Round(time.Millisecond))
			result.Detail = msg
			result.ErrorStr = msg
		} else if parentCtx.Err() != nil {
			result.Status = types.StatusCancelled
			result.Detail = "Cancelled (fail-fast triggered)"
			result.ErrorStr = result.Detail
		} else {
			result.Status = types.StatusFailed
			result.Detail = stderr.String()
			result.ErrorStr = stderr.String()
			if exitErr, ok := err.(*exec.ExitError); ok {
				result.ExitCode = exitErr.ExitCode()
			}
		}
	} else {
		result.Status = types.StatusSuccess
		result.Detail = stdout.String()
	}

	return result
}
