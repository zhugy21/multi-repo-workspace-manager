package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// wsBinary is the path to the pre-built ws CLI binary used by all tests.
var wsBinary string

func init() {
	// Determine the path to cmd/ws from the test package directory.
	cwd, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("failed to get working directory: %v", err))
	}
	wsPkg := filepath.Join(cwd, "../../cmd/ws")

	// Build the binary once to a temp directory so we can run it from
	// any working directory (the temp dir for each test).
	buildDir, err := os.MkdirTemp("", "ws-e2e-*")
	if err != nil {
		panic(fmt.Sprintf("failed to create temp dir for binary: %v", err))
	}
	wsBinary = filepath.Join(buildDir, "ws")

	build := exec.Command("go", "build", "-o", wsBinary, wsPkg)
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		panic(fmt.Sprintf("failed to build ws binary: %v", err))
	}
}

// wsCmd returns an exec.Cmd that runs the ws CLI binary with the given arguments.
// The caller should set cmd.Dir to the desired working directory (usually a temp dir).
func wsCmd(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()
	return exec.Command(wsBinary, args...)
}

// makeGitRepo creates a git repo at dir with an initial empty commit.
func makeGitRepo(t *testing.T, dir string) {
	t.Helper()
	run := func(args ...string) {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git command %v failed: %v", args, err)
		}
	}
	run("git", "init")
	run("git", "config", "user.email", "test@test.com")
	run("git", "config", "user.name", "Test")
	run("git", "commit", "--allow-empty", "-m", "initial")
}

// makeGitRepoWithRemote creates a git repo with a local bare remote and pushes the initial commit.
// Returns the path to the working repo.
func makeGitRepoWithRemote(t *testing.T, parentDir, name string) string {
	t.Helper()

	repoDir := filepath.Join(parentDir, name)
	bareDir := filepath.Join(parentDir, name+".git")

	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("failed to create repo dir: %v", err)
	}

	// Create bare repo
	if err := exec.Command("git", "init", "--bare", bareDir).Run(); err != nil {
		t.Fatalf("git init --bare failed: %v", err)
	}

	// Create working repo with initial commit
	makeGitRepo(t, repoDir)

	// Get current branch name
	out, err := exec.Command("git", "-C", repoDir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		t.Fatalf("failed to get branch name: %v", err)
	}
	branch := strings.TrimSpace(string(out))

	// Add remote and push
	run := func(args ...string) {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repoDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git command %v failed: %v", args, err)
		}
	}
	run("git", "remote", "add", "origin", bareDir)
	run("git", "push", "--set-upstream", "origin", branch)

	return repoDir
}

// makeRepoWithDivergedHistory creates a repo where git merge --ff-only will fail
// because local and remote have diverged.
func makeRepoWithDivergedHistory(t *testing.T, parentDir, name string) string {
	t.Helper()

	repoDir := makeGitRepoWithRemote(t, parentDir, name)

	// Get current branch name
	out, err := exec.Command("git", "-C", repoDir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		t.Fatalf("failed to get branch: %v", err)
	}
	branch := strings.TrimSpace(string(out))

	// Clone the bare repo to a temporary working dir
	bareDir := filepath.Join(parentDir, name+".git")
	otherDir := filepath.Join(parentDir, name+"-other")
	if err := exec.Command("git", "clone", bareDir, otherDir).Run(); err != nil {
		t.Fatalf("git clone failed: %v", err)
	}

	// Configure git in the other clone
	for _, args := range [][]string{
		{"-C", otherDir, "config", "user.email", "test@test.com"},
		{"-C", otherDir, "config", "user.name", "Test"},
	} {
		if err := exec.Command("git", args...).Run(); err != nil {
			t.Fatalf("git config failed: %v", err)
		}
	}

	// Make a commit in the other clone and push (updates the remote's branch)
	for _, args := range [][]string{
		{"-C", otherDir, "commit", "--allow-empty", "-m", "remote change"},
		{"-C", otherDir, "push", "origin", branch},
	} {
		if err := exec.Command("git", args...).Run(); err != nil {
			t.Fatalf("git %v failed: %v", args, err)
		}
	}

	// Make a local commit in the original repo (don't push — creates divergence)
	cmd := exec.Command("git", "-C", repoDir, "commit", "--allow-empty", "-m", "local change")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create local diverged commit: %v", err)
	}

	return repoDir
}
