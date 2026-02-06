"""Tests for yaml_update.inputs module."""

import pytest

from yaml_update.inputs import InputError, parse_inputs


@pytest.fixture
def key_mode_env(monkeypatch):
    """Set up minimal valid environment for key mode."""
    monkeypatch.setenv("INPUT_FILES", "values.yaml")
    monkeypatch.setenv("INPUT_MODE", "key")
    monkeypatch.setenv("INPUT_KEYS", "app.image.tag")
    monkeypatch.setenv("INPUT_VALUES", "v1.2.3")


@pytest.fixture
def image_mode_env(monkeypatch):
    """Set up minimal valid environment for image mode."""
    monkeypatch.setenv("INPUT_FILES", "values.yaml")
    monkeypatch.setenv("INPUT_MODE", "image")
    monkeypatch.setenv("INPUT_IMAGE_NAME", "myapp")
    monkeypatch.setenv("INPUT_IMAGE_TAG", "v2.0.0")


class TestParseInputsKeyMode:
    def test_minimal_valid(self, key_mode_env):
        inputs = parse_inputs()
        assert inputs.files == ["values.yaml"]
        assert inputs.mode == "key"
        assert inputs.keys == ["app.image.tag"]
        assert inputs.values == ["v1.2.3"]

    def test_multiple_keys_values(self, monkeypatch):
        monkeypatch.setenv("INPUT_FILES", "a.yaml\nb.yaml")
        monkeypatch.setenv("INPUT_MODE", "key")
        monkeypatch.setenv("INPUT_KEYS", "key1\nkey2\nkey3")
        monkeypatch.setenv("INPUT_VALUES", "val1\nval2\nval3")
        inputs = parse_inputs()
        assert inputs.files == ["a.yaml", "b.yaml"]
        assert inputs.keys == ["key1", "key2", "key3"]
        assert inputs.values == ["val1", "val2", "val3"]

    def test_missing_keys_raises(self, monkeypatch):
        monkeypatch.setenv("INPUT_FILES", "values.yaml")
        monkeypatch.setenv("INPUT_MODE", "key")
        monkeypatch.setenv("INPUT_VALUES", "v1")
        with pytest.raises(InputError, match="'keys' is required"):
            parse_inputs()

    def test_missing_values_raises(self, monkeypatch):
        monkeypatch.setenv("INPUT_FILES", "values.yaml")
        monkeypatch.setenv("INPUT_MODE", "key")
        monkeypatch.setenv("INPUT_KEYS", "key1")
        with pytest.raises(InputError, match="'values' is required"):
            parse_inputs()

    def test_mismatched_keys_values_raises(self, monkeypatch):
        monkeypatch.setenv("INPUT_FILES", "values.yaml")
        monkeypatch.setenv("INPUT_MODE", "key")
        monkeypatch.setenv("INPUT_KEYS", "key1\nkey2")
        monkeypatch.setenv("INPUT_VALUES", "val1")
        with pytest.raises(InputError, match="same number of entries"):
            parse_inputs()


class TestParseInputsImageMode:
    def test_minimal_valid(self, image_mode_env):
        inputs = parse_inputs()
        assert inputs.mode == "image"
        assert inputs.image_name == "myapp"
        assert inputs.image_tag == "v2.0.0"

    def test_missing_image_name_raises(self, monkeypatch):
        monkeypatch.setenv("INPUT_FILES", "values.yaml")
        monkeypatch.setenv("INPUT_MODE", "image")
        monkeypatch.setenv("INPUT_IMAGE_TAG", "v1")
        with pytest.raises(InputError, match="'image_name' is required"):
            parse_inputs()

    def test_missing_image_tag_raises(self, monkeypatch):
        monkeypatch.setenv("INPUT_FILES", "values.yaml")
        monkeypatch.setenv("INPUT_MODE", "image")
        monkeypatch.setenv("INPUT_IMAGE_NAME", "myapp")
        with pytest.raises(InputError, match="'image_tag' is required"):
            parse_inputs()


