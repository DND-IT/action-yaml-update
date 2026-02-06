"""GitHub API client using PyGithub."""

from __future__ import annotations

from typing import TYPE_CHECKING, Any

from github import Auth, Github, GithubException

if TYPE_CHECKING:
    from github.PullRequest import PullRequest


class GitHubAPIError(Exception):
    """Raised when a GitHub API call fails."""


def _get_client(token: str, base_url: str) -> Github:
    """Create an authenticated GitHub client."""
    auth = Auth.Token(token) if token else None
    if base_url and base_url != "https://api.github.com":
        return Github(auth=auth, base_url=base_url)
    return Github(auth=auth)


def create_pull_request(
    api_url: str,
    token: str,
    owner: str,
    repo: str,
    title: str,
    body: str,
    head: str,
    base: str,
) -> dict[str, Any]:
    """Create a pull request. Returns a dict with number, html_url, node_id."""
    try:
        gh = _get_client(token, api_url)
        repository = gh.get_repo(f"{owner}/{repo}")
        pr: PullRequest = repository.create_pull(title=title, body=body, head=head, base=base)
        return {
            "number": pr.number,
            "html_url": pr.html_url,
            "node_id": pr.raw_data.get("node_id", ""),
        }
    except GithubException as e:
        raise GitHubAPIError(f"Failed to create PR: {e.data}") from e


def add_labels(
    api_url: str,
    token: str,
    owner: str,
    repo: str,
    issue_number: int,
    labels: list[str],
) -> None:
    """Add labels to an issue/PR."""
    try:
        gh = _get_client(token, api_url)
        repository = gh.get_repo(f"{owner}/{repo}")
        issue = repository.get_issue(issue_number)
        issue.add_to_labels(*labels)
    except GithubException as e:
        raise GitHubAPIError(f"Failed to add labels: {e.data}") from e


def request_reviewers(
    api_url: str,
    token: str,
    owner: str,
    repo: str,
    pr_number: int,
    reviewers: list[str],
) -> None:
    """Request reviewers for a pull request."""
    try:
        gh = _get_client(token, api_url)
        repository = gh.get_repo(f"{owner}/{repo}")
        pr = repository.get_pull(pr_number)
        pr.create_review_request(reviewers=reviewers)
    except GithubException as e:
        raise GitHubAPIError(f"Failed to request reviewers: {e.data}") from e


def enable_auto_merge(
    graphql_url: str,
    token: str,
    pr_node_id: str,
    merge_method: str = "SQUASH",
) -> None:
    """Enable auto-merge on a pull request using GraphQL.

    PyGithub doesn't have direct support for auto-merge, so we use the
    underlying requester to make a GraphQL call.
    """
    # Extract base URL from graphql_url for the client
    # graphql_url is like "https://api.github.com/graphql"
    base_url = graphql_url.rsplit("/graphql", 1)[0]

    try:
        gh = _get_client(token, base_url)

        query = """
        mutation EnableAutoMerge($prId: ID!, $mergeMethod: PullRequestMergeMethod!) {
          enablePullRequestAutoMerge(input: {
            pullRequestId: $prId,
            mergeMethod: $mergeMethod
          }) {
            pullRequest {
              autoMergeRequest {
                enabledAt
              }
            }
          }
        }
        """

        variables = {"prId": pr_node_id, "mergeMethod": merge_method}

        # Use the underlying requester for GraphQL
        _, data = gh._Github__requester.requestJsonAndCheck(
            "POST",
            "/graphql",
            input={"query": query, "variables": variables},
        )

        if "errors" in data:
            errors = data["errors"]
            msg = "; ".join(e.get("message", str(e)) for e in errors)
            raise GitHubAPIError(f"GraphQL error: {msg}")

    except GithubException as e:
        raise GitHubAPIError(f"Failed to enable auto-merge: {e.data}") from e
