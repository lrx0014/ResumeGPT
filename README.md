# ResumeGPT

ResumeGPT is an evidence-backed CV and cover-letter optimization platform. This repository currently contains the product architecture and the first runnable application skeleton.

## Current Skeleton

- Vue 3 and TypeScript web application.
- Go API built as a modular monolith.
- Independent Go worker entry point.
- Profile and job domain modules with repository ports.
- PostgreSQL adapters with embedded migrations, workspace RLS, audit events, and transactional outbox writes.
- Development and OIDC authentication modes with workspace RBAC.
- S3-compatible signed upload and download URLs, with MinIO for local development.
- PostgreSQL-backed durable jobs and a leased outbox dispatcher.
- Optional in-memory adapters for zero-dependency local development.
- OpenTelemetry HTTP tracing and W3C trace-context propagation.
- Profile knowledge with immutable text evidence, versioned review, safe retrieval, export, and deletion.
- An isolated Python extraction engine for TXT, TeX, DOC/DOCX, PDF, PNG, and JPEG, including OCR metadata.
- English-first internationalization setup.

Infrastructure integrations remain behind application ports and adapters so that storage, identity, and messaging choices can change without rewriting domain services.

## Delivery Roadmap and Feature Checklist

This checklist is the project-level source of truth for planned delivery. An item is marked complete only when it is implemented, tested, and usable through its intended interface.

### Milestones

| Milestone | Outcome | Status |
|---|---|---|
| M0 — Foundation Skeleton | Runnable web application, API, worker, domain boundaries, and local development setup | Complete |
| M1 — Durable Core | PostgreSQL persistence, workspace security, object storage, and recoverable background jobs | Complete |
| M2 — Profile Knowledge Base | Evidence registry and extraction engine are complete; staged processing and semantic retrieval remain | In progress |
| M3 — Job and Application Tracking | URL acquisition, normalized requirements, application workflow, and reporting | Planned |
| M4 — Tailored Content Generation | Provider-independent LLM orchestration, CVs, cover letters, and iterative revision | Planned |
| M5 — Rendering, Validation, and Trust | Managed templates, DOCX/PDF output, visual QA, and unsupported-claim controls | Planned |
| M6 — Beta and Scale Readiness | Collaboration, quotas, observability, deployment automation, and scale-out adapters | Planned |

### M0 — Foundation Skeleton

- [x] Record the system architecture and initial ADRs.
- [x] Establish the Go module and modular backend layout.
- [x] Provide separate API and worker entry points.
- [x] Define Profile and Job domain models, services, and repository ports.
- [x] Provide in-memory repositories for zero-dependency development.
- [x] Expose health, capability, Profile, and Job HTTP endpoints.
- [x] Add request IDs, structured logging, CORS, timeouts, and graceful shutdown.
- [x] Create the Vue 3, TypeScript, Router, Pinia, and i18n application shell.
- [x] Create Overview, Profiles, Jobs, and Generate views.
- [x] Connect Profile and Job forms to the development API.
- [x] Add a complete local Compose stack with web, API, worker, automatic migrations, PostgreSQL/pgvector, and MinIO.
- [x] Add Go tests, static analysis, frontend type checking, production builds, and dependency auditing.
- [x] Add continuous integration for tests, builds, audits, formatting, and PostgreSQL integration.

### M1 — Durable Core

- [x] Define versioned, embedded database migrations.
- [x] Implement PostgreSQL Profile and Job repository adapters.
- [x] Add workspace and membership records and remove the implicit production workspace fallback.
- [x] Integrate OIDC Bearer-token verification for production API authentication.
- [x] Implement workspace membership and Owner/Admin/Editor/Viewer RBAC.
- [x] Add application-level workspace scoping and PostgreSQL Row-Level Security.
- [x] Implement S3-compatible object storage and signed upload/download URLs.
- [x] Implement durable PostgreSQL jobs, leases, heartbeats, retries, cancellation, and terminal failure handling.
- [x] Write transactional outbox events with Profile and Job mutations.
- [x] Implement leased outbox publication with stable event IDs and retry backoff.
- [x] Persist audit events with Profile and Job mutations.
- [x] Add baseline OpenTelemetry HTTP tracing and W3C context propagation.
- [x] Add automated backup and restore checks for local/test environments.

### M2a — Evidence Registry

- [x] Support multiple profiles within a workspace.
- [x] Accept direct text and UTF-8 TXT sources.
- [x] Validate UTF-8 text, real MIME type, size, line limits, control characters, and SHA-256 content hash.
- [x] Deduplicate identical source content within a profile.
- [x] Extract immutable paragraph-level text evidence with source and content hashes.
- [x] Define the versioned statement fact JSON Schema.
- [x] Create verbatim candidate statements without granting confirmed status.
- [x] Link immutable fact versions to source evidence.
- [x] Require explicit user review and treat all imported statements as sensitive by default.
- [x] Add fact editing, assertion, confirmation, dispute, rejection, optimistic concurrency, and history.
- [x] Add safe lexical retrieval restricted to confirmed, non-sensitive current versions.
- [x] Implement profile knowledge export and complete active-database source deletion.
- [x] Provide a Profile knowledge UI for import, provenance, review, history, retrieval, export, and deletion.

