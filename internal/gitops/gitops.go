// Package gitops provides Git operations via subprocess.
package gitops

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Configure sets up git with user info and authentication.
func Configure(userName, userEmail, token, repository, serverURL string) error {
	// Mark workspace as safe before any local git operations
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	if err := run("git", "config", "--global", "--add", "safe.directory", cwd); err != nil {
		return err
	}

	if err := run("git", "config", "user.name", userName); err != nil {
		return err
	}
	if err := run("git", "config", "user.email", userEmail); err != nil {
		return err
	}

	// Set up authenticated remote
	if token != "" && repository != "" {
		host := strings.TrimPrefix(strings.TrimPrefix(serverURL, "https://"), "http://")
		remoteURL := fmt.Sprintf("https://x-access-token:%s@%s/%s.git", token, host, repository)
		if err := run("git", "remote", "set-url", "origin", remoteURL); err != nil {
			return err
		}
	}

	return nil
}

// GetDefaultBranch returns the default branch name.
func GetDefaultBranch() string {
	out, err := output("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	if err != nil {
		return "main"
	}
	return strings.TrimPrefix(strings.TrimSpace(out), "refs/remotes/origin/")
}

// PrepareWorktree fetches origin/<base> and resets the working tree to a fresh
// <branch> based on it.
//
// It must run BEFORE any YAML edits: at that point the working tree is clean, so
// switching away from whatever ref the caller checked out (e.g. a release tag on
// a diverged hotfix branch) never hits "local changes would be overwritten".
// Editing afterwards then makes the changes — and the resulting commit/PR —
// relative to origin/<base>, independent of the caller's checkout. When <branch>
// equals <base>, this simply checks out the up-to-date base for a direct commit.
func PrepareWorktree(branch, base string) error {
	if err := run("git", "fetch", "origin", base); err != nil {
		return err
	}
	// -B creates or resets the branch to origin/<base>, discarding any stale
	// local branch of the same name from a previous run.
	return run("git", "checkout", "-B", branch, "origin/"+base)
}

// CommitAndPush stages files, commits, and force-pushes to origin.
// The branch always represents HEAD + the YAML changes, so force push is the default.
func CommitAndPush(files []string, message, branch string) (string, error) {
	for _, f := range files {
		if err := run("git", "add", f); err != nil {
			return "", err
		}
	}

	if err := run("git", "commit", "-m", message); err != nil {
		return "", err
	}

	// Fetch the remote branch so --force-with-lease has a valid reference.
	// Ignore errors: the branch may not exist on the remote yet.
	_ = run("git", "fetch", "origin", branch)

	if err := run("git", "push", "--force-with-lease", "-u", "origin", branch); err != nil {
		return "", fmt.Errorf("push failed: %w", err)
	}

	sha, err := output("git", "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(sha), nil
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func output(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.Output()
	return string(out), err
}
