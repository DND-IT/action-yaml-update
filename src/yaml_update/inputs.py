"""Parse and validate GitHub Action INPUT_* environment variables."""

from __future__ import annotations

import os
from dataclasses import dataclass, field


class InputError(Exception):
    """Raised when action inputs are invalid."""


def _get(name: str, default: str | None = None, required: bool = False) -> str:
    """Read an action input from the environment (INPUT_<NAME> in uppercase)."""
    env_key = f"INPUT_{name.upper().replace('-', '_')}"
    value = os.environ.get(env_key, "")
    if value:
        return value
    if required:
        raise InputError(f"Input '{name}' is required but was not provided")
    return default if default is not None else ""


def _parse_bool(value: str) -> bool:
    return value.strip().lower() in ("true", "yes", "1")


def _parse_lines(value: str) -> list[str]:
    """Split a newline-separated string into a list, stripping empty lines."""
    if not value.strip():
        return []
    return [line.strip() for line in value.strip().splitlines() if line.strip()]


def _parse_csv(value: str) -> list[str]:
    """Split a comma-separated string into a list, stripping whitespace."""
    if not value.strip():
        return []
    return [item.strip() for item in value.split(",") if item.strip()]


@dataclass(frozen=True)
class ActionInputs:
    files: list[str]
    mode: str
    keys: list[str]
    values: list[str]
    image_name: str
    image_tag: str
    create_pr: bool
    target_branch: str
    pr_branch: str
    pr_title: str
    pr_body: str
    pr_labels: list[str]
    pr_reviewers: list[str]
    commit_message: str
    token: str
    auto_merge: bool
    merge_method: str
    dry_run: bool
    git_user_name: str
    git_user_email: str
    github_repository: str = field(default="")
    github_api_url: str = field(default="")
    github_graphql_url: str = field(default="")
    github_server_url: str = field(default="")


def parse_inputs() -> ActionInputs:
    """Parse and validate all action inputs from the environment."""
    files = _parse_lines(_get("files", required=True))
    if not files:
        raise InputError("Input 'files' must contain at least one file path")

    mode = _get("mode", default="key").strip().lower()
    if mode not in ("key", "image"):
        raise InputError(f"Input 'mode' must be 'key' or 'image', got '{mode}'")

    keys = _parse_lines(_get("keys"))
    values = _parse_lines(_get("values"))
    image_name = _get("image_name")
    image_tag = _get("image_tag")

    if mode == "key":
        if not keys:
            raise InputError("Input 'keys' is required when mode is 'key'")
        if not values:
            raise InputError("Input 'values' is required when mode is 'key'")
        if len(keys) != len(values):
            raise InputError(
                f"'keys' and 'values' must have the same number of entries "
                f"(got {len(keys)} keys and {len(values)} values)"
            )
    elif mode == "image":
        if not image_name:
            raise InputError("Input 'image_name' is required when mode is 'image'")
        if not image_tag:
            raise InputError("Input 'image_tag' is required when mode is 'image'")

    merge_method = _get("merge_method", default="SQUASH").strip().upper()
    if merge_method not in ("MERGE", "SQUASH", "REBASE"):
        raise InputError(
            f"Input 'merge_method' must be MERGE, SQUASH, or REBASE, got '{merge_method}'"
        )

    return ActionInputs(
        files=files,
        mode=mode,
        keys=keys,
        values=values,
        image_name=image_name,
        image_tag=image_tag,
        create_pr=_parse_bool(_get("create_pr", default="true")),
        target_branch=_get("target_branch"),
        pr_branch=_get("pr_branch"),
        pr_title=_get("pr_title", default="chore: update YAML values"),
        pr_body=_get("pr_body"),
        pr_labels=_parse_csv(_get("pr_labels")),
        pr_reviewers=_parse_csv(_get("pr_reviewers")),
        commit_message=_get("commit_message", default="chore: update YAML values"),
        token=_get("token"),
        auto_merge=_parse_bool(_get("auto_merge", default="false")),
        merge_method=merge_method,
        dry_run=_parse_bool(_get("dry_run", default="false")),
        git_user_name=_get("git_user_name", default="github-actions[bot]"),
        git_user_email=_get(
            "git_user_email",
            default="41898282+github-actions[bot]@users.noreply.github.com",
        ),
        github_repository=os.environ.get("GITHUB_REPOSITORY", ""),
        github_api_url=os.environ.get("GITHUB_API_URL", "https://api.github.com"),
        github_graphql_url=os.environ.get("GITHUB_GRAPHQL_URL", "https://api.github.com/graphql"),
        github_server_url=os.environ.get("GITHUB_SERVER_URL", "https://github.com"),
    )
