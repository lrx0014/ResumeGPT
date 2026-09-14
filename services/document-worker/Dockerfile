FROM python:3.12-slim

ARG DEBIAN_FRONTEND=noninteractive
RUN apt-get update \
    && apt-get install --yes --no-install-recommends antiword clamav clamav-freshclam libreoffice-writer tesseract-ocr tesseract-ocr-eng texlive-fonts-extra texlive-fonts-recommended texlive-latex-extra texlive-plain-generic \
    && freshclam \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY pyproject.toml uv.lock ./
RUN pip install --no-cache-dir uv \
    && uv sync --frozen --no-dev
COPY resumegpt_document_worker ./resumegpt_document_worker

USER 65532:65532
EXPOSE 8090
ENTRYPOINT ["/app/.venv/bin/python", "-m", "resumegpt_document_worker", "--serve"]
