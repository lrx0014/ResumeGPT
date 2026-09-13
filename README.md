# ResumeGPT

ResumeGPT is an evidence-backed CV and cover-letter optimization platform. This repository currently contains the product architecture and the first runnable application skeleton.

## Current Skeleton

- Vue 3 and TypeScript web application.
- Go API built as a modular monolith.
- Independent Go worker entry point.
- Profile and job domain modules with repository ports.
- In-memory adapters for zero-dependency local development.
- PostgreSQL/pgvector and MinIO local infrastructure definitions.
- English-first internationalization setup.

The in-memory repositories are intentionally temporary. PostgreSQL remains the planned system of record and will be introduced through adapters without changing the domain services.

## Delivery Roadmap and Feature Checklist

This checklist is the project-level source of truth for planned delivery. An item is marked complete only when it is implemented, tested, and usable through its intended interface.

### Milestones

| Milestone | Outcome | Status |
|---|---|---|
| M0 — Foundation Skeleton | Runnable web application, API, worker, domain boundaries, and local development setup | Complete |
| M1 — Durable Core | PostgreSQL persistence, workspace security, object storage, and recoverable background jobs | Next |
| M2 — Profile Knowledge Base | Multi-format ingestion, OCR, evidence-linked facts, review, and retrieval | Planned |
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
- [x] Add PostgreSQL/pgvector and MinIO local Compose definitions.
- [x] Add Go tests, static analysis, frontend type checking, production builds, and dependency auditing.
- [ ] Add continuous integration for tests, builds, audits, and formatting.

### M1 — Durable Core

- [ ] Define versioned database migrations.
- [ ] Implement PostgreSQL Profile and Job repository adapters.
- [ ] Add workspace records and remove the implicit production workspace fallback.
- [ ] Integrate an OIDC identity provider.
- [ ] Implement workspace membership and Owner/Admin/Editor/Viewer RBAC.
- [ ] Add application-level workspace scoping and PostgreSQL Row-Level Security.
- [ ] Implement S3-compatible object storage and signed upload/download URLs.
- [ ] Implement durable PostgreSQL jobs, leases, heartbeats, retries, cancellation, and dead-letter handling.
- [ ] Implement a transactional outbox and idempotent consumers.
- [ ] Add audit events and baseline OpenTelemetry instrumentation.
- [ ] Add automated backup and restore checks for local/test environments.

### M2 — Profile Knowledge Base

- [ ] Support multiple profiles within a workspace.
- [ ] Accept PDF, DOC/DOCX, TeX, TXT, PNG, JPG/JPEG, and direct text sources.
- [ ] Validate real MIME type, size, content hash, and malware status.
- [ ] Build the isolated Python document-processing worker.
- [ ] Extract text, page/paragraph location, and image bounding boxes.
- [ ] Add OCR with confidence and actionable failure states.
- [ ] Define versioned fact-type JSON Schemas.
- [ ] Extract candidate facts without granting confirmed status.
- [ ] Link immutable fact versions to source evidence.
- [ ] Implement user review for low-confidence, conflicting, and sensitive facts.
- [ ] Add fact editing, confirmation, rejection, supersession, and history.
- [ ] Add pgvector indexing with mandatory workspace/profile filters.
- [ ] Add hybrid retrieval and retrieval-quality evaluation.
- [ ] Implement profile source export and complete deletion.

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
- Docker with Compose for optional local infrastructure.
- GNU Make is optional; the underlying commands can be run directly.

## Quick Start

Install web dependencies:

```bash
npm --prefix apps/web install
```

Run the API:

```bash
make dev-api
```

In another terminal, run the web application:

```bash
make dev-web
```

Open `http://localhost:5173`. Vite proxies `/api` requests to the API at `http://localhost:8080`.

The development API uses `ws_personal_dev` as a temporary workspace when no authentication context is present. This fallback must not be used in production.

## Local Infrastructure

PostgreSQL with pgvector and MinIO can be started with:

```bash
cp .env.example .env
make compose-up
```

The current API does not require these containers because it uses in-memory adapters. They are provided for the next persistence implementation phase.

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
internal/transport/httpapi/ HTTP transport
docs/                       Architecture and ADRs
```

## Architecture

- [System architecture](./docs/architecture-design.md)
- [Architecture decision records](./docs/adr/README.md)
