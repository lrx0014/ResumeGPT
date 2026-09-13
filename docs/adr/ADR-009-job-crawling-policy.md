# ADR-009: Compliant and Constrained Job Crawling

- Status: Proposed
- Date: 2026-09-13
- Decision owners: TBD

## Context

The product needs to retrieve a job posting from a user-supplied URL. Crawling involves site terms, robots rules, copyright, dynamic pages, authentication/CAPTCHA, SSRF, malicious content, and changing pages.

## Decision

Use a layered acquisition strategy: official/public APIs and dedicated site adapters first, a constrained HTTP fetcher second, and a sandboxed headless browser only as a last resort. Every failure offers manual JD entry as a fallback.

### Compliance Boundary

- Honor RFC 9309 robots directives, site rate limits, and known terms of service.
- Do not bypass authentication, paywalls, CAPTCHA, access controls, or technical protection measures.
- Do not submit applications, send messages, or mutate third-party state.
- Default to no crawling for prohibited or uncertain sites and direct the user to manual entry.
- Each adapter records its owner, allowed paths, request rate, and last compliance review.

### Security Boundary

- Allow only `http` and `https`. Resolve DNS and block loopback, link-local, private, metadata, and internal address ranges; validate again after redirects.
- Limit response size, redirects, total duration, content types, and compression ratio.
- Give the crawler a separate network policy, low-privilege identity, and no business-database credentials.
- Sanitize HTML and attachments. Page text is always untrusted data and can never become model or tool instructions.
- Disable downloads, persistent credentials, and arbitrary extensions in the browser; do not reuse cookies or browser profiles between jobs.

### Data and Updates

- Store normalized body text, acquisition time, canonical URL, content hash, and required provenance.
- Retain raw HTML for a short period. The normalized job snapshot used for generation is immutable.
- Deduplicate by canonical URL and content hash. Create a new snapshot when the page changes rather than overwriting historical generation input.
- Support user corrections and flags for incomplete or stale sources.

## Alternatives

- **Manual input only:** Safest and simplest, but weakens a core experience; retain it as fallback.
- **General browser automation for every site:** Rejected due to excessive cost, security, and compliance risk.
- **Third-party crawling SaaS:** May be an adapter, but remains subject to the same compliance, data, and security policy.

## Consequences

Some URLs will not import automatically, but failure is explainable and recoverable. Dedicated adapters require maintenance, and fixture/contract tests must detect site changes.

## References

- [RFC 9309: Robots Exclusion Protocol](https://www.rfc-editor.org/rfc/rfc9309.html)
- [OWASP SSRF Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html)

## Revisit

Review immediately for new sites, terms changes, complaints, or security events. Review every adapter at least quarterly.

