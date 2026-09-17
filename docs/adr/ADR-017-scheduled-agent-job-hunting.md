# ADR-017: Scheduled Agent Job Hunting

- Status: Accepted
- Date: 2026-09-17
- Decision owners: Project owner

## Context

Users want ResumeGPT to find new roles proactively instead of requiring every opportunity to be entered or imported manually. Search criteria must remain easy to understand, scheduled work must survive restarts, and discovered pages still need the dynamic-page extraction already implemented by the Job Import Agent.

## Decision

Store each Job Hunter as a workspace-scoped scheduled search with explicit role, location, work mode, employment type, experience, keywords, additional instructions, cadence, LLM connection, and model fields. A Hunter may also reference one Profile so the agent can rank openings against the user's background. Users choose a per-run result limit from 1 to 10; omitted values default to 10.

Run each due search as a `job.hunt.v1` durable task. A bounded LangChainGo Hunter Agent must call a scoped public-web search tool before returning no more candidates than the configured per-run limit. When selected, the Profile's text, target role, and language are supplied as untrusted matching context. Searches prioritize individual Indeed and LinkedIn postings and may use job-discovery services such as Google Jobs to locate trustworthy direct posting URLs. The Hunter does not write ready Jobs directly. New normalized URLs are deduplicated and queued as ordinary AI-assisted `job.page.import.v1` tasks, allowing the existing Import Agent to render, expand, inspect, and parse each page.

Hunter imports remain hidden from the Opportunities list until parsing succeeds. A terminal acquisition or extraction failure moves the source URL into a Hunter-specific confirmation inbox instead of leaving an incomplete Job behind. Each Hunter card exposes the pending count. Users can open the original page, add the opportunity manually, or dismiss it. The confirmation inbox is deliberately limited to failed Hunter discoveries and is not a general approval workflow for Jobs.

Support four simple cadence presets: every 6 hours, every 12 hours, daily, and weekly. Do not allow overlapping runs for the same Hunter. Users may also queue an immediate run, pause or resume a schedule, edit all criteria, or delete it. Deleting a Hunter cancels queued Hunter runs but preserves opportunities it already discovered.

Record each Job origin as `manual`, `url_import`, or `hunter` so the Opportunities page can filter them. Deduplication is based on the workspace and normalized source URL, including removal of common marketing-tracking parameters.

## Security and Reliability Boundaries

- Search and page content are untrusted data and never agent instructions.
- The search tool can access only its configured public HTTPS search host and rejects private-network resolution.
- Candidate URLs must be public HTTPS URLs before they enter the import pipeline.
- Search calls, candidates, agent iterations, execution time, retries, and concurrent runs are bounded.
- Profile text is bounded before it is added to the agent context, and a selected Profile must belong to the same workspace.
- Authentication walls, CAPTCHAs, and restricted pages are not bypassed; affected URLs move to the user's confirmation inbox.
- The public search adapter remains replaceable by a supported commercial search API when reliability or quota requirements justify one.

## Consequences

The user gets one simple automation model while ResumeGPT reuses its durable queue, Task Monitor, LLM settings, page-rendering isolation, parsing agent, and opportunity editing flow. Search-engine availability remains an external dependency; failures are visible on the Hunter and in Task Monitor and use bounded durable retries.

## Validation

- Verify schedule creation, pause/resume, immediate runs, and overlap prevention.
- Verify search results cannot inject instructions or introduce non-public URLs.
- Verify duplicate normalized URLs produce only one Opportunity.
- Verify Hunter opportunities remain hidden until parsing succeeds and inaccessible pages move to the Hunter confirmation inbox.
- Verify optional Profile matching and the 1–10 per-run result limit.
- Verify Hunter deletion preserves discovered Opportunities and clears their deleted Hunter reference.
- Verify workspace RLS protects Hunters, scheduled tasks, and discovered Opportunities.
