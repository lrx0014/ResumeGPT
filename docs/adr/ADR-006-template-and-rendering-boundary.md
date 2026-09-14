# ADR-006: Managed Templates and a Sandboxed Rendering Boundary

- Status: Superseded by [ADR-014](./ADR-014-simple-template-library.md) for template management; the sandboxed rendering guidance remains deferred
- Date: 2026-09-13
- Decision owners: TBD

## Context

DOCX and TeX templates are powerful but may contain macros, external links, complex fields, package loading, and code-execution risks. Promising support for arbitrary templates would prevent the prototype from providing reliable layout and security.

## Decision

Initially support only an import-validated, managed template subset. All content first enters the IR defined by ADR-003, then a separate, pinned, network-isolated render worker renders it. PDF is authoritative for layout acceptance and preview. DOCX is an editable deliverable, not the source of pixel-level truth.

### DOCX Boundary

- Support paragraphs, tables, styles, headers/footers, hyperlinks, and defined placeholders or content controls.
- Reject VBA macros, OLE/ActiveX, external templates, external image downloads, dynamic field code, and arbitrary scripts.
- Remove or reject unsupported capabilities during import and return a capability report.
- Convert to PDF with a controlled LibreOffice image whose application and font versions are pinned.

### TeX Boundary

- Prefer official templates initially. User templates enter quarantine and a test-rendering workflow.
- Disable shell escape, network access, host paths, and arbitrary package installation.
- Allowlist packages, commands, and fonts; cap CPU, memory, processes, file size, and compile time.
- Compile in a disposable, low-privilege container with a read-only root filesystem and per-job working directory.

### Template Admission

A template declares its schema version, locale, paper size, supported sections, fonts, and page capabilities. It becomes `approved` only after security scanning, structural validation, fixture rendering, PDF text comparison, and visual checks.

## Visual QA

Deterministic checks detect page count, overflow, clipping, blank pages, missing fonts, lost text, and alignment. A vision model may supplement subjective layout review. Automatic repair is limited to two attempts, after which the user must shorten content or choose another template.

## Alternatives

- **Arbitrary user templates:** Rejected because security and compatibility are unbounded.
- **HTML/PDF only:** Simpler but does not satisfy DOCX/TeX requirements.
- **LLM-authored TeX or DOCX XML:** Rejected because it is difficult to validate and expands the injection surface.

## Consequences

The supported surface is narrower than “any Word or TeX template,” but output can be secure, reliable, and regression-tested. The UI must explain limitations before upload and must not present a rejected template as an unknown error.

## Revisit

Expand capabilities one at a time using real failure samples. Supporting a file extension never implies support for every feature of that format.
