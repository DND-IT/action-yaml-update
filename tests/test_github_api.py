"""Tests for yaml_update.github_api module."""

from unittest.mock import MagicMock, patch

import pytest
from github import GithubException

from yaml_update.github_api import (
    GitHubAPIError,
    add_labels,
    create_pull_request,
    enable_auto_merge,
    request_reviewers,
)

API_URL = "https://api.github.com"
GQL_URL = "https://api.github.com/graphql"
TOKEN = "ghp_test_token"


@patch("yaml_update.github_api.Github")
class TestCreatePullRequest:
    def test_creates_pr(self, mock_github_class):
        mock_gh = MagicMock()
        mock_github_class.return_value = mock_gh

        mock_repo = MagicMock()
        mock_gh.get_repo.return_value = mock_repo

        mock_pr = MagicMock()
        mock_pr.number = 42
        mock_pr.html_url = "https://github.com/owner/repo/pull/42"
        mock_pr.raw_data = {"node_id": "PR_node123"}
        mock_repo.create_pull.return_value = mock_pr

        result = create_pull_request(
            API_URL, TOKEN, "owner", "repo", "title", "body", "feature", "main"
        )

        assert result["number"] == 42
        assert result["html_url"] == "https://github.com/owner/repo/pull/42"
        assert result["node_id"] == "PR_node123"

        mock_gh.get_repo.assert_called_once_with("owner/repo")
        mock_repo.create_pull.assert_called_once_with(
            title="title", body="body", head="feature", base="main"
        )

    def test_api_error_raises(self, mock_github_class):
        mock_gh = MagicMock()
        mock_github_class.return_value = mock_gh
        mock_gh.get_repo.side_effect = GithubException(422, {"message": "Validation Failed"}, None)

        with pytest.raises(GitHubAPIError, match="Failed to create PR"):
            create_pull_request(API_URL, TOKEN, "owner", "repo", "t", "b", "h", "base")


@patch("yaml_update.github_api.Github")
class TestAddLabels:
    def test_adds_labels(self, mock_github_class):
        mock_gh = MagicMock()
        mock_github_class.return_value = mock_gh

        mock_repo = MagicMock()
        mock_gh.get_repo.return_value = mock_repo

        mock_issue = MagicMock()
        mock_repo.get_issue.return_value = mock_issue

        add_labels(API_URL, TOKEN, "owner", "repo", 42, ["bug", "enhancement"])

        mock_repo.get_issue.assert_called_once_with(42)
        mock_issue.add_to_labels.assert_called_once_with("bug", "enhancement")

    def test_api_error_raises(self, mock_github_class):
        mock_gh = MagicMock()
        mock_github_class.return_value = mock_gh
        mock_gh.get_repo.side_effect = GithubException(404, {"message": "Not Found"}, None)

        with pytest.raises(GitHubAPIError, match="Failed to add labels"):
            add_labels(API_URL, TOKEN, "owner", "repo", 42, ["bug"])


@patch("yaml_update.github_api.Github")
class TestRequestReviewers:
    def test_requests_reviewers(self, mock_github_class):
        mock_gh = MagicMock()
        mock_github_class.return_value = mock_gh

        mock_repo = MagicMock()
        mock_gh.get_repo.return_value = mock_repo

        mock_pr = MagicMock()
        mock_repo.get_pull.return_value = mock_pr

        request_reviewers(API_URL, TOKEN, "owner", "repo", 42, ["alice", "bob"])

        mock_repo.get_pull.assert_called_once_with(42)
        mock_pr.create_review_request.assert_called_once_with(reviewers=["alice", "bob"])

    def test_api_error_raises(self, mock_github_class):
        mock_gh = MagicMock()
        mock_github_class.return_value = mock_gh
        mock_gh.get_repo.side_effect = GithubException(404, {"message": "Not Found"}, None)

        with pytest.raises(GitHubAPIError, match="Failed to request reviewers"):
            request_reviewers(API_URL, TOKEN, "owner", "repo", 42, ["alice"])


@patch("yaml_update.github_api.Github")
class TestEnableAutoMerge:
    def test_enables_auto_merge(self, mock_github_class):
        mock_gh = MagicMock()
        mock_github_class.return_value = mock_gh

        mock_requester = MagicMock()
        mock_gh._Github__requester = mock_requester
        mock_requester.requestJsonAndCheck.return_value = (
            {},
            {
                "data": {
                    "enablePullRequestAutoMerge": {
                        "pullRequest": {"autoMergeRequest": {"enabledAt": "2024-01-01"}}
                    }
                }
            },
        )

        enable_auto_merge(GQL_URL, TOKEN, "PR_abc123", "SQUASH")

        mock_requester.requestJsonAndCheck.assert_called_once()
        call_args = mock_requester.requestJsonAndCheck.call_args
        assert call_args[0][0] == "POST"
        assert call_args[0][1] == "/graphql"
        assert "enablePullRequestAutoMerge" in call_args[1]["input"]["query"]
        assert call_args[1]["input"]["variables"]["prId"] == "PR_abc123"
        assert call_args[1]["input"]["variables"]["mergeMethod"] == "SQUASH"

    def test_graphql_error_raises(self, mock_github_class):
        mock_gh = MagicMock()
        mock_github_class.return_value = mock_gh

        mock_requester = MagicMock()
        mock_gh._Github__requester = mock_requester
        mock_requester.requestJsonAndCheck.return_value = (
            {},
            {"errors": [{"message": "Could not enable auto-merge"}]},
        )

        with pytest.raises(GitHubAPIError, match="Could not enable auto-merge"):
            enable_auto_merge(GQL_URL, TOKEN, "PR_abc123", "SQUASH")

    def test_github_exception_raises(self, mock_github_class):
        mock_gh = MagicMock()
        mock_github_class.return_value = mock_gh

        mock_requester = MagicMock()
        mock_gh._Github__requester = mock_requester
        mock_requester.requestJsonAndCheck.side_effect = GithubException(
            401, {"message": "Bad credentials"}, None
        )

        with pytest.raises(GitHubAPIError, match="Failed to enable auto-merge"):
            enable_auto_merge(GQL_URL, TOKEN, "PR_abc123", "SQUASH")


@patch("yaml_update.github_api.Github")
class TestCustomBaseUrl:
    def test_enterprise_base_url(self, mock_github_class):
        mock_gh = MagicMock()
        mock_github_class.return_value = mock_gh

        mock_repo = MagicMock()
        mock_gh.get_repo.return_value = mock_repo

        mock_pr = MagicMock()
        mock_pr.number = 1
        mock_pr.html_url = "https://gh.enterprise.com/owner/repo/pull/1"
        mock_pr.raw_data = {"node_id": "PR_1"}
        mock_repo.create_pull.return_value = mock_pr

        create_pull_request(
            "https://gh.enterprise.com/api/v3",
            TOKEN,
            "owner",
            "repo",
            "title",
            "body",
            "head",
            "base",
        )

        # Verify Github was called with base_url for enterprise
        call_kwargs = mock_github_class.call_args[1]
        assert call_kwargs["base_url"] == "https://gh.enterprise.com/api/v3"
