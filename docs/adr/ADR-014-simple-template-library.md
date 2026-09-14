# ADR-014: Simple User-Managed Template Library

- Status: Accepted
- Date: 2026-09-14
- Decision owners: Project owner
- Supersedes: The template-admission and version-management parts of [ADR-006](./ADR-006-template-and-rendering-boundary.md)

## Context

ResumeGPT needs reusable resume and cover-letter templates before generation can be implemented. The earlier proposal required capability manifests, immutable template versions, fixture rendering, approval, and a managed admission lifecycle. That is disproportionate for a personal application and makes uploading a source file difficult to understand.

Users need a small library: keep the provided default resume template, upload a TeX or Word file, label it, inspect it, and select it during a later generation. Template rendering still requires a separate sandbox, but those rendering concerns do not require a complex template-management workflow.

## Decision

Represent each custom template as one mutable record with:

- a name and optional description;
- a document type: `resume` or `cover_letter`;
- a source format: `latex`, `doc`, or `docx`, where LaTeX accepts a single `.tex` file or multi-file `.zip` archive;
- an optional relative LaTeX entry-file path for ZIP archives;
- the original filename and workspace-scoped object reference;
- a workspace-scoped, pre-generated PDF preview object reference;
- extracted text for later LLM and generation workflows; and
- a processing state with actionable error details.

Users create custom templates by uploading `.tex`, `.zip`, `.doc`, or `.docx` files of at most 5 MiB. A ZIP represents a multi-file LaTeX project. Its optional entry path must identify a relative `.tex` file; without one, processing selects root `main.tex`, an unambiguous nested `main.tex`, or the archive's only `.tex` file. Ambiguous archives require explicit entry selection. Uploads use signed object storage. A durable background task reuses the isolated document service to validate the file, scan it for malware, extract text, and convert the source to PDF. A template becomes `ready` only after clean, non-empty text and the PDF object reference have been stored. Users may list, view, download, replace the source, edit metadata, or delete custom templates. There is no review, approval, publishing, or template-version lifecycle.

PDF previews are write-through artifacts. They are generated on initial upload and regenerated only when the source file is replaced; metadata-only edits do not trigger conversion. The preview endpoint reads the stored PDF and never performs synchronous rendering. The built-in template uses a source-content hash as its preview object key, so it is converted once and automatically receives a new cached preview when its embedded source changes.

The application includes one read-only built-in resume template with a stable identifier. It is embedded in the Go binary rather than stored in `docs` or copied into each workspace. Its attribution is part of both the API representation and source header:

- Name: Rezume
- Author: Nanu Panchamurthy
- License: MIT
- Source: <https://www.overleaf.com/latex/templates/rezume/kfrvqywfkwjs>

Generation will read the selected template's extracted content. A generation may snapshot that content when reproducibility is implemented; this does not make the Template module versioned.

## Security Boundary

Malware scanning and file validation happen before an uploaded template becomes ready. LaTeX ZIP extraction rejects absolute and parent-relative paths, symbolic links, encrypted entries, more than 200 files, files larger than 10 MiB, and expanded archives larger than 25 MiB. Preview conversion runs in the isolated document worker: TeX disables shell escape and Word uses headless LibreOffice with bounded input, output, and execution time. Readiness means the source is available for selection and text analysis and that its source preview converted successfully; it does not certify future generated content or replace generation-time validation. Final artifact rendering remains subject to the sandbox guidance in ADR-006.

## Consequences

Template management is direct and consistent with the simplified Profile and Job modules. PostgreSQL holds metadata, LLM-readable text, and preview references, while object storage retains both original files and cached PDFs. The built-in default works without database seeding and always retains its attribution.

Source conversion failures are reported during background processing without introducing an approval workflow. A successful source preview does not prove that every future generated document will satisfy page, font, overflow, or content constraints; those checks belong to the generation/rendering workflow.

## Revisit

Add template versions or capability metadata only if real generation workflows need reproducible historical templates or deterministic preflight checks. Do not add an approval lifecycle for a single-user library.
