"""Tests for yaml_update.__main__ module."""

from pathlib import Path
from unittest.mock import patch

from yaml_update.__main__ import main

FIXTURES_DIR = Path(__file__).parent / "fixtures"


def _set_key_mode_env(monkeypatch, files, keys, values, **extras):
    """Helper to set up key mode environment variables."""
    monkeypatch.setenv("INPUT_FILES", files)
    monkeypatch.setenv("INPUT_MODE", "key")
    monkeypatch.setenv("INPUT_KEYS", keys)
    monkeypatch.setenv("INPUT_VALUES", values)
    monkeypatch.setenv("INPUT_DRY_RUN", extras.get("dry_run", "true"))
    monkeypatch.setenv("INPUT_CREATE_PR", extras.get("create_pr", "false"))
    for k, v in extras.items():
        if k not in ("dry_run", "create_pr"):
            monkeypatch.setenv(f"INPUT_{k.upper()}", v)


def _set_image_mode_env(monkeypatch, files, image_name, image_tag, **extras):
    """Helper to set up image mode environment variables."""
    monkeypatch.setenv("INPUT_FILES", files)
    monkeypatch.setenv("INPUT_MODE", "image")
    monkeypatch.setenv("INPUT_IMAGE_NAME", image_name)
    monkeypatch.setenv("INPUT_IMAGE_TAG", image_tag)
    monkeypatch.setenv("INPUT_DRY_RUN", extras.get("dry_run", "true"))
    monkeypatch.setenv("INPUT_CREATE_PR", extras.get("create_pr", "false"))
    for k, v in extras.items():
        if k not in ("dry_run", "create_pr"):
            monkeypatch.setenv(f"INPUT_{k.upper()}", v)


class TestMainInvalidInputs:
    def test_missing_files_returns_1(self, monkeypatch):
        monkeypatch.delenv("INPUT_FILES", raising=False)
        assert main() == 1

    def test_invalid_mode_returns_1(self, monkeypatch):
        monkeypatch.setenv("INPUT_FILES", "x.yaml")
        monkeypatch.setenv("INPUT_MODE", "bad")
        assert main() == 1


class TestMainDryRunKeyMode:
    def test_dry_run_with_changes(self, monkeypatch, tmp_path):
        yaml_file = tmp_path / "values.yaml"
        yaml_file.write_text("app:\n  version: v1.0.0\n")
        output_file = tmp_path / "github_output"
        output_file.touch()
        monkeypatch.setenv("GITHUB_OUTPUT", str(output_file))

        _set_key_mode_env(monkeypatch, str(yaml_file), "app.version", "v2.0.0")

        result = main()
        assert result == 0

        outputs = output_file.read_text()
        assert "changed=true" in outputs

        # File should NOT be modified in dry run
        assert "v1.0.0" in yaml_file.read_text()

    def test_dry_run_no_changes(self, monkeypatch, tmp_path):
        yaml_file = tmp_path / "values.yaml"
        yaml_file.write_text("app:\n  version: v1.0.0\n")
        output_file = tmp_path / "github_output"
        output_file.touch()
        monkeypatch.setenv("GITHUB_OUTPUT", str(output_file))

        _set_key_mode_env(monkeypatch, str(yaml_file), "app.version", "v1.0.0")

        result = main()
        assert result == 0
        outputs = output_file.read_text()
        assert "changed=false" in outputs


class TestMainDryRunImageMode:
    def test_dry_run_image_mode(self, monkeypatch, tmp_path):
        yaml_file = tmp_path / "values.yaml"
        yaml_file.write_text("image:\n  repository: ghcr.io/myorg/webapp\n  tag: v1.0.0\n")
        output_file = tmp_path / "github_output"
        output_file.touch()
        monkeypatch.setenv("GITHUB_OUTPUT", str(output_file))

        _set_image_mode_env(monkeypatch, str(yaml_file), "webapp", "v2.0.0")

        result = main()
        assert result == 0
        outputs = output_file.read_text()
        assert "changed=true" in outputs


class TestMainFileNotFound:
    def test_missing_file_returns_1(self, monkeypatch):
        _set_key_mode_env(monkeypatch, "/nonexistent/file.yaml", "key", "value")
        assert main() == 1


class TestMainKeyError:
    def test_invalid_key_path_returns_1(self, monkeypatch, tmp_path):
        yaml_file = tmp_path / "values.yaml"
        yaml_file.write_text("app:\n  version: v1.0.0\n")
        _set_key_mode_env(monkeypatch, str(yaml_file), "nonexistent.path", "value")
        assert main() == 1


class TestMainEmptyYaml:
    def test_empty_yaml_skipped(self, monkeypatch, tmp_path):
        yaml_file = tmp_path / "empty.yaml"
        yaml_file.write_text("")
        output_file = tmp_path / "github_output"
        output_file.touch()
        monkeypatch.setenv("GITHUB_OUTPUT", str(output_file))

        _set_key_mode_env(monkeypatch, str(yaml_file), "key", "value")
        result = main()
        assert result == 0


