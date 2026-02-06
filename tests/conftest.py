from pathlib import Path

import pytest

FIXTURES_DIR = Path(__file__).parent / "fixtures"


@pytest.fixture
def fixtures_dir():
    return FIXTURES_DIR


@pytest.fixture
def tmp_yaml(tmp_path):
    """Helper to create a temporary YAML file from fixture content."""

    def _make(name: str, content: str) -> Path:
        p = tmp_path / name
        p.write_text(content)
        return p

    return _make
