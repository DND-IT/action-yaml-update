FROM ghcr.io/astral-sh/uv:0.6-python3.13-bookworm-slim AS builder

WORKDIR /app

COPY pyproject.toml .
COPY src/ src/

RUN uv venv && uv pip install .


FROM python:3.13-slim-bookworm

RUN apt-get update && \
  apt-get install -y --no-install-recommends git && \
  rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/.venv /app/.venv

ENV PATH="/app/.venv/bin:$PATH"

ENTRYPOINT ["python", "-m", "yaml_update"]
