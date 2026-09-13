# ADR-001: Modular Monolith with Independent Workers

- Status: Proposed
- Date: 2026-09-13
- Decision owners: TBD

## Context

The product is still at the prototype stage, but document parsing, web crawling, LLM generation, and document rendering have different resource profiles and security boundaries. Starting with a full microservice architecture would introduce distributed transactions, contract compatibility, deployment, and troubleshooting overhead. Putting everything in one process would allow untrusted document processing to affect the online API.

## Decision

Use a Go modular-monolith codebase that builds two Go binaries, `api` and `worker`. Run Python document processing and high-risk rendering as separate worker processes or containers.

- Go domain modules must not access another module's internal tables or packages. They collaborate through application interfaces and domain events.
- The API performs only short transactions and task submission; it does not wait synchronously for OCR, crawling, LLM, or rendering work.
- Workers can scale independently while initially sharing PostgreSQL and object storage.
- Python workers do not directly mutate Go-owned aggregates. They return processing results or events, and the owning Go module commits the state transition.
- The render worker runs in a separate, network-isolated, low-privilege sandbox.
- Infrastructure SDKs may appear only in adapters or the composition root, never in domain packages.

## Service Extraction Criteria

Extract a domain service only when at least one of these conditions is supported by sustained operational evidence:

1. The module needs an independent release cadence and coordinated releases repeatedly block delivery.
2. It requires an independent data-governance, regional-residency, or compliance boundary.
3. Its scaling profile differs materially and vertical scaling is no longer economical.
4. Its failures continue to affect unrelated capabilities despite process isolation.
5. Profiling proves the shared process or database is the bottleneck.

Before extraction, define API/event contracts, data ownership, SLOs, migration, and rollback. Two services must not remain long-term writers to the same table.

## Alternatives

- **Full microservices:** Rejected for the prototype because the operational and consistency costs exceed the current benefit.
- **Single-process monolith:** Rejected because parsers, browsers, and TeX/Office renderers need resource and security isolation.
- **Serverless functions:** May be used for individual adapters, but not as the default architecture because of long-running tasks, binary dependencies, and cold starts.

## Consequences

Development and transactions remain simple while risky workloads retain isolation and independent scaling. The team must actively preserve module boundaries through package visibility, architecture tests, contract tests, and review. A shared database is not permission for cross-module coupling.

## Validation

- CI rejects imports of concrete provider SDKs from domain packages.
- Architecture tests enforce allowed module dependency directions.
- API, worker, and render worker can be built, deployed, and stopped independently.
- Restarting the API during task execution does not lose accepted work.

## Revisit

Review at each Beta milestone or after a material change in team or deployment boundaries. Do not split services on a fixed schedule.

