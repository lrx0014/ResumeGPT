# ADR-005: PostgreSQL Jobs and Transactional Outbox for the MVP

- Status: Proposed
- Date: 2026-09-13
- Decision owners: TBD

## Context

Ingestion, crawling, LLM generation, and rendering are long-running tasks that need retries and recovery. Kafka transports events but is not a complete workflow engine. Durable-execution systems such as Temporal are powerful, but introduce another operational system and programming model.

## Decision

For the MVP, persist jobs, step attempts, leases, and a transactional outbox in PostgreSQL. Use at-least-once delivery and idempotent consumers; do not claim exactly-once semantics.

### Data and Execution Rules

- Create business records and outbox events in the same database transaction.
- Workers claim jobs with `FOR UPDATE SKIP LOCKED` and maintain `lease_owner`, `lease_expires_at`, and `heartbeat_at`.
- Every step has a stable idempotency key. Record attempt state around external side effects.
- Retry only classified transient errors with exponential backoff, jitter, maximum attempts, and a deadline.
- Classify failures as `retryable`, `permanent`, `needs_user_action`, or `security_quarantine`.
- Store large payloads in object storage; job/outbox rows contain references, hashes, and versions.
- Support cancellation, dead-lettering, audited replay, and orphaned-job recovery.
- Both publisher and consumer maintain deduplication/inbox records.

### Kafka Is Not the Job Record

Kafka may receive outbox events when consumer count, throughput, event retention, or cross-service integration warrants it. User-visible progress, business task state, and manual recovery still require a durable domain model.

### Triggers for a Workflow Engine

- Workflows commonly last hours or days and include timers, signals, or human waiting.
- Branching, compensation, and fan-out/fan-in make the custom state machine a recurring source of recovery defects.
- Workflow version migration, visualization, and cross-service orchestration become major engineering costs.
- The team can operate or purchase a managed workflow platform.

Pilot one complex workflow first; do not migrate every background job at once.

## Alternatives

- **Use Temporal immediately:** Strong long-term capabilities, but MVP complexity does not yet justify it.
- **Use Kafka immediately:** Good for event streams, but does not solve business state and compensation.
- **Redis queue:** May accelerate dispatch, but is not the sole durable job record.

## Consequences

The initial dependency and transaction model stays small. The team must correctly implement leases, heartbeats, retries, cancellation, DLQ, and telemetry. It should migrate rather than endlessly expand a custom engine when workflows become complex.

## Validation

- Inject crashes before and after claim, external call, and completion.
- Verify duplicate delivery does not create duplicate artifacts or cost records.
- Verify expired leases are safely reclaimed and cancellation cannot corrupt completed work.

## Revisit

Review when any workflow exceeds eight persisted steps, includes cross-day waiting, or recovery defects repeatedly become a major incident source.

