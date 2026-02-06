"""Tests for yaml_update.outputs module."""

from yaml_update.outputs import (
    log_error,
    log_info,
    log_warning,
    set_output,
)


class TestSetOutput:
    def test_single_line_output(self, tmp_path, monkeypatch):
        output_file = tmp_path / "github_output"
        output_file.touch()
        monkeypatch.setenv("GITHUB_OUTPUT", str(output_file))

        set_output("changed", "true")

        content = output_file.read_text()
        assert "changed=true\n" in content

    def test_multiline_output(self, tmp_path, monkeypatch):
        output_file = tmp_path / "github_output"
        output_file.touch()
        monkeypatch.setenv("GITHUB_OUTPUT", str(output_file))

        set_output("diff", "line1\nline2\nline3")

        content = output_file.read_text()
        assert "diff<<ghadelimiter_" in content
        assert "line1\nline2\nline3" in content

    def test_multiple_outputs(self, tmp_path, monkeypatch):
        output_file = tmp_path / "github_output"
        output_file.touch()
        monkeypatch.setenv("GITHUB_OUTPUT", str(output_file))

        set_output("changed", "true")
        set_output("pr_number", "42")

        content = output_file.read_text()
        assert "changed=true\n" in content
        assert "pr_number=42\n" in content

    def test_no_github_output_env(self, monkeypatch):
        monkeypatch.delenv("GITHUB_OUTPUT", raising=False)
        # Should not raise
        set_output("key", "value")


class TestLogging:
    def test_log_info(self, capsys):
        log_info("hello world")
        assert "hello world" in capsys.readouterr().out

    def test_log_warning(self, capsys):
        log_warning("something went wrong")
        assert "::warning::something went wrong" in capsys.readouterr().err

    def test_log_error(self, capsys):
        log_error("fatal error")
        assert "::error::fatal error" in capsys.readouterr().err
