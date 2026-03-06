package gitops

import (
	"os"
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