### M2b — Isolated Document Extraction

- [ ] Accept PDF, DOC/DOCX, TeX, PNG, and JPG/JPEG sources through staged object storage.
- [x] Build a signature-driven Python extraction engine for TXT, TeX, DOC/DOCX, PDF, PNG, and JPG/JPEG.
- [x] Validate binary signatures, extension agreement, expanded DOCX size, total size, and SHA-256 content hash.
- [x] Fail closed with actionable states when malware scanning, conversion, PDF rendering, or OCR is unavailable.
- [x] Build and verify the non-root extraction container locally.
- [ ] Provision current malware definitions and scan before releasing quarantined content.
- [ ] Connect staged object uploads to the durable document-processing job consumer.
- [ ] Add transactional inbox deduplication with the first asynchronous document consumer.
- [x] Extract text with page, paragraph, confidence, and optional image bounding-box location.
- [x] Add Tesseract OCR for images and image-only PDF pages with confidence and actionable failure states.

### M2c — Fact Intelligence and Retrieval

- [ ] Add typed fact schemas for employment, education, projects, skills, awards, and certifications.
- [ ] Extract structured candidate facts without granting confirmed status.
- [ ] Implement user review for low-confidence, conflicting, and sensitive facts.
- [ ] Add pgvector indexing with mandatory workspace/profile filters.
- [ ] Add hybrid retrieval and retrieval-quality evaluation.

### M3 — Job and Application Tracking

- [x] Support basic manual Job creation and listing.
- [x] Capture title, company, location, source URL, and description.
- [ ] Implement Job detail and editing.
- [ ] Implement single and batch URL import.
- [ ] Add SSRF-safe HTTP acquisition and a sandboxed browser fallback.
- [ ] Add dedicated adapters for approved job sites.
- [ ] Store immutable Job snapshots and detect page changes.
- [ ] Extract responsibilities, must-haves, nice-to-haves, seniority, and keywords.
- [ ] Detect duplicate, incomplete, expired, and removed postings.
- [ ] Implement the configurable application status state machine.
- [ ] Add status history, notes, deadlines, priorities, and channels.
- [ ] Build application funnel, activity, and conversion reports.
- [ ] Add report export and optional reminders.

### M4 — Tailored Content Generation

- [ ] Define and version the ResumeDocument and CoverLetterDocument JSON Schemas.
- [ ] Implement the provider-independent LLM and embedding gateway.
- [ ] Add a provider capability registry and logical model aliases.
- [ ] Support at least one cloud provider and one local OpenAI-compatible endpoint.
- [ ] Enforce data-classification, residency, retention, and fallback policy.
- [ ] Freeze Profile, Job, prompt, template, and model input snapshots.
- [ ] Build the job-requirement-to-profile-fact matching matrix.
- [ ] Generate a schema-constrained content plan and draft.
- [ ] Generate tailored one-page, two-page, and custom-length CVs.
- [ ] Generate tailored cover letters.
- [ ] Add streaming generation progress and cancellation.
- [ ] Implement artifact revisions with parent history and restoration.
- [ ] Support conversational revision through allowlisted structured operations.
- [ ] Show semantic diffs and allow users to lock sections.
- [ ] Track model version, prompt version, token usage, latency, and cost.

### M5 — Rendering, Validation, and Trust

- [ ] Define the managed DOCX and TeX template capability model.
- [ ] Implement template quarantine, scanning, fixture rendering, and approval.
- [ ] Build a network-isolated, resource-limited render worker.
- [ ] Generate DOCX from the document intermediate representation.
- [ ] Generate PDF through pinned LibreOffice and TeX toolchains.
- [ ] Produce per-page preview images.
- [ ] Detect page-count violations, overflow, clipping, overlap, blank pages, and missing fonts.
- [ ] Compare normalized PDF text with the structured draft.
- [ ] Add bounded automatic layout repair.
- [ ] Split generated text into atomic claims.
- [ ] Validate names, dates, organizations, titles, and metrics deterministically.
- [ ] Add evidence-based semantic-expansion checks.
- [ ] Implement blocking, warning, and style finding levels.
- [ ] Implement Verified and explicitly acknowledged Unverified export policies.
- [ ] Revalidate after every AI or manual revision.
- [ ] Add prompt-injection and malicious-template regression suites.

### M6 — Beta and Scale Readiness

