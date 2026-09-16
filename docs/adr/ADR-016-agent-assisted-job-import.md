# ADR-016: Opt-In Agent-Assisted Job Import

- Status: Accepted
- Date: 2026-09-16
- Decision owners: ResumeGPT maintainers
- Amends: The URL Import and Acquisition Boundary sections of [ADR-012](./ADR-012-simple-job-tracking-and-import.md)

## Context

The deterministic Job importer is fast and inexpensive, but it is limited to public LinkedIn and Indeed HTML that exposes useful JSON-LD or metadata. Many career sites render content with JavaScript or hide the full job description behind an expandable control. ResumeGPT users already manage reusable LLM connections and need an explicit way to trade additional latency and model usage for broader extraction capability.

## Decision

Keep deterministic import as the default. Add an `Enable AI assistance` option to Import via URLs. When disabled, the existing LinkedIn/Indeed validation and static parser remain unchanged. When enabled, the batch accepts public HTTPS job pages and requires the user to select one saved LLM connection and model.

AI-assisted import is Agent-first. The worker creates an initial rendered-page snapshot and starts a LangChainGo Job Import Agent immediately. JSON-LD, metadata, visible text, and heuristic candidates are complementary Agent tools rather than a pre-Agent fallback pipeline. The Agent may also expand a visible control or scroll within strict action limits. It must finish by submitting a schema-bound Job record. Missing title or company uses the existing `needs_user_action` path and does not introduce review or version management.

Dynamic rendering runs in a separate Playwright web worker. The service has no database credentials, object-storage credentials, or LLM tokens. Its API accepts one source URL plus a bounded replayable action list and returns a size-limited snapshot. The Go worker retains the durable task, resolves the encrypted model connection, runs the Agent, validates its final output, and persists the editable Job.

## Safety Boundary

- Accept only HTTPS URLs without credentials and only the standard HTTPS port.
- Resolve destinations and reject non-public, loopback, private, link-local, reserved, and special-purpose addresses.
- Revalidate browser requests and the final page URL.
- Bound page load time, content returned to the Agent, Agent iterations, browser actions, and batch size.
- Expose only page inspection, metadata reading, heuristic candidates, visible text, expansion, scrolling, and structured completion tools.
- Do not expose arbitrary network requests, shell access, files, databases, secrets, typing, downloads, application submission, login sessions, CAPTCHA bypass, or access-control bypass.
- Treat every page field as untrusted data. Page instructions cannot alter the Agent policy or select unrelated tools.

The browser worker's application-level DNS validation is defense in depth. Production deployment should additionally enforce egress policy or an SSRF-aware proxy so DNS rebinding cannot rely solely on application checks.

## Consequences

Users deliberately choose whether an import consumes LLM capacity. AI-assisted imports can support a wider range of public dynamic pages and can reveal ordinary expandable content, but they take longer and remain unable to process authentication walls, CAPTCHA, region restrictions, or strong anti-automation controls. Failures preserve the placeholder Job and an actionable manual-edit path.

The system gains a browser image and a network-sensitive service. Keeping it stateless and credential-free limits impact and allows replacement without changing the Job aggregate. The first implementation relaunches a bounded browser render and replays prior actions for each Agent tool call; a short-lived session protocol may replace this only if measured latency warrants the added lifecycle complexity.

## Revisit

Revisit when browser latency requires session reuse, site-specific APIs become available, compliance policy requires per-domain controls, or application-level URL validation must be replaced with a dedicated controlled-egress proxy.
