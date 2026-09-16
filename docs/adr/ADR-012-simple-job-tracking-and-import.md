# ADR-012: Simple Job Tracking and Background URL Import

- Status: Accepted
- Date: 2026-09-14
- Decision owners: ResumeGPT maintainers
- Supersedes: [ADR-009](./ADR-009-job-crawling-policy.md)
- Amended by: [ADR-016](./ADR-016-agent-assisted-job-import.md)

## Context

ResumeGPT is currently a personal application. The earlier Job design introduced source records, immutable page snapshots, extracted requirement entities, configurable workflows, history, and reporting before those capabilities had demonstrated value. That model made ordinary tasks—saving a role, fixing imported text, or changing its application status—harder to understand and implement.

The desired experience is direct: paste a public LinkedIn or Indeed URL and let ResumeGPT fill in a Job, paste several URLs for batch import, or enter the same fields manually. Imported data must remain fully editable. Failed acquisition must never prevent manual use.

## Decision

Use one mutable `jobs` row as the authoritative record for both Job details and the current application status.

Each Job contains:

- title, company, location, country, and city;
- work mode and employment type;
- source URL and a Markdown-friendly description;
- one current application status; and
- import state plus an actionable import error.

Manual creation, reading, editing, and deletion use ordinary CRUD operations. The module does not add review, approval, source-version, immutable Job-snapshot, status-history, or separate application workflows.

### URL Import

- A persistent input accepts one public URL, while the batch form accepts up to 50 URLs.
- Automatic acquisition is limited to HTTPS hosts under `linkedin.com` and `indeed.com`.
- The API normalizes and validates the entire batch before queueing work.
- Creating the placeholder Job and its durable task occurs in one PostgreSQL transaction.
- Repeated imports of the same normalized URL in a workspace return the existing Job instead of creating duplicate work.
- The worker prefers schema.org `JobPosting` JSON-LD and uses only limited page metadata as a fallback.
- Successful extraction updates the same Job. Missing required metadata produces `needs_user_action`; it does not create a review workflow.
- Users may edit any field at any time. A manual save makes their input authoritative and prevents a later importer result from overwriting it.

### Acquisition Boundary

- Resolve DNS before connecting and reject loopback, private, link-local, unspecified, and multicast addresses.
- Revalidate the scheme and approved host after redirects.
- Bound redirects, response headers, body size, and total request duration.
- Accept only HTML responses and do not use environment proxies.
- Do not authenticate, reuse browser sessions, bypass CAPTCHA or access controls, submit applications, or mutate the source site.
- Do not add a headless-browser fallback. Pages that are not publicly available return an actionable manual-entry path.
- Treat all downloaded content as untrusted data, never as model or tool instructions.

## Consequences

The everyday workflow is small and understandable, and imported fields require no special editing path. PostgreSQL durable jobs still provide recovery and bounded retries without introducing a separate workflow product.

Some dynamic, authenticated, region-blocked, or changed pages will not import. Users must enter those roles manually. The module intentionally does not retain historical page changes or application-status history. A future generation workflow may copy the current Job fields into its own immutable input record when reproducibility is required.

## Alternatives

- **Manual entry only:** simpler and safer, but removes the convenience of URL import.
- **Immutable source snapshots and extracted requirement entities:** deferred until generation quality or provenance requirements demonstrate a concrete need.
- **Headless browser acquisition:** rejected for now because its security, maintenance, and compliance costs are disproportionate for a personal application.
- **Separate application aggregate and customizable state machine:** deferred until one Job needs multiple applications or workflow customization becomes a real requirement.

## Revisit

Revisit when supported sites offer stable official APIs, when import failure rates justify a dedicated adapter, when multiple applications per Job are required, or when generation reproducibility cannot be satisfied by copying current Job fields at generation time.
