"""Write action outputs to GITHUB_OUTPUT and provide logging helpers."""

from __future__ import annotations

import os
import sys
import uuid


def _write_output(name: str, value: str) -> None:
    """Write a single output to the GITHUB_OUTPUT file."""
    output_file = os.environ.get("GITHUB_OUTPUT")
    if not output_file:
        return

    with open(output_file, "a") as f:
        if "\n" in value:
            delimiter = f"ghadelimiter_{uuid.uuid4()}"
            f.write(f"{name}<<{delimiter}\n{value}\n{delimiter}\n")
        else:
            f.write(f"{name}={value}\n")


def set_output(name: str, value: str) -> None:
    """Set a GitHub Action output value."""
    _write_output(name, value)
    log_info(f"Output {name}={value[:100]}{'...' if len(value) > 100 else ''}")


def log_info(message: str) -> None:
    """Log an info message."""
    print(message, flush=True)


def log_warning(message: str) -> None:
    """Log a warning message in GitHub Actions format."""
    print(f"::warning::{message}", file=sys.stderr, flush=True)


def log_error(message: str) -> None:
    """Log an error message in GitHub Actions format."""
    print(f"::error::{message}", file=sys.stderr, flush=True)


def log_group(title: str) -> None:
    """Start a log group."""
    print(f"::group::{title}", flush=True)


def log_endgroup() -> None:
    """End a log group."""
    print("::endgroup::", flush=True)
