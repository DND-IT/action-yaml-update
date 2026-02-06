"""Git CLI wrapper for branch creation, commits, and pushes."""

from __future__ import annotations

import subprocess


class GitError(Exception):
    """Raised when a git command fails."""


def _run(args: list[str], check: bool = True) -> subprocess.CompletedProcess[str]:
    """Run a git command and return the result."""
    result = subprocess.run(
        ["git", *args],
        capture_output=True,
        text=True,
    )
    if check and result.returncode != 0:
        raise GitError(f"git {' '.join(args)} failed: {result.stderr.strip()}")
    return result


def configure(
    name: str,
    email: str,
    token: str,
    repository: str,
    server_url: str = "https://github.com",
) -> None:
    """Configure git user, safe directory, and remote URL with token."""
    _run(["config", "user.name", name])
    _run(["config", "user.email", email])
    _run(["config", "--global", "--add", "safe.directory", "/github/workspace"])

    if token and repository:
        remote_url = f"{server_url}/{repository}.git"
        remote_url = remote_url.replace("https://", f"https://x-access-token:{token}@")
        _run(["remote", "set-url", "origin", remote_url])


def fetch(branch: str | None = None) -> None:
    """Fetch from origin."""
    if branch:
        _run(["fetch", "origin", branch])
    else:
        _run(["fetch", "origin"])


def create_branch(name: str, base: str | None = None) -> None:
    """Create and checkout a new branch, optionally from a base."""
    if base:
        fetch(base)
        _run(["checkout", "-b", name, f"origin/{base}"])
    else:
        _run(["checkout", "-b", name])


def get_default_branch() -> str:
    """Get the default branch name from the remote."""
    result = _run(["symbolic-ref", "refs/remotes/origin/HEAD"], check=False)
    if result.returncode == 0:
        return result.stdout.strip().replace("refs/remotes/origin/", "")
    return "main"


def commit_and_push(files: list[str], message: str, branch: str) -> str:
    """Stage files, commit, and push. Returns the commit SHA."""
    for f in files:
        _run(["add", f])

    _run(["commit", "-m", message])

    result = _run(["rev-parse", "HEAD"])
    sha = result.stdout.strip()

    _run(["push", "origin", branch])

    return sha


def has_changes() -> bool:
    """Check if there are staged or unstaged changes."""
    result = _run(["status", "--porcelain"], check=False)
    return bool(result.stdout.strip())
