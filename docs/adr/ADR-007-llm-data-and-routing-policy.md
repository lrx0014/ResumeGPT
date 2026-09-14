# ADR-007: Data-Classification-Driven LLM Routing and Retention

- Status: Proposed
- Date: 2026-09-13
- Decision owners: TBD

## Context

CV data contains identity, contact details, and employment history. Providers, endpoints, regions, and features have different retention, training, logging, and Zero Data Retention properties. A generic fallback could accidentally send a local-only task to a cloud provider.

## Decision

Route requests using three inputs: data classification, user/workspace policy, and a provider capability registry. Model quality and price participate only after data-policy requirements are satisfied.

### Data Classes

- `public`: public job postings and public templates.
- `personal`: ordinary employment history and user instructions.
- `sensitive`: contact details, addresses, identifiers, and non-public employment data.
- `restricted`: data the workspace requires to remain local or within an approved region.

### Provider Capability Registry

For each logical deployment, record provider, model/version, region, allowed data classes, retention, training policy, ZDR or equivalent controls, capabilities, budget, timeout, and fallback group. Administrators approve registry entries; requests cannot construct them dynamically.

### Routing Rules

- `restricted` data may use only local or explicitly approved dedicated environments and never falls back to a public cloud.
- A fallback must belong to an equal or stricter data-policy and regional group.
- Send only the explicitly selected profile and job inputs required for the task; never include unrelated profiles.
- Explicitly disable optional provider storage. Do not assume parameters or semantics are portable across providers.
- Local models also require authentication, TLS or controlled networking, redacted logs, and version registration.
- Evaluate embedding, reranking, vision, and text-generation endpoints independently because each receives user data.

### Product-Side Retention

- The business database retains immutable input references, configuration, and hashes. Raw prompts and responses use a short, configurable retention period based on product needs.
- Production logs and traces exclude document bodies, complete prompts, token content, and signed URLs.
- Deletion fans out to PostgreSQL, object storage, any optional vector storage or caches, and provider-side state that supports deletion.
- Classify PII and validate schemas/claims before persisting model output.

## Alternatives

- **One default cloud provider:** Simple, but incompatible with residency and local-only modes.
- **Everything local:** Strong privacy but potentially inadequate quality, hardware economics, or availability; offer it as a policy rather than a global mandate.
- **Unrestricted automatic fallback:** More available but may violate user intent and is rejected.

## Consequences

Provider switching becomes auditable and fallback cannot bypass privacy policy. The cost is maintaining the capability registry, reviewing provider terms, and contract-testing policy combinations.

## References

- [OpenAI API data controls](https://platform.openai.com/docs/models/default-usage-policies-by-endpoint)

## Revisit

Review immediately when provider terms, endpoint behavior, regions, or compliance requirements change, and audit the registry at least quarterly.
