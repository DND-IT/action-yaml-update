"""Tests for yaml_update.updater module."""

from io import StringIO
from pathlib import Path

import pytest
from ruamel.yaml import YAML

from yaml_update.updater import (
    diff_yaml,
    dump_yaml,
    load_yaml,
    update_image_tags,
    update_keys,
)

FIXTURES_DIR = Path(__file__).parent / "fixtures"

yaml = YAML()
yaml.preserve_quotes = True


def _load_str(content: str):
    """Load YAML from string."""
    return yaml.load(content)


def _dump_str(data) -> str:
    """Dump YAML to string."""
    stream = StringIO()
    yaml.dump(data, stream)
    return stream.getvalue()


class TestUpdateKeys:
    def test_simple_key_update(self):
        data = _load_str("app:\n  version: v1.0.0\n")
        changes = update_keys(data, ["app.version"], ["v2.0.0"])
        assert len(changes) == 1
        assert changes[0]["old"] == "v1.0.0"
        assert changes[0]["new"] == "v2.0.0"
        assert data["app"]["version"] == "v2.0.0"

    def test_nested_key_update(self):
        data = _load_str("a:\n  b:\n    c: old\n")
        changes = update_keys(data, ["a.b.c"], ["new"])
        assert len(changes) == 1
        assert data["a"]["b"]["c"] == "new"

    def test_multiple_keys(self):
        data = _load_str("x: 1\ny: 2\n")
        changes = update_keys(data, ["x", "y"], ["10", "20"])
        assert len(changes) == 2
        assert data["x"] == 10
        assert data["y"] == 20

    def test_list_index(self):
        data = _load_str("items:\n  - name: a\n    value: old\n")
        changes = update_keys(data, ["items.0.value"], ["new"])
        assert len(changes) == 1
        assert data["items"][0]["value"] == "new"

    def test_no_change_when_same_value(self):
        data = _load_str("key: same\n")
        changes = update_keys(data, ["key"], ["same"])
        assert len(changes) == 0

    def test_key_not_found_raises(self):
        data = _load_str("key: value\n")
        with pytest.raises(KeyError, match="not found"):
            update_keys(data, ["missing.path"], ["val"])

    def test_invalid_list_index_raises(self):
        data = _load_str("items:\n  - a\n")
        with pytest.raises(KeyError, match="integer index"):
            update_keys(data, ["items.notanumber"], ["val"])


class TestTypeCoercion:
    def test_int_stays_int(self):
        data = _load_str("replicas: 3\n")
        update_keys(data, ["replicas"], ["5"])
        assert data["replicas"] == 5
        assert isinstance(data["replicas"], int)

    def test_bool_stays_bool(self):
        data = _load_str("enabled: true\n")
        update_keys(data, ["enabled"], ["false"])
        assert data["enabled"] is False

    def test_float_stays_float(self):
        data = _load_str("ratio: 1.5\n")
        update_keys(data, ["ratio"], ["2.5"])
        assert data["ratio"] == 2.5
        assert isinstance(data["ratio"], float)

    def test_string_stays_string(self):
        data = _load_str("name: hello\n")
        update_keys(data, ["name"], ["world"])
        assert data["name"] == "world"
        assert isinstance(data["name"], str)

    def test_int_with_non_numeric_becomes_string(self):
        data = _load_str("port: 8080\n")
        update_keys(data, ["port"], ["not-a-number"])
        assert data["port"] == "not-a-number"


class TestUpdateImageTags:
    def test_helm_style_match(self):
        data = _load_str("image:\n  repository: ghcr.io/myorg/webapp\n  tag: v1.0.0\n")
        changes = update_image_tags(data, "webapp", "v2.0.0")
        assert len(changes) == 1
        assert changes[0]["old"] == "v1.0.0"
        assert data["image"]["tag"] == "v2.0.0"

    def test_helm_style_no_match(self):
        data = _load_str("image:\n  repository: ghcr.io/myorg/other\n  tag: v1.0.0\n")
        changes = update_image_tags(data, "webapp", "v2.0.0")
        assert len(changes) == 0
        assert data["image"]["tag"] == "v1.0.0"

    def test_kustomize_style_match(self):
        data = _load_str("images:\n  - name: ghcr.io/myorg/webapp\n    newTag: v1.0.0\n")
        changes = update_image_tags(data, "webapp", "v2.0.0")
        assert len(changes) == 1
        assert data["images"][0]["newTag"] == "v2.0.0"

    def test_multiple_matches(self):
        content = (FIXTURES_DIR / "multi_image.yaml").read_text()
        data = _load_str(content)
        changes = update_image_tags(data, "api", "v5.0.0")
        # Should match api service and migrations initContainer
        assert len(changes) == 2
        assert data["services"]["api"]["image"]["tag"] == "v5.0.0"
        assert data["initContainers"][0]["image"]["tag"] == "v5.0.0"

    def test_nested_helm_values(self):
        content = (FIXTURES_DIR / "helm_values.yaml").read_text()
        data = _load_str(content)
        changes = update_image_tags(data, "webapp", "v9.9.9")
        assert len(changes) == 1
        assert data["webapp"]["image"]["tag"] == "v9.9.9"
        # Backend should be untouched
        assert data["backend"]["image"]["tag"] == "v2.3.1"

    def test_same_tag_no_change(self):
        data = _load_str("image:\n  repository: ghcr.io/myorg/webapp\n  tag: v1.0.0\n")
        changes = update_image_tags(data, "webapp", "v1.0.0")
        assert len(changes) == 0

    def test_exact_name_match(self):
        data = _load_str("image:\n  repository: webapp\n  tag: v1.0.0\n")
        changes = update_image_tags(data, "webapp", "v2.0.0")
        assert len(changes) == 1


class TestCommentPreservation:
    def test_comments_preserved_after_key_update(self):
        content = (FIXTURES_DIR / "comments_preserved.yaml").read_text()
        data = _load_str(content)
        update_keys(data, ["app.version"], ["2.0.0"])
        result = _dump_str(data)
        assert "# Top-level comment" in result
        assert "# This is the version" in result
        assert "# inline comment" in result
        assert "# app name" in result
        assert "# should be false in prod" in result

    def test_comments_preserved_after_image_update(self):
        content = (FIXTURES_DIR / "helm_values.yaml").read_text()
        data = _load_str(content)
        update_image_tags(data, "webapp", "v9.9.9")
        result = _dump_str(data)
        assert "# Helm values for webapp" in result
        assert "# Backend service" in result


class TestLoadDump:
    def test_load_and_dump_roundtrip(self, tmp_path):
        src = FIXTURES_DIR / "helm_values.yaml"
        data = load_yaml(src)
        dst = tmp_path / "output.yaml"
        dump_yaml(data, dst)
        reloaded = load_yaml(dst)
        assert reloaded["webapp"]["image"]["tag"] == "v1.0.0"


class TestDiffYaml:
    def test_diff_shows_changes(self):
        content = "app:\n  version: v1.0.0\n"
        data = _load_str(content)
        data["app"]["version"] = "v2.0.0"
        result = diff_yaml(Path("test.yaml"), content, data)
        assert "-  version: v1.0.0" in result
        assert "+  version: v2.0.0" in result

    def test_diff_empty_when_no_changes(self):
        content = "app:\n  version: v1.0.0\n"
        data = _load_str(content)
        result = diff_yaml(Path("test.yaml"), content, data)
        assert result == ""
