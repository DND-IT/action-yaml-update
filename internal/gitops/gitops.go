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
	cwd, _ := os.Getwd()
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

// CreateBranch creates and checks out a new branch from a base.
func CreateBranch(name, base string) error {
	if err := run("git", "fetch", "origin", base); err != nil {
		return err
	}
	return run("git", "checkout", "-b", name, "origin/"+base)
}

// CommitAndPush stages files, commits, pushes, and returns the commit SHA.
func CommitAndPush(files []string, message, branch string) (string, error) {
	for _, f := range files {
		if err := run("git", "add", f); err != nil {
			return "", err
		}
	}

	if err := run("git", "commit", "-m", message); err != nil {
		return "", err
	}

	if err := run("git", "push", "-u", "origin", branch); err != nil {
		return "", err
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
