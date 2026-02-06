"""Main entrypoint for the yaml-update GitHub Action."""

from __future__ import annotations

import hashlib
import sys
import time
from pathlib import Path

from yaml_update import git_ops, github_api, outputs
from yaml_update.inputs import InputError, parse_inputs
from yaml_update.updater import diff_yaml, dump_yaml, load_yaml, update_image_tags, update_keys


def _generate_branch_name(inputs) -> str:
    """Generate a unique branch name based on the inputs."""
    seed = f"{inputs.files}{inputs.keys}{inputs.values}{inputs.image_name}{inputs.image_tag}"
    short_hash = hashlib.sha256(seed.encode()).hexdigest()[:8]
    return f"yaml-update/{short_hash}-{int(time.time())}"


def main() -> int:
    # --- Parse inputs ---
    try:
        inputs = parse_inputs()
    except InputError as e:
        outputs.log_error(str(e))
        return 1

    outputs.log_info(f"Mode: {inputs.mode}")
    outputs.log_info(f"Files: {inputs.files}")
    if inputs.dry_run:
        outputs.log_info("Dry run mode enabled — no changes will be persisted")

    # --- Process each file ---
    all_changes: list[dict] = []
    changed_files: list[str] = []
    all_diffs: list[str] = []

    for file_path_str in inputs.files:
        file_path = Path(file_path_str)
        if not file_path.exists():
            outputs.log_error(f"File not found: {file_path}")
            return 1

        outputs.log_group(f"Processing {file_path}")

        original_content = file_path.read_text()
        data = load_yaml(file_path)

        if data is None:
            outputs.log_warning(f"Skipping empty YAML file: {file_path}")
            outputs.log_endgroup()
            continue

        try:
            if inputs.mode == "key":
                changes = update_keys(data, inputs.keys, inputs.values)
            else:
                changes = update_image_tags(data, inputs.image_name, inputs.image_tag)
        except KeyError as e:
            outputs.log_error(f"Update failed for {file_path}: {e}")
            outputs.log_endgroup()
            return 1

        if changes:
            for c in changes:
                outputs.log_info(f"  {c['key']}: {c['old']} -> {c['new']}")
            all_changes.extend(changes)
            changed_files.append(file_path_str)

            file_diff = diff_yaml(file_path, original_content, data)
            if file_diff:
                all_diffs.append(file_diff)

            if not inputs.dry_run:
                dump_yaml(data, file_path)
        else:
            outputs.log_info(f"  No changes needed for {file_path}")

        outputs.log_endgroup()

    # --- Write outputs ---
    has_changes = len(changed_files) > 0
    outputs.set_output("changed", str(has_changes).lower())
    outputs.set_output("changed_files", "\n".join(changed_files))
    diff_text = "\n".join(all_diffs)
    outputs.set_output("diff", diff_text)

    if not has_changes:
        outputs.log_info("No changes detected across any files")
        outputs.set_output("pr_number", "")
        outputs.set_output("pr_url", "")
        outputs.set_output("commit_sha", "")
        return 0

    if inputs.dry_run:
        outputs.log_info("Dry run complete. Changes that would be made:")
        if diff_text:
            outputs.log_info(diff_text)
        outputs.set_output("pr_number", "")
        outputs.set_output("pr_url", "")
        outputs.set_output("commit_sha", "")
        return 0

    # --- Git operations ---
    outputs.log_group("Git operations")

    owner, repo = "", ""
    if inputs.github_repository:
        parts = inputs.github_repository.split("/", 1)
        if len(parts) == 2:
            owner, repo = parts

    git_ops.configure(
        inputs.git_user_name,
        inputs.git_user_email,
        inputs.token,
        inputs.github_repository,
        inputs.github_server_url,
    )

    target_branch = inputs.target_branch or git_ops.get_default_branch()

    if inputs.create_pr:
        pr_branch = inputs.pr_branch or _generate_branch_name(inputs)
        git_ops.create_branch(pr_branch, target_branch)
        commit_branch = pr_branch
    else:
        commit_branch = target_branch

    sha = git_ops.commit_and_push(changed_files, inputs.commit_message, commit_branch)
    outputs.set_output("commit_sha", sha)
    outputs.log_info(f"Committed and pushed: {sha}")
    outputs.log_endgroup()

    # --- Create PR ---
    if inputs.create_pr:
        outputs.log_group("Pull request")

        pr_body = inputs.pr_body
        if not pr_body:
            change_lines = []
            for c in all_changes:
                change_lines.append(f"- `{c['key']}`: `{c['old']}` → `{c['new']}`")
            pr_body = "## Changes\n\n" + "\n".join(change_lines)

        try:
            pr_data = github_api.create_pull_request(
                inputs.github_api_url,
                inputs.token,
                owner,
                repo,
                inputs.pr_title,
                pr_body,
                pr_branch,
                target_branch,
            )

            pr_number = pr_data["number"]
            pr_url = pr_data["html_url"]
            pr_node_id = pr_data.get("node_id", "")

            outputs.set_output("pr_number", str(pr_number))
            outputs.set_output("pr_url", pr_url)
            outputs.log_info(f"Created PR #{pr_number}: {pr_url}")

            if inputs.pr_labels:
                github_api.add_labels(
                    inputs.github_api_url, inputs.token, owner, repo, pr_number, inputs.pr_labels
                )
                outputs.log_info(f"Added labels: {inputs.pr_labels}")

            if inputs.pr_reviewers:
                github_api.request_reviewers(
                    inputs.github_api_url,
                    inputs.token,
                    owner,
                    repo,
                    pr_number,
                    inputs.pr_reviewers,
                )
                outputs.log_info(f"Requested reviewers: {inputs.pr_reviewers}")

            if inputs.auto_merge and pr_node_id:
                github_api.enable_auto_merge(
                    inputs.github_graphql_url,
                    inputs.token,
                    pr_node_id,
                    inputs.merge_method,
                )
                outputs.log_info(f"Enabled auto-merge ({inputs.merge_method})")

        except github_api.GitHubAPIError as e:
            outputs.log_error(f"GitHub API error: {e}")
            outputs.log_endgroup()
            return 1

        outputs.log_endgroup()
    else:
        outputs.set_output("pr_number", "")
        outputs.set_output("pr_url", "")

    outputs.log_info("Done!")
    return 0


if __name__ == "__main__":
    sys.exit(main())
