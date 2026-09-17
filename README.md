# ResumeGPT

ResumeGPT is a self-hosted workspace for tailoring CVs and cover letters to specific job opportunities. It combines profile management, job tracking, scheduled job discovery, reusable templates, configurable LLM agents, PDF generation, visual review, and background-task monitoring in one application.

## TL;DR

With Docker Compose installed, start the complete application with one command:

```bash
docker compose up -d --build
```

Then open [http://localhost:5173](http://localhost:5173). The Compose stack supplies development defaults, creates its storage volumes, applies database migrations, and waits for dependencies to become healthy.

## Features

### Profiles

- Create and manage multiple profiles with a name, target role, default language, and Markdown-friendly content.
- Enter profile content directly or import text from TXT, Markdown, TeX, DOC, DOCX, PDF, PNG, and JPEG files.
- Review and edit extracted text before saving it to a profile.
- Attach an optional JPEG or PNG avatar for CV layouts that support a photo.
- Search, filter, paginate, edit, and delete profiles from the web interface.

### Job Opportunities

- Track job title, company, location, country, city, work mode, employment type, source URL, description, and application status.
- Create opportunities manually or import up to 50 public job URLs at once.
- Use fast deterministic extraction for supported LinkedIn and Indeed pages.
- Enable the Job Import Agent for AI-assisted extraction from other public HTTPS job pages.
- Edit imported information and update application status directly from the opportunity list.
- Filter opportunities by origin so manual, URL-imported, and Job Hunter results remain distinguishable.
- Select multiple opportunities and create CV or cover-letter tasks in a batch.

### Job Hunter

- Schedule recurring searches by occupation, location, work mode, contract type, experience, keywords, and an additional prompt.
- Optionally use a saved profile as matching context.
- Limit each run to at most 10 new opportunities.
- Run, pause, resume, edit, or delete a Hunter from the web interface.
- Deduplicate discovered URLs before creating opportunities.
- Keep blocked or unparseable results out of the opportunity list and collect them in a confirmation inbox for manual review.

### Templates

- Manage separate CV and cover-letter templates.
- Upload DOC, DOCX, single-file TeX, or multi-file LaTeX ZIP projects.
- Specify a LaTeX entry file when a ZIP does not use an unambiguous `main.tex`.
- Scan uploaded sources, extract LLM-readable content, and generate a cached PDF preview.
- View, update, download, and delete custom templates.
- Use the included read-only Rezume LaTeX template, with attribution to its original [Overleaf source](https://www.overleaf.com/latex/templates/rezume/kfrvqywfkwjs).

### CV and Cover-Letter Generation

- Generate a tailored CV or cover letter from a saved profile and job opportunity.
- Use an optional LaTeX template, or leave the template empty and let the Document Designer Agent create a print-ready HTML/CSS layout.
- Configure a different default LLM for each Agent, or override a generation with one model for the entire workflow.
- Run Writer, Template Applying or Document Designer, and Visual Reviewer roles through bounded LangChainGo agent executors.
- Generate LaTeX and HTML/CSS artifacts, plus Word template previews, through isolated rendering tools.
- Rasterize PDF pages for visual review and perform up to two automatic layout-repair rounds.
- Preserve a valid PDF with a visible warning when the selected model cannot perform visual inspection.
- Fall back to a safe basic layout if an AI-generated document cannot be rendered reliably.
- Follow generation progress, inspect drafts, intermediate PDFs, review feedback, warnings, and user prompts in a timeline.
- Edit a completed or failed application's inputs and regenerate it without discarding earlier timeline records.
- Send a follow-up instruction to revise a completed artifact.

### LLM Connections and Settings

- Configure OpenAI, OpenAI-compatible, and local Ollama connections.
- Store API tokens encrypted at rest and never return plaintext tokens to the browser.
- Discover available models automatically from `/models` or Ollama's `/api/tags` endpoint.
- Assign default connections and models independently to the Writer, Template Applying, Document Designer, Visual Reviewer, Job Import, and Job Hunter Agents.
- Override the default routing with a single model for an individual generation.
- Configure the interface language and System, Light, or Dark theme.

### Background Processing and Operations

- Execute document extraction, template preparation, job import, job hunting, and document generation through a PostgreSQL-backed durable work queue.
- Recover leased jobs, retry transient failures, and persist task lifecycle events.
- Inspect task state, attempts, sanitized inputs, errors, and logs in the Task Monitor.
- Store source files, avatars, previews, and generated PDFs in S3-compatible object storage.
- Persist transactional outbox and audit records for core mutations.
- Emit structured logs and OpenTelemetry traces with W3C trace-context propagation.
- Create PostgreSQL backups and run an automated restore check with the supplied scripts.

## Local Stack

The default Compose deployment runs:

| Service | Purpose | Local access |
|---|---|---|
| `web` | Vue 3 application served by Nginx | `http://localhost:5173` |
| `api` | Go HTTP API | `http://localhost:8080` |
| `worker` | Durable background processors and schedulers | Internal |
| `document-worker` | Malware scanning, extraction, OCR, preview, and PDF rendering | Internal |
| `web-worker` | Isolated Playwright page rendering for AI-assisted imports | Internal |
| `postgres` | Application data, queue, events, and settings | `localhost:5432` |
| `minio` | S3-compatible object storage | API `localhost:9000`, console `localhost:9001` |
| `migrate` | One-shot database migration process | Internal |

Check status or follow logs with:

```bash
docker compose ps
docker compose logs -f
```

Stop the stack without deleting data:

```bash
docker compose down
```

PostgreSQL and MinIO data remain in named volumes. Use `docker compose down --volumes` only when you intentionally want to erase local application data.

## First Use

1. Open `http://localhost:5173/settings`.
2. Add an OpenAI, OpenAI-compatible, or Ollama connection and test it.
3. Choose default models for the Agents you plan to use.
4. Create a Profile and save your source content.
5. Add or import a Job Opportunity.
6. Create a CV or cover letter, optionally selecting a template.

For Ollama running on the Docker host, use `http://host.docker.internal:11434` as the Base URL.

## Configuration

Compose provides development defaults, so an `.env` file is optional. Copy `.env.example` when you want to customize ports, credentials, storage, authentication, or tracing:

```bash
cp .env.example .env
```

Important settings include:

| Variable | Purpose |
|---|---|
| `WEB_PORT` | Browser-facing web port; default `5173` |
| `API_PORT` | Browser-facing API port; default `8080` |
| `POSTGRES_*` | Local PostgreSQL database and credentials |
| `MINIO_ROOT_USER`, `MINIO_ROOT_PASSWORD` | Local object-storage credentials |
| `SETTINGS_ENCRYPTION_KEY` | Encrypts stored LLM API tokens |
| `AUTH_MODE` | `development` or `oidc` |
| `OIDC_ISSUER`, `OIDC_CLIENT_ID` | Required when `AUTH_MODE=oidc` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Optional OpenTelemetry collector endpoint |

The built-in encryption key and storage credentials are development-only defaults. Set private values before using the application outside a local machine.

## Development

Host development requires Go 1.26.8 or later, Node.js 22 or later, Python 3.12, `uv`, and GNU Make.

Install frontend and Python dependencies:

```bash
npm --prefix apps/web install
cd services/document-worker && uv sync --dev --locked && cd ../..
cd services/web-worker && uv sync --dev --locked && cd ../..
```

Start PostgreSQL and MinIO, apply migrations, and run the main processes in separate terminals:

```bash
cp .env.example .env
make compose-infra
make migrate
make dev-api
make dev-worker
make dev-web
```

The full worker also expects the document and web workers. For end-to-end development, the complete Compose stack is the simplest option.

Run validation:

```bash
make test
make build
```

Useful commands:

```bash
make compose-up
make compose-ps
make compose-logs
make compose-down
make backup
make restore-check
```

## Repository Layout

```text
apps/web/                   Vue 3 and TypeScript frontend
cmd/api/                    Go API entry point
cmd/worker/                 Go background worker entry point
cmd/migrate/                Embedded migration runner
internal/                   Domain modules, services, ports, and adapters
migrations/                 Versioned PostgreSQL migrations
services/document-worker/   Isolated document processing and rendering service
services/web-worker/        Isolated Playwright browser service
scripts/                    Backup and restore-check utilities
docs/                       Current architecture documentation
```

## Architecture

See [Architecture](./docs/architecture-design.md) for the implemented component model, persistence layout, workflows, and security boundaries.

## License

ResumeGPT is distributed under the terms in [LICENSE](./LICENSE). The bundled Rezume template retains its own attribution and license metadata in the application.
