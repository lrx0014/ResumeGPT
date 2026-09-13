FROM python:3.12-slim

ARG DEBIAN_FRONTEND=noninteractive
RUN apt-get update \
    && apt-get install --yes --no-install-recommends antiword clamav tesseract-ocr tesseract-ocr-eng \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY pyproject.toml uv.lock ./
RUN pip install --no-cache-dir uv \
    && uv sync --frozen --no-dev
COPY resumegpt_document_worker ./resumegpt_document_worker

USER 65532:65532
ENTRYPOINT ["/app/.venv/bin/python", "-m", "resumegpt_document_worker"]