class TestMainWithGitOps:
    @patch("yaml_update.__main__.git_ops")
    def test_direct_commit_no_pr(self, mock_git, monkeypatch, tmp_path):
        yaml_file = tmp_path / "values.yaml"
        yaml_file.write_text("app:\n  version: v1.0.0\n")
        output_file = tmp_path / "github_output"
        output_file.touch()
        monkeypatch.setenv("GITHUB_OUTPUT", str(output_file))
        monkeypatch.setenv("GITHUB_REPOSITORY", "owner/repo")

        _set_key_mode_env(
            monkeypatch,
            str(yaml_file),
            "app.version",
            "v2.0.0",
            dry_run="false",
            create_pr="false",
        )
        monkeypatch.setenv("INPUT_TOKEN", "ghp_test")

        mock_git.get_default_branch.return_value = "main"
        mock_git.commit_and_push.return_value = "abc123def"

        result = main()
        assert result == 0

        mock_git.configure.assert_called_once()
        mock_git.commit_and_push.assert_called_once()
        outputs = output_file.read_text()
        assert "commit_sha=abc123def" in outputs
        assert "pr_number=" in outputs

    @patch("yaml_update.__main__.github_api")
    @patch("yaml_update.__main__.git_ops")
    def test_create_pr(self, mock_git, mock_api, monkeypatch, tmp_path):
        yaml_file = tmp_path / "values.yaml"
        yaml_file.write_text("app:\n  version: v1.0.0\n")
        output_file = tmp_path / "github_output"
        output_file.touch()
        monkeypatch.setenv("GITHUB_OUTPUT", str(output_file))
        monkeypatch.setenv("GITHUB_REPOSITORY", "owner/repo")

        _set_key_mode_env(
            monkeypatch,
            str(yaml_file),
            "app.version",
            "v2.0.0",
            dry_run="false",
            create_pr="true",
        )
        monkeypatch.setenv("INPUT_TOKEN", "ghp_test")
        monkeypatch.setenv("INPUT_PR_TITLE", "test PR")
        monkeypatch.setenv("INPUT_PR_LABELS", "automated")
        monkeypatch.setenv("INPUT_PR_REVIEWERS", "alice")

        mock_git.get_default_branch.return_value = "main"
        mock_git.commit_and_push.return_value = "abc123"
        mock_api.create_pull_request.return_value = {
            "number": 42,
            "html_url": "https://github.com/owner/repo/pull/42",
            "node_id": "PR_1",
        }

        result = main()
        assert result == 0

        mock_api.create_pull_request.assert_called_once()
        mock_api.add_labels.assert_called_once()
        mock_api.request_reviewers.assert_called_once()
        outputs = output_file.read_text()
        assert "pr_number=42" in outputs

    @patch("yaml_update.__main__.github_api")
    @patch("yaml_update.__main__.git_ops")
    def test_create_pr_with_auto_merge(self, mock_git, mock_api, monkeypatch, tmp_path):
        yaml_file = tmp_path / "values.yaml"
        yaml_file.write_text("app:\n  version: v1.0.0\n")
        output_file = tmp_path / "github_output"
        output_file.touch()
        monkeypatch.setenv("GITHUB_OUTPUT", str(output_file))
        monkeypatch.setenv("GITHUB_REPOSITORY", "owner/repo")

        _set_key_mode_env(
            monkeypatch,
            str(yaml_file),
            "app.version",
            "v2.0.0",
            dry_run="false",
            create_pr="true",
        )
        monkeypatch.setenv("INPUT_TOKEN", "ghp_test")
        monkeypatch.setenv("INPUT_AUTO_MERGE", "true")
        monkeypatch.setenv("INPUT_MERGE_METHOD", "SQUASH")

        mock_git.get_default_branch.return_value = "main"
        mock_git.commit_and_push.return_value = "abc123"
        mock_api.create_pull_request.return_value = {
            "number": 10,
            "html_url": "https://github.com/owner/repo/pull/10",
            "node_id": "PR_node",
        }

        result = main()
        assert result == 0
        mock_api.enable_auto_merge.assert_called_once()

    @patch("yaml_update.__main__.github_api")
    @patch("yaml_update.__main__.git_ops")
    def test_github_api_error_returns_1(self, mock_git, mock_api, monkeypatch, tmp_path):
        from yaml_update.github_api import GitHubAPIError

        yaml_file = tmp_path / "values.yaml"
        yaml_file.write_text("app:\n  version: v1.0.0\n")
        output_file = tmp_path / "github_output"
        output_file.touch()
        monkeypatch.setenv("GITHUB_OUTPUT", str(output_file))
        monkeypatch.setenv("GITHUB_REPOSITORY", "owner/repo")

        _set_key_mode_env(
            monkeypatch,
            str(yaml_file),
            "app.version",
            "v2.0.0",
            dry_run="false",
            create_pr="true",
        )
        monkeypatch.setenv("INPUT_TOKEN", "ghp_test")

        mock_git.get_default_branch.return_value = "main"
        mock_git.commit_and_push.return_value = "abc123"
        # Set the real exception class on the mock so `except` works
        mock_api.GitHubAPIError = GitHubAPIError
        mock_api.create_pull_request.side_effect = GitHubAPIError("API error")

        result = main()
        assert result == 1


class TestMainMultipleFiles:
    def test_multiple_files_dry_run(self, monkeypatch, tmp_path):
        f1 = tmp_path / "a.yaml"
        f1.write_text("key: old1\n")
        f2 = tmp_path / "b.yaml"
        f2.write_text("key: old2\n")
        output_file = tmp_path / "github_output"
        output_file.touch()
        monkeypatch.setenv("GITHUB_OUTPUT", str(output_file))

        _set_key_mode_env(monkeypatch, f"{f1}\n{f2}", "key", "new")

        result = main()
        assert result == 0
        outputs = output_file.read_text()
        assert "changed=true" in outputs
