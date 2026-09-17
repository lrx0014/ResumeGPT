# ResumeGPT Architecture Decision Records

This directory records significant ResumeGPT architecture decisions. Once accepted, an ADR's historical decision is not rewritten. If the decision changes, create a new ADR and mark the old one as `Superseded`.

## Statuses

- `Proposed`: Recommended but not yet approved for implementation.
- `Accepted`: Approved; implementation should follow it.
- `Deprecated`: Discouraged for new code but not formally replaced.
- `Superseded`: Replaced by a newer ADR.

## Decision Index

| ADR | Summary | Status |
|---|---|---|
| [ADR-001](./ADR-001-modular-monolith-and-workers.md) | Modular monolith plus independent workers; extract services only on objective signals | Proposed |
| [ADR-002](./ADR-002-profile-facts-and-evidence.md) | Relational fact core, typed JSONB, and immutable evidence | Superseded |
| [ADR-003](./ADR-003-document-intermediate-representation.md) | Versioned JSON IR separating content from rendering | Proposed |
| [ADR-004](./ADR-004-vector-store.md) | Use pgvector for the MVP; evaluate Qdrant at measured thresholds | Superseded |
| [ADR-005](./ADR-005-durable-jobs-and-workflows.md) | PostgreSQL jobs and transactional outbox for the MVP | Proposed |
| [ADR-006](./ADR-006-template-and-rendering-boundary.md) | Managed template admission and sandboxed rendering | Superseded in part by ADR-014 |
| [ADR-007](./ADR-007-llm-data-and-routing-policy.md) | Data-classification-driven provider routing and governance | Superseded |
| [ADR-008](./ADR-008-unsupported-claim-gate.md) | Block verified export for unsupported high-risk claims | Proposed |
| [ADR-009](./ADR-009-job-crawling-policy.md) | Compliant, constrained, auditable crawling with manual fallback | Superseded |
| [ADR-010](./ADR-010-workspace-tenancy-and-authorization.md) | Workspace tenant boundary and defense-in-depth authorization from day one | Proposed |
| [ADR-011](./ADR-011-simple-editable-profiles.md) | One editable text body per profile with review-before-save document extraction | Accepted |
| [ADR-012](./ADR-012-simple-job-tracking-and-import.md) | One editable Job record with safe background import and manual fallback | Accepted |
| [ADR-013](./ADR-013-user-managed-llm-connections.md) | User-managed encrypted LLM connections with per-Agent defaults | Accepted |
| [ADR-014](./ADR-014-simple-template-library.md) | Simple user-managed TeX and Word template library | Accepted |
| [ADR-015](./ADR-015-bounded-agent-generation.md) | Bounded single- or multi-model generation with visual repair | Accepted |
| [ADR-016](./ADR-016-agent-assisted-job-import.md) | Agent-first extraction for arbitrary public job pages | Accepted |
| [ADR-017](./ADR-017-scheduled-agent-job-hunting.md) | Scheduled agent search feeding the existing job-import pipeline | Accepted |

## Maintenance Rules

Every implementation PR should reference its ADR and include the corresponding contract or security tests. Reconsideration must be supported by production metrics, cost, incident evidence, or new compliance requirements—not preference alone.
