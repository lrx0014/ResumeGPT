# ADR-013: User-Managed LLM Connections with Per-Generation Model Choice

- Status: Accepted
- Date: 2026-09-14
- Decision owners: ResumeGPT maintainers
- Supersedes: [ADR-007](./ADR-007-llm-data-and-routing-policy.md)

## Context

ResumeGPT is a personal application. The previous proposal introduced an administrator-approved provider capability registry, logical aliases, automatic policy routing, fallback groups, and regional governance before the generation workflow existed. Users currently need a smaller capability: configure reusable cloud or local endpoints safely, verify them, and choose the connection and model explicitly for each Opportunity-specific generation.

Global generation defaults are undesirable because profile, Opportunity, document type, language, page target, template, connection, and model are decisions for one generation run.

## Decision

Add a workspace Settings page that manages multiple named LLM connections. A connection records only:

- execution mode: `cloud` or `local`;
- provider adapter: `openai`, `openai_compatible`, or `ollama`;
- Base URL; and
- an optional encrypted API token.

The Settings page can test a saved connection and discover its available models. It does not select a default connection or model. The generation workflow will require an explicit connection and model for every run and will capture those selections in its immutable input record.

Interface language and theme are workspace preferences. Document type, output language, page target, paper size, template, Profile, and Opportunity remain generation-specific choices.

### Secret Handling

- Encrypt API tokens with AES-GCM before writing them to PostgreSQL.
- Derive the encryption key from the deployment-owned `SETTINGS_ENCRYPTION_KEY`, which must contain at least 32 characters and is required outside development.
- Never return plaintext tokens to the browser. Responses expose only `apiTokenConfigured`.
- A blank token during update preserves the existing secret; clearing it requires an explicit flag.
- Exclude tokens from logs, traces, audit metadata, and outbox payloads.

### Network Boundary

- Cloud connections require HTTPS and resolve only to public addresses.
- Local connections may resolve only to loopback, private, or explicitly local carrier-grade NAT addresses.
- Disable environment proxies and redirects for connection tests.
- Test only the provider's known model-list endpoint with strict timeout, response-header, and response-size limits.

## Consequences

Users can keep an OpenAI connection and a local Ollama connection without re-entering secrets. Generation behavior remains explicit and reproducible. The implementation avoids automatic provider routing and fallback complexity.

The deployment encryption key becomes required secret material. Losing or changing it makes stored API tokens unreadable, so it must be backed up and rotated through a separately designed migration. Model discovery proves reachability and authentication but does not guarantee that a model supports every future structured-generation capability.

## Alternatives

- **One global provider and model:** simpler storage, but forces repeated replacement and creates hidden generation behavior.
- **Store generation defaults in Settings:** rejected because each Opportunity may require different inputs, language, format, and model.
- **Store plaintext tokens:** rejected because database reads and backups would expose provider credentials.
- **Automatic capability routing and fallback:** deferred until measured usage demonstrates that explicit selection is inadequate.

## Revisit

Revisit when users need automatic fallback, multiple workspace administrators, provider-specific capabilities beyond model discovery, embedding endpoints, or managed secret-store integration.
