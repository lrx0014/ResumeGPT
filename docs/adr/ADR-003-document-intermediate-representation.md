# ADR-003: Versioned Document Intermediate Representation

- Status: Proposed
- Date: 2026-09-13
- Decision owners: TBD

## Context

Allowing an LLM to directly generate DOCX or TeX mixes facts, content structure, template syntax, and layout. That makes validation, multi-format output, version comparison, and template-injection prevention unreliable.

## Decision

Define format-neutral, versioned JSON intermediate representations (IRs): `ResumeDocument` and `CoverLetterDocument`. The LLM may generate or modify only permitted IR content fields. Renderers combine an IR with an immutable template version to create output files.

The IR includes at least:

- `schema_version`, `document_language`, `paper_size`, and `page_target`.
- Sections, blocks, and bullets with stable IDs rather than array positions as identity.
- `fact_version_ids`, `evidence_ids`, and verification state for each claim.
- A finite set of semantic style tokens such as `emphasis` and `compact`; never arbitrary CSS, TeX, or XML.
- Optional soft page-break hints; the renderer remains responsible for pagination.

Content, template, and rendering configuration are versioned independently:

```text
ArtifactRevision = Content IR + TemplateVersion + RenderProfileVersion
```

Generation inputs reference immutable snapshots. Revisions use domain commands or an allowlisted subset of JSON Patch paths. They cannot modify tenant identifiers, evidence, or system validation fields.

## Schema Evolution

- Use integer major versions such as `resume/v1`.
- Within a major version, add only backward-compatible optional fields.
- A breaking change creates a new major version and a deterministic offline migrator.
- Preserve original revisions; never silently rewrite history during reads.
- APIs return the schema version and renderers declare their supported range.

## Alternatives

- **HTML as the only IR:** Expressive, but weak in document semantics/evidence and unstable for DOCX conversion.
- **TeX as the IR:** Couples the domain to one rendering technology and expands execution risk.
- **LLM-generated variants per format:** Produces inconsistent outputs and is rejected.

## Consequences

One content revision can reliably produce multiple formats and support semantic diffing, claim validation, and rendering regression tests. The cost is maintaining schemas, migrators, and renderer mappings.

## Validation

- Golden IR fixtures render reliably with every supported renderer.
- Test schema validation, migration, and round trips.
- Compare normalized PDF-extracted text with IR content.
- Display a structural diff for each revision and allow restoration of its parent.

## Revisit

Create a new major version only when a new document type cannot be represented by extending the existing block model.