class TestParseInputsValidation:
    def test_missing_files_raises(self, monkeypatch):
        monkeypatch.delenv("INPUT_FILES", raising=False)
        with pytest.raises(InputError, match="'files' is required"):
            parse_inputs()

    def test_empty_files_raises(self, monkeypatch):
        monkeypatch.setenv("INPUT_FILES", "  \n  ")
        with pytest.raises(InputError, match="at least one file path"):
            parse_inputs()

    def test_invalid_mode_raises(self, monkeypatch):
        monkeypatch.setenv("INPUT_FILES", "values.yaml")
        monkeypatch.setenv("INPUT_MODE", "invalid")
        with pytest.raises(InputError, match="must be 'key' or 'image'"):
            parse_inputs()

    def test_invalid_merge_method_raises(self, monkeypatch):
        monkeypatch.setenv("INPUT_FILES", "values.yaml")
        monkeypatch.setenv("INPUT_KEYS", "k")
        monkeypatch.setenv("INPUT_VALUES", "v")
        monkeypatch.setenv("INPUT_MERGE_METHOD", "INVALID")
        with pytest.raises(InputError, match="MERGE, SQUASH, or REBASE"):
            parse_inputs()


class TestDefaults:
    def test_default_values(self, key_mode_env):
        inputs = parse_inputs()
        assert inputs.create_pr is True
        assert inputs.pr_title == "chore: update YAML values"
        assert inputs.commit_message == "chore: update YAML values"
        assert inputs.auto_merge is False
        assert inputs.merge_method == "SQUASH"
        assert inputs.dry_run is False
        assert inputs.git_user_name == "github-actions[bot]"
        assert "noreply.github.com" in inputs.git_user_email

    def test_default_mode_is_key(self, monkeypatch):
        monkeypatch.setenv("INPUT_FILES", "values.yaml")
        monkeypatch.setenv("INPUT_KEYS", "k")
        monkeypatch.setenv("INPUT_VALUES", "v")
        # No INPUT_MODE set
        inputs = parse_inputs()
        assert inputs.mode == "key"


class TestBooleanParsing:
    @pytest.mark.parametrize("value", ["true", "True", "TRUE", "yes", "1"])
    def test_truthy_values(self, key_mode_env, monkeypatch, value):
        monkeypatch.setenv("INPUT_DRY_RUN", value)
        inputs = parse_inputs()
        assert inputs.dry_run is True

    @pytest.mark.parametrize("value", ["false", "False", "FALSE", "no", "0", ""])
    def test_falsy_values(self, key_mode_env, monkeypatch, value):
        monkeypatch.setenv("INPUT_DRY_RUN", value)
        inputs = parse_inputs()
        assert inputs.dry_run is False


class TestCSVAndLinesParsing:
    def test_pr_labels(self, key_mode_env, monkeypatch):
        monkeypatch.setenv("INPUT_PR_LABELS", "bug, enhancement, help wanted")
        inputs = parse_inputs()
        assert inputs.pr_labels == ["bug", "enhancement", "help wanted"]

    def test_pr_reviewers(self, key_mode_env, monkeypatch):
        monkeypatch.setenv("INPUT_PR_REVIEWERS", "alice,bob")
        inputs = parse_inputs()
        assert inputs.pr_reviewers == ["alice", "bob"]

    def test_empty_labels(self, key_mode_env):
        inputs = parse_inputs()
        assert inputs.pr_labels == []

    def test_files_with_blank_lines(self, monkeypatch):
        monkeypatch.setenv("INPUT_FILES", "a.yaml\n\n  \nb.yaml\n")
        monkeypatch.setenv("INPUT_KEYS", "k")
        monkeypatch.setenv("INPUT_VALUES", "v")
        inputs = parse_inputs()
        assert inputs.files == ["a.yaml", "b.yaml"]