- [ ] Add workspace invitations and multi-user collaboration.
- [ ] Add quotas, rate limits, budget controls, and usage reporting.
- [ ] Complete data export, deletion, retention, and consent workflows.
- [ ] Add notification delivery and user-configurable reminders.
- [ ] Add multilingual document generation and template validation beyond English.
- [ ] Build de-identified golden datasets and quality release gates.
- [ ] Add model shadowing, canary rollout, and fallback monitoring.
- [ ] Add production container images, Kubernetes manifests, health probes, and autoscaling.
- [ ] Add production dashboards, alerts, SLOs, and recovery runbooks.
- [ ] Add CI/CD with backward-compatible migration gates and rollback support.
- [ ] Evaluate Kafka only when event throughput or consumer count justifies it.
- [ ] Evaluate Qdrant only when pgvector misses measured latency, recall, or isolation targets.
- [ ] Evaluate a durable workflow engine when long-running workflow complexity exceeds the PostgreSQL job model.
- [ ] Complete the target-market privacy, security, and compliance review.

### Checklist Maintenance

- Keep completed items checked when a later implementation replaces them; create a follow-up item for migration work.
- Add links to the relevant ADR, issue, or pull request when a task becomes active.
- Split an item when only part of its acceptance criteria is implemented.
- Update milestone status only when its outcome is demonstrably usable end to end.

## Prerequisites

- Go 1.24 or later.
- Node.js 22.12 or later and npm.
- Python 3.10 or later and uv.
- Docker with Compose for the complete local stack or optional infrastructure-only development.
- GNU Make is optional; the underlying commands can be run directly.

## Quick Start

Start the complete application stack:

```bash
cp .env.example .env
make compose-up
```

Open `http://localhost:5173`. Compose builds and starts the Vue application, Go API, Go worker, PostgreSQL, and MinIO, and applies database migrations before the API becomes ready.

Useful local endpoints are:

- Web application: `http://localhost:5173`
- API health: `http://localhost:8080/healthz`
- MinIO console: `http://localhost:9001`

Inspect service status or follow logs with:

```bash
make compose-ps
make compose-logs
```

Development authentication maps requests to the seeded `ws_personal_dev` workspace. Production deployments must use `AUTH_MODE=oidc`, configure the issuer and client ID, and send an explicit `X-Workspace-ID` header.

## Host Development

Install web and document-worker dependencies:

```bash
npm --prefix apps/web install
cd services/document-worker
uv sync --dev --locked
cd ../..
```

Start only PostgreSQL and MinIO, then apply migrations:

```bash
cp .env.example .env
make compose-infra
make migrate
```

Run the API and web application in separate terminals:

```bash
make dev-api
make dev-web
```

Open `http://localhost:5173`. Vite proxies `/api` requests to the API at `http://localhost:8080`.

## Local Infrastructure

PostgreSQL with pgvector and MinIO can be started with:

```bash
cp .env.example .env
make compose-infra
```

The example environment selects PostgreSQL and MinIO. Compose overrides service-to-service endpoints while preserving the localhost endpoints used by host processes and browser-facing signed object URLs. Set `PERSISTENCE_MODE=memory` and `OBJECT_STORAGE_MODE=memory` for a zero-dependency host development session.

`make compose-down` stops the stack but retains PostgreSQL and MinIO volumes. Use `docker compose down --volumes` only when you intentionally want to delete local application data.

Create a local database backup or verify a complete backup-and-restore cycle:

```bash
make backup
make restore-check
```

Set `OTEL_EXPORTER_OTLP_ENDPOINT` to send API and worker traces to an OpenTelemetry-compatible collector. When it is empty, trace propagation remains enabled without exporting spans.

Run the extraction engine directly against a quarantined document:

```bash
cd services/document-worker
uv run python -m resumegpt_document_worker /absolute/path/to/document.pdf
```

The command intentionally fails when ClamAV or current malware definitions are unavailable. The hidden `--skip-malware-scan` switch exists for automated parser tests only and must not be used for user documents.

## Validation

```bash
make test
make build
```

## Repository Layout

```text
apps/web/                   Vue application
cmd/api/                    Go API entry point
cmd/worker/                 Go worker entry point
internal/profile/           Profile domain and application service
internal/job/               Job domain and application service
internal/adapters/          Infrastructure adapters
internal/platform/          Database, queue, storage, and telemetry abstractions
internal/transport/httpapi/ HTTP transport
migrations/                 Versioned embedded PostgreSQL migrations
schemas/                    Versioned JSON Schemas
services/document-worker/   Isolated Python document extraction
scripts/                    Local backup and restore verification
docs/                       Architecture and ADRs
```

## Architecture

- [System architecture](./docs/architecture-design.md)
- [Architecture decision records](./docs/adr/README.md)
