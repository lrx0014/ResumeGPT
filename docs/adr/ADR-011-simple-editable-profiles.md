# ADR-011: Simple Editable Profiles

- Status: Accepted
- Date: 2026-09-14
- Supersedes: [ADR-002](./ADR-002-profile-facts-and-evidence.md), [ADR-004](./ADR-004-vector-store.md)
- Decision owners: Project owner

## Context

ResumeGPT is currently a personal application. The source, segment, fact, evidence-link, review-state, and fact-version model introduced substantial product and implementation complexity before generation had demonstrated a need for it. Users primarily need a clear place to maintain reusable professional background text.

Users also need document import, but extraction must not silently replace their saved profile. They should see and edit extracted text before accepting it.

## Decision

Treat each profile as one simple, editable aggregate. A user may own multiple profiles, and each profile stores:

- A name.
- An optional target role.
- A primary content language.
- One UTF-8, Markdown-friendly text body of at most 1 MiB.
- An optional workspace-scoped avatar object identifier.

PostgreSQL is the authoritative store for profile metadata and text. Later LLM workflows read the selected profile's saved text directly. Embeddings or a vector index may be added as rebuildable derived data only after retrieval requirements justify them.

Document import is a separate preparation flow:

1. The browser stages a supported document in workspace-scoped object storage.
2. A durable job sends it to the isolated malware-scanning extraction service.
3. The extraction result is stored on the upload record.
4. The browser loads the extracted text into the profile editor.
5. The profile changes only when the user explicitly saves it.

Profiles support create, read, update, and delete operations. The Profile module does not implement source registries, structured facts, review states, confirmation workflows, or fact version history.

## Consequences

The profile model and UI have one obvious source of truth and are easier to understand, operate, and evolve. Users can paste text directly or use the same editor to correct imported content. Generation receives exactly the content last saved by the user.

The application no longer provides statement-level provenance, confirmation state, or automatic conflict detection. If later product evidence shows those features are necessary, they should be introduced as derived analysis or a separate workflow instead of complicating the editable profile aggregate by default.

Avatar and staged document objects can outlive an abandoned edit or deleted profile. Object lifecycle cleanup is an operational retention concern and must not make profile updates depend on object deletion.

## Migration

Existing profile `domain` values become `target_role`. Existing descriptions and imported source text are combined into `content`. The source, segment, fact, evidence-link, and fact-version tables are then removed. Existing completed document uploads lose their former source reference; new extraction results are stored in `document_uploads.extracted_text`.

## Revisit

Revisit only when measured generation quality or large-profile retrieval latency shows that direct use of saved profile text is insufficient. Any future index must remain workspace scoped, rebuildable, and subordinate to PostgreSQL profile content.
