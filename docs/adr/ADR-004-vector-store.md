# ADR-004: Use pgvector for the MVP

- Status: Proposed
- Date: 2026-09-13
- Decision owners: TBD

## Context

Profile and job retrieval require vector similarity plus strict workspace/profile filtering. At MVP scale, a separate vector cluster would add backup, monitoring, consistency, and local-development overhead.

## Decision

Use PostgreSQL with pgvector for the MVP, hidden behind a product-owned `VectorIndex` port. Qdrant is the preferred scale-out candidate, not a mandatory initial component.

### MVP Implementation

- PostgreSQL remains the source of truth for text, metadata, and index state; embeddings are rebuildable.
- Every embedding includes `workspace_id`, `profile_id`, `source_segment_id`, `embedding_model`, dimensions, and source version.
- Start with exact search at small scale; establish a quality baseline before adding HNSW.
- Monitor approximate-search recall and compare sampled results with exact search.
- Enforce tenant/profile filters inside the repository. Never interpolate raw caller-provided filters into SQL.
- Separate models or dimensions by table/partition or constrained partial indexes; do not mix distance spaces.
- Reliably cascade embedding deletion and periodically scan for orphans.

### Triggers for Evaluating Qdrant

Start an evaluation when a sustained issue remains after pgvector tuning:

1. Vector workloads materially harm OLTP CPU, memory, vacuum, or backup windows.
2. Filtered tenant/profile queries cannot meet target p95 latency and recall.
3. Independent vector scaling, dedicated sharding, hybrid retrieval, or high-rate bulk indexing is required.
4. Embedding scale or update rate makes PostgreSQL more expensive to operate than a dedicated service.

Migrate through dual writes, backfill, shadow queries, consistency checks, read cutover, and finally old-write retirement.

## Alternatives

- **Use Qdrant immediately:** Mature capabilities, but adds a stateful system before it is justified.
- **SQL only:** Acceptable for the earliest spike, but cannot validate semantic retrieval.
- **Operate two backends permanently:** Expands the test matrix; use dual operation only during migration.

## Consequences

The MVP retains one backup and consistency boundary. The risk is that approximate pgvector indexes require careful evaluation under selective filters and multi-tenant workloads; unfiltered benchmarks are insufficient.

## References

- [Official pgvector documentation](https://github.com/pgvector/pgvector)
- [Official Qdrant multitenancy documentation](https://qdrant.tech/documentation/manage-data/multitenancy/)

## Revisit

Review whenever embedding count grows by an order of magnitude or the retrieval SLO misses its target for two consecutive weeks.

