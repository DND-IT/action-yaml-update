"""Core YAML update logic using ruamel.yaml round-trip mode."""

from __future__ import annotations

from pathlib import Path
from typing import Any

from ruamel.yaml import YAML
from ruamel.yaml.comments import CommentedMap, CommentedSeq

yaml = YAML()
yaml.preserve_quotes = True


def _coerce_value(new_value: str, existing_value: Any) -> Any:
    """Coerce a string value to match the type of the existing value."""
    if existing_value is None:
        return new_value
    if isinstance(existing_value, bool):
        return new_value.strip().lower() in ("true", "yes", "1")
    if isinstance(existing_value, int):
        try:
            return int(new_value)
        except ValueError:
            return new_value
    if isinstance(existing_value, float):
        try:
            return float(new_value)
        except ValueError:
            return new_value
    return new_value


def _resolve_key_path(data: CommentedMap, key_path: str) -> tuple[Any, str | int]:
    """Walk a dot-notation path and return (parent_container, final_key).

    Supports:
    - Simple keys: "app.version"
    - List indices: "images.0.newTag"
    """
    parts = key_path.split(".")
    current = data
    for part in parts[:-1]:
        if isinstance(current, (list, CommentedSeq)):
            try:
                idx = int(part)
            except ValueError as exc:
                raise KeyError(
                    f"Expected integer index for list, got '{part}' in path '{key_path}'"
                ) from exc
            current = current[idx]
        else:
            if part not in current:
                raise KeyError(f"Key '{part}' not found in path '{key_path}'")
            current = current[part]

    final = parts[-1]
    if isinstance(current, (list, CommentedSeq)):
        try:
            return current, int(final)
        except ValueError as exc:
            raise KeyError(
                f"Expected integer index for list, got '{final}' in path '{key_path}'"
            ) from exc
    return current, final


def update_keys(data: CommentedMap, keys: list[str], values: list[str]) -> list[dict[str, Any]]:
    """Update values at dot-notation key paths. Returns list of changes made."""
    changes = []
    for key_path, new_value in zip(keys, values):
        parent, final_key = _resolve_key_path(data, key_path)
        old_value = parent[final_key]
        coerced = _coerce_value(new_value, old_value)
        if old_value != coerced:
            parent[final_key] = coerced
            changes.append(
                {
                    "key": key_path,
                    "old": old_value,
                    "new": coerced,
                }
            )
    return changes


def _walk_image_tags(
    node: Any,
    image_name: str,
    new_tag: str,
    changes: list[dict[str, Any]],
    path: str = "",
) -> None:
    """Recursively walk a YAML tree looking for image repository/tag pairs."""
    if isinstance(node, CommentedMap):
        # Check for repository/tag pattern (Helm-style)
        if "repository" in node and "tag" in node:
            repo_val = str(node["repository"])
            if repo_val.endswith(f"/{image_name}") or repo_val == image_name:
                old_tag = node["tag"]
                coerced = _coerce_value(new_tag, old_tag)
                if old_tag != coerced:
                    node["tag"] = coerced
                    tag_path = f"{path}.tag" if path else "tag"
                    changes.append(
                        {
                            "key": tag_path,
                            "old": old_tag,
                            "new": coerced,
                        }
                    )

        # Check for name/newTag pattern (Kustomize-style)
        if "name" in node and "newTag" in node:
            name_val = str(node["name"])
            if name_val.endswith(f"/{image_name}") or name_val == image_name:
                old_tag = node["newTag"]
                coerced = _coerce_value(new_tag, old_tag)
                if old_tag != coerced:
                    node["newTag"] = coerced
                    tag_path = f"{path}.newTag" if path else "newTag"
                    changes.append(
                        {
                            "key": tag_path,
                            "old": old_tag,
                            "new": coerced,
                        }
                    )

        # Recurse into children
        for key in node:
            child_path = f"{path}.{key}" if path else str(key)
            _walk_image_tags(node[key], image_name, new_tag, changes, child_path)

    elif isinstance(node, (list, CommentedSeq)):
        for i, item in enumerate(node):
            child_path = f"{path}.{i}"
            _walk_image_tags(item, image_name, new_tag, changes, child_path)


def update_image_tags(data: CommentedMap, image_name: str, new_tag: str) -> list[dict[str, Any]]:
    """Search for image references matching image_name and update their tags."""
    changes: list[dict[str, Any]] = []
    _walk_image_tags(data, image_name, new_tag, changes)
    return changes


def load_yaml(path: Path) -> CommentedMap:
    """Load a YAML file in round-trip mode."""
    with open(path) as f:
        return yaml.load(f)


def dump_yaml(data: CommentedMap, path: Path) -> None:
    """Write YAML data back to file, preserving format."""
    with open(path, "w") as f:
        yaml.dump(data, f)


def diff_yaml(path: Path, original_content: str, data: CommentedMap) -> str:
    """Generate a human-readable diff between original and updated YAML."""
    from io import StringIO

    stream = StringIO()
    yaml.dump(data, stream)
    new_content = stream.getvalue()

    if original_content == new_content:
        return ""

    lines = []
    orig_lines = original_content.splitlines()
    new_lines = new_content.splitlines()

    lines.append(f"--- {path}")
    lines.append(f"+++ {path}")

    import difflib

    for group in difflib.unified_diff(orig_lines, new_lines, lineterm="", n=3):
        if not group.startswith(("---", "+++")):
            lines.append(group)

    return "\n".join(lines)
