# ADR-002: Profile Facts and Evidence

- Status: Proposed
- Date: 2026-09-13
- Decision owners: TBD

## Context

The system must distinguish source material, model-extracted candidate facts, user-confirmed facts, and wording in a final document. Storing only vectors or free-form JSON would make validation, conflict detection, deletion, and provenance unreliable.

## Decision

Use PostgreSQL as the system of record with a hybrid model: relational common fields, typed JSONB payloads, immutable source evidence, and versioned facts.

### Core Entities

- `profile_sources`: metadata, content hash, object reference, and processing state for each file or text submission.
- `source_segments`: immutable source text with page, paragraph, and optional bounding-box location.
- `facts`: stable identity, fact type, current-version pointer, and lifecycle state.
- `fact_versions`: immutable versions of values, validity periods, confidence, and verification state.
- `fact_evidence_links`: many-to-many links from a fact version to source segments.
- `profile_fact_links`: explicit association between facts and one or more profiles.

### Relational Fields

`workspace_id`, profile association, `fact_type`, `verification_status`, validity range, sensitivity, version, and timestamps must be normal columns. Type-specific content belongs in JSONB validated by `fact_type + schema_version`.

### Verification State

```text
extracted -> user_confirmed
    |      -> disputed
    +------> rejected
user_confirmed -> superseded
```

- A model can never assign `user_confirmed`.
- Direct user input may be `user_asserted`, but high-risk claims still require explicit confirmation. It may be implemented as a distinct state or confirmation event.
- Editing a confirmed fact creates a new version instead of overwriting history.
- Conflicting facts remain visible in a conflict group; the model must not silently choose one.

### Evidence Rules

- Models cannot rewrite source segments.
- Each segment retains a content hash, parser version, and source-file version.
- When a source is deleted, its text and vectors follow the retention policy. Referencing facts become “source evidence unavailable” and require renewed confirmation rather than retaining phantom evidence.
- Embeddings are rebuildable derived data, not facts.

## Alternatives

- **A normalized table per fact type:** Strong queries, but excessive schema and migration cost across domains.
- **One JSON document:** Quick initially, but weak for concurrent edits, references, indexes, migration, and audit.
- **Vector database only:** Cannot provide transactions, versions, or evidence integrity.

## Consequences

Generation and validation can reference immutable `fact_version_id` values, while export and deletion remain tractable. The cost is maintaining a JSON Schema registry, migrations, and type-specific validators.

## Implementation Constraints

- Validate every JSONB write against its schema.
- Evidence links point to a fact version, not a mutable fact head.
- Every query is scoped by workspace and profile.
- Store normalized forms for dates, numbers, currency, and percentages separately from display text.
- Exclude or redact highly sensitive values from logs and embeddings.

## Revisit

Adding a fact type does not require review. Revisit only if measured JSONB query or migration cost justifies dedicated tables.

