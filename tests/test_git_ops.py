"""Tests for yaml_update.git_ops module."""

from unittest.mock import MagicMock, call, patch

import pytest

from yaml_update.git_ops import (
    GitError,
    commit_and_push,
    configure,
    create_branch,
    fetch,
    get_default_branch,
    has_changes,
)


def _make_result(returncode=0, stdout="", stderr=""):
    result = MagicMock()
    result.returncode = returncode
    result.stdout = stdout
    result.stderr = stderr
    return result


@patch("yaml_update.git_ops.subprocess.run")
class TestConfigure:
    def test_basic_configure(self, mock_run):
        mock_run.return_value = _make_result()
        configure("bot", "bot@test.com", "", "")
        calls = mock_run.call_args_list
        assert call(["git", "config", "user.name", "bot"], capture_output=True, text=True) in calls
        assert (
            call(["git", "config", "user.email", "bot@test.com"], capture_output=True, text=True)
            in calls
        )

    def test_configure_with_token(self, mock_run):
        mock_run.return_value = _make_result()
        configure("bot", "bot@test.com", "mytoken", "owner/repo")
        calls = mock_run.call_args_list
        # Check remote set-url was called with token in URL
        set_url_call = [c for c in calls if "set-url" in c[0][0]]
        assert len(set_url_call) == 1
        url_arg = set_url_call[0][0][0][-1]
        assert "x-access-token:mytoken@" in url_arg
        assert "owner/repo.git" in url_arg

    def test_configure_with_custom_server_url(self, mock_run):
        mock_run.return_value = _make_result()
        configure("bot", "bot@test.com", "tok", "owner/repo", "https://gh.enterprise.com")
        calls = mock_run.call_args_list
        set_url_call = [c for c in calls if "set-url" in c[0][0]]
        url_arg = set_url_call[0][0][0][-1]
        assert "gh.enterprise.com" in url_arg


@patch("yaml_update.git_ops.subprocess.run")
class TestFetch:
    def test_fetch_all(self, mock_run):
        mock_run.return_value = _make_result()
        fetch()
        mock_run.assert_called_with(["git", "fetch", "origin"], capture_output=True, text=True)

    def test_fetch_branch(self, mock_run):
        mock_run.return_value = _make_result()
        fetch("main")
        mock_run.assert_called_with(
            ["git", "fetch", "origin", "main"], capture_output=True, text=True
        )


@patch("yaml_update.git_ops.subprocess.run")
class TestCreateBranch:
    def test_create_branch_from_base(self, mock_run):
        mock_run.return_value = _make_result()
        create_branch("feature", "main")
        calls = mock_run.call_args_list
        assert call(["git", "fetch", "origin", "main"], capture_output=True, text=True) in calls
        assert (
            call(
                ["git", "checkout", "-b", "feature", "origin/main"],
                capture_output=True,
                text=True,
            )
            in calls
        )

    def test_create_branch_no_base(self, mock_run):
        mock_run.return_value = _make_result()
        create_branch("feature")
        mock_run.assert_called_with(
            ["git", "checkout", "-b", "feature"], capture_output=True, text=True
        )


@patch("yaml_update.git_ops.subprocess.run")
class TestCommitAndPush:
    def test_commit_and_push(self, mock_run):
        mock_run.return_value = _make_result(stdout="abc123\n")
        sha = commit_and_push(["a.yaml", "b.yaml"], "update files", "my-branch")
        assert sha == "abc123"
        calls = mock_run.call_args_list
        assert call(["git", "add", "a.yaml"], capture_output=True, text=True) in calls
        assert call(["git", "add", "b.yaml"], capture_output=True, text=True) in calls
        assert (
            call(
                ["git", "commit", "-m", "update files"],
                capture_output=True,
                text=True,
            )
            in calls
        )
        assert (
            call(
                ["git", "push", "origin", "my-branch"],
                capture_output=True,
                text=True,
            )
            in calls
        )

    def test_commit_failure_raises(self, mock_run):
        def side_effect(args, **kwargs):
            if "commit" in args:
                return _make_result(returncode=1, stderr="nothing to commit")
            return _make_result(stdout="abc123\n")

        mock_run.side_effect = side_effect
        with pytest.raises(GitError, match="nothing to commit"):
            commit_and_push(["a.yaml"], "msg", "branch")


@patch("yaml_update.git_ops.subprocess.run")
class TestGetDefaultBranch:
    def test_returns_branch_name(self, mock_run):
        mock_run.return_value = _make_result(stdout="refs/remotes/origin/main\n")
        assert get_default_branch() == "main"

    def test_returns_develop(self, mock_run):
        mock_run.return_value = _make_result(stdout="refs/remotes/origin/develop\n")
        assert get_default_branch() == "develop"

    def test_fallback_to_main(self, mock_run):
        mock_run.return_value = _make_result(returncode=1)
        assert get_default_branch() == "main"


@patch("yaml_update.git_ops.subprocess.run")
class TestHasChanges:
    def test_has_changes(self, mock_run):
        mock_run.return_value = _make_result(stdout=" M file.yaml\n")
        assert has_changes() is True

    def test_no_changes(self, mock_run):
        mock_run.return_value = _make_result(stdout="")
        assert has_changes() is False
