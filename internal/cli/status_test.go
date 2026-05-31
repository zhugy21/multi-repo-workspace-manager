package cli

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestStatusCommandGitRepo verifies that status on a real git repo produces
// output containing "branch:", "dirty:", and "last_commit:".
func TestStatusCommandGitRepo(t *testing.T) {
	dir := t.TempDir()
	repoDir := filepath.Join(dir, "myrepo")

	// Initialize a git repository.
	if out, err := exec.Command("git", "init", repoDir).CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}

	// Configure git user for the commit.
	for _, args := range [][]string{
		{"-C", repoDir, "config", "user.email", "test@test.com"},
		{"-C", repoDir, "config", "user.name", "Test"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git config %v failed: %v\n%s", args, err, out)
		}
	}

	// Create a file and commit.
	if err := os.WriteFile(filepath.Join(repoDir, "test.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"-C", repoDir, "add", "."},
		{"-C", repoDir, "commit", "-m", "initial commit"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	// Write workspace.yaml pointing to the repo.
	wsContent := []byte(`workspace_name: "test"
repos:
  - name: "myrepo"
    path: "` + repoDir + `"
`)
	if err := os.WriteFile(filepath.Join(dir, "workspace.yaml"), wsContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Change to the test directory so resolveWorkspacePath finds workspace.yaml.
	cwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(cwd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Capture stdout.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runStatus(&cobra.Command{}, nil)

	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	output := buf.String()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	for _, want := range []string{"branch:", "dirty:", "last_commit:"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, output)
		}
	}
}

// TestStatusCommandMissingPath verifies that a repo path that does not exist
// produces output containing "MISSING".
func TestStatusCommandMissingPath(t *testing.T) {
	dir := t.TempDir()

	// Write workspace.yaml pointing to a non-existent path.
	wsContent := []byte(`workspace_name: "test"
repos:
  - name: "missing-repo"
    path: "/nonexistent/path/for/testing"
`)
	if err := os.WriteFile(filepath.Join(dir, "workspace.yaml"), wsContent, 0644); err != nil {
		t.Fatal(err)
	}

	cwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(cwd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Capture stdout.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runStatus(&cobra.Command{}, nil)

	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	output := buf.String()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(output, "MISSING") {
		t.Errorf("expected output to contain \"MISSING\", got:\n%s", output)
	}
}

// TestStatusCommandFilteredByGroup verifies that when a group filter is active,
// only repos belonging to that group appear in the output.
func TestStatusCommandFilteredByGroup(t *testing.T) {
	dir := t.TempDir()

	// Create two repo directories (non-git is fine — we just need them to exist).
	repoDir1 := filepath.Join(dir, "repo1")
	repoDir2 := filepath.Join(dir, "repo2")
	if err := os.MkdirAll(repoDir1, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(repoDir2, 0755); err != nil {
		t.Fatal(err)
	}

	// Write workspace.yaml with two groups.
	wsContent := []byte(`workspace_name: "test"
groups:
  frontend: ["repo1"]
  backend: ["repo2"]
repos:
  - name: "repo1"
    path: "` + repoDir1 + `"
    group: frontend
  - name: "repo2"
    path: "` + repoDir2 + `"
    group: backend
`)
	if err := os.WriteFile(filepath.Join(dir, "workspace.yaml"), wsContent, 0644); err != nil {
		t.Fatal(err)
	}

	cwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(cwd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Set the group filter to "frontend" and ensure it is restored.
	groupFlag = "frontend"
	defer func() { groupFlag = "" }()

	// Capture stdout.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runStatus(&cobra.Command{}, nil)

	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	output := buf.String()

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(output, "repo1") {
		t.Errorf("expected output to contain \"repo1\" (matching group), got:\n%s", output)
	}
	if strings.Contains(output, "repo2") {
		t.Errorf("expected output NOT to contain \"repo2\" (non-matching group), got:\n%s", output)
	}
}
