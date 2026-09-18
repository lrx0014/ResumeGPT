FROM python:3.12-slim

ARG DEBIAN_FRONTEND=noninteractive
RUN apt-get update \
    && apt-get install --yes --no-install-recommends antiword clamav clamav-freshclam fonts-noto-cjk libpango-1.0-0 libpangoft2-1.0-0 libreoffice-writer shared-mime-info tesseract-ocr tesseract-ocr-eng texlive-fonts-extra texlive-fonts-recommended texlive-lang-chinese texlive-latex-extra texlive-plain-generic texlive-xetex \
    && freshclam \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
ENV XDG_CACHE_HOME=/tmp/resumegpt-cache
COPY pyproject.toml uv.lock ./
RUN pip install --no-cache-dir uv \
    && uv sync --frozen --no-dev
COPY resumegpt_document_worker ./resumegpt_document_worker

USER 65532:65532
EXPOSE 8090
ENTRYPOINT ["/app/.venv/bin/python", "-m", "resumegpt_document_worker", "--serve"]
