FROM mcr.microsoft.com/playwright/python:v1.55.0-noble

WORKDIR /app
COPY pyproject.toml ./
COPY resumegpt_web_worker ./resumegpt_web_worker
RUN pip install --no-cache-dir .

USER 65532:65532
EXPOSE 8091
ENTRYPOINT ["python", "-m", "resumegpt_web_worker", "--serve"]
