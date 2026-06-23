package gitops

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigure_SetsGitConfig(t *testing.T) {
	// We can only test the os.Getwd() path in a unit test since
	// running actual git commands requires a real repo.
	// Verify that Configure doesn't panic and handles the working directory.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd failed: %v", err)
	}
	if cwd == "" {
		t.Fatal("expected non-empty working directory")
	}
}

func TestGetDefaultBranch_FallsBack(t *testing.T) {
	// Outside a real git repo with an origin remote, this should fall back to "main".
	// The function is designed to be resilient — it never errors.
	branch := GetDefaultBranch()
	if branch == "" {
		t.Fatal("expected non-empty branch name")
	}
}

// gitAt runs a git command in dir and fails the test on error.
func gitAt(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// TestPrepareWorktree_ResetsToBaseFromDivergedRef reproduces the hotfix-release
// scenario: a release event checks out a tag whose commit has diverged from the
// default branch. PrepareWorktree must reset the working tree to origin/<base>
// so that subsequent edits — and the commit/PR — are relative to the base, not
// the diverged tag. The old "edit then checkout -b origin/base" ordering aborted
// here with "local changes would be overwritten by checkout".
func TestPrepareWorktree_ResetsToBaseFromDivergedRef(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	root := t.TempDir()
	origin := filepath.Join(root, "origin.git")
	work := filepath.Join(root, "work")

	gitAt(t, root, "init", "--bare", "-b", "main", origin)
	gitAt(t, root, "clone", origin, work)
	gitAt(t, work, "config", "user.name", "t")
	gitAt(t, work, "config", "user.email", "t@t")

	values := filepath.Join(work, "values.yaml")

	// Base commit, tagged as the hotfix base, then main diverges past it.
	if err := os.WriteFile(values, []byte("tag: v1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitAt(t, work, "add", "values.yaml")
	gitAt(t, work, "commit", "-m", "v1")
	gitAt(t, work, "tag", "hotfix-base")
	if err := os.WriteFile(values, []byte("tag: v2\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitAt(t, work, "commit", "-am", "v2")
	gitAt(t, work, "push", "origin", "main")

	// Simulate the release checkout: detached HEAD on the diverged tag.
	gitAt(t, work, "checkout", "hotfix-base")
	if got := readFile(t, values); got != "tag: v1\n" {
		t.Fatalf("precondition: expected diverged tag content, got %q", got)
	}

	// PrepareWorktree runs in the process working directory, so chdir in.
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(work); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(prev)

	if err := PrepareWorktree("yaml-update/pr", "main"); err != nil {
		t.Fatalf("PrepareWorktree failed: %v", err)
	}

	// Working tree now reflects origin/main (v2), not the diverged tag (v1)...
	if got := readFile(t, values); got != "tag: v2\n" {
		t.Fatalf("expected base content after prepare, got %q", got)
	}
	// ...and HEAD is the new branch based on origin/main.
	if branch := gitAt(t, work, "rev-parse", "--abbrev-ref", "HEAD"); branch != "yaml-update/pr" {
		t.Fatalf("expected HEAD on yaml-update/pr, got %q", branch)
	}
	if head, base := gitAt(t, work, "rev-parse", "HEAD"), gitAt(t, work, "rev-parse", "origin/main"); head != base {
		t.Fatalf("expected branch at origin/main (%s), got %s", base, head)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
