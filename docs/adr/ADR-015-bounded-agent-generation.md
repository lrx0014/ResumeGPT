# ADR-015: Bounded Multi-Role Generation and Visual Repair

- Status: Accepted
- Date: 2026-09-14
- Decision owners: Project maintainer
- Supersedes: The prohibition on LLM-authored TeX in [ADR-006](./ADR-006-template-and-rendering-boundary.md)

## Context

ResumeGPT now has editable Profiles, Opportunities, user-managed LLM connections, and reusable templates. Generation must combine those inputs without turning a personal application into a workflow-platform project. Different models may be better at writing, LaTeX work, and visual inspection, while local users may prefer one model for cost and simplicity.

## Decision

Each generation is one durable background run with immutable JSON snapshots of the selected Profile, Opportunity, Template, and model choices. The default `single` mode maps one connection and model to all roles. The optional `multi` mode selects a writer, renderer, and visual reviewer independently.

The bounded workflow is:

1. The writer produces profile-grounded Markdown content from the saved Profile and Opportunity.
2. The renderer produces one complete LaTeX entry file while preserving the selected template's class, macros, and asset references.
3. The isolated document worker compiles it with shell escape disabled and no network access.
4. The document worker rasterizes up to three PDF pages for the reviewer.
5. Deterministic page-count validation and the visual reviewer can request a repair from the renderer.
6. The workflow stops after at most two repairs and either stores a reliable PDF or returns an actionable failure.

If the selected reviewer model explicitly rejects image input, the workflow degrades to the deterministic PDF and page-count checks. It stores the compiled PDF as ready and records a persistent warning that model-based visual QA was skipped. Connection failures, timeouts, malformed reviewer output, and other review errors do not use this fallback.

Ollama responses are streamed and every role has a bounded output-token budget. If a renderer still returns an incomplete LaTeX document, ResumeGPT generates a safely escaped basic LaTeX layout from the grounded Markdown draft. The PDF remains available, while a persistent warning makes it clear that the selected template was not applied.

For a multi-file LaTeX ZIP, only the selected or automatically discovered entry file is replaced; other source and asset files are preserved. Provider-specific OpenAI, OpenAI-compatible, and Ollama requests remain behind one gateway.

The first increment supports LaTeX output. Word templates remain manageable and previewable, but are not selectable for generation until a structured DOCX renderer can preserve styles reliably.

## Safety and Simplicity

- Profile and Opportunity text are explicitly treated as untrusted data, not instructions.
- Writer prompts forbid unsupported facts.
- Renderer prompts forbid external commands, file writes, network access, and shell escape.
- Compilation retains the isolated, read-only document-worker boundary.
- API tokens are resolved only inside the worker and are never copied into generation snapshots.
- Runs use explicit states rather than a general-purpose agent graph or workflow engine.

## Consequences

Users get one simple Generate form and can opt into specialized models. Failures remain inspectable by stage, and completed PDFs are stable object-storage artifacts. A text-only model can still produce a downloadable PDF, with the missing visual review made explicit instead of hidden. Visual approval is model-dependent and does not yet replace deterministic clipping, text-loss, or unsupported-claim analysis. LLM-authored LaTeX increases flexibility but requires the existing sandbox and bounded repair policy.

## Revisit

Add structured DOCX rendering, schema-constrained document IR, stronger deterministic PDF checks, cancellation, or a dedicated workflow engine only when the current pipeline demonstrates a concrete need.
