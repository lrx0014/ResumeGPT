# ADR-015: Bounded Multi-Role Generation and Visual Repair

- Status: Accepted
- Date: 2026-09-14
- Decision owners: Project maintainer
- Supersedes: The prohibition on LLM-authored TeX in [ADR-006](./ADR-006-template-and-rendering-boundary.md)

## Context

ResumeGPT now has editable Profiles, Opportunities, user-managed LLM connections, and reusable templates. Generation must combine those inputs without turning a personal application into a workflow-platform project. Different models may be better at writing, LaTeX work, and visual inspection, while local users may prefer one model for cost and simplicity.

## Decision

Each generation is one durable background run with immutable JSON snapshots of the selected Profile, Opportunity, optional Template, and model choices. The default `single` mode maps one connection and model to all roles. The optional `multi` mode selects a writer, document renderer, and visual reviewer independently.

LLM-backed roles use LangChainGo agents rather than a project-specific agent protocol. The writer, template applier, document designer, visual reviewer, and layout polisher run through the standard LangChainGo agent executor and its bounded ReAct `Action`, `Observation`, and `Final Answer` loop. ResumeGPT's encrypted provider connections are exposed through a LangChainGo `llms.Model` adapter. Each agent receives only its scoped LangChainGo tools: immutable generation context for the writer, complete template-source reading followed by sandboxed LaTeX compilation for template application, a design brief followed by sandboxed HTML/CSS-to-PDF rendering for the document designer, and bounded PDF rasterization for the visual reviewer. Artifact storage is also a standard tool but remains processor-controlled because persistence is not an LLM decision. The durable processor remains responsible for ordering, retries, persisted stages, and the repair limit.

Each material result is appended to a simple generation timeline: configuration change, writer draft, rendered PDF and source, reviewer feedback, workflow warning, or user follow-up prompt. A ready run can be queued again with one follow-up instruction; the active rendering Agent receives the grounded draft and current LaTeX or HTML/CSS, and previous timeline entries remain immutable. A completed or failed run can also refresh its Profile, optional Template, and model snapshots and restart from writing while retaining the same append-only history.

The bounded workflow is:

1. The writer produces profile-grounded Markdown content from the saved Profile and Opportunity.
2. With a selected template, the template applier reads the complete project and produces one compatible LaTeX entry file.
3. Without a template, the document designer reads the design brief and produces complete, self-contained, print-ready HTML/CSS. An optional Profile avatar is exposed only through a controlled placeholder.
4. The isolated document worker compiles LaTeX with shell escape disabled or renders HTML/CSS with external and local resources disabled.
5. The document worker rasterizes up to three PDF pages for the reviewer.
6. Deterministic page-count validation and the visual reviewer can request a repair from the renderer.
7. The workflow stops after at most two repairs and either stores a reliable PDF or returns an actionable failure.

If the selected reviewer model explicitly rejects image input, the workflow degrades to the deterministic PDF and page-count checks. It stores the compiled PDF as ready and records a persistent warning that model-based visual QA was skipped. Connection failures, timeouts, malformed reviewer output, and other review errors do not use this fallback.

Ollama responses are streamed and every role has a bounded output-token budget. If a renderer still returns an incomplete source document, ResumeGPT generates a safely escaped basic layout in the active source format. The PDF remains available with a persistent fallback warning.

For a multi-file LaTeX ZIP, only the selected or automatically discovered entry file is replaced; other source and asset files are preserved. When the selected Profile has a PNG or JPEG avatar, generation injects it beside the entry file under a stable `resumegpt-avatar.*` filename. The template applier uses the template's existing portrait macro or slot when one exists and does not invent a new photo layout when it does not. Single-file LaTeX templates are packaged with the same asset convention for compilation. Provider-specific OpenAI, OpenAI-compatible, and Ollama requests remain behind one gateway.

The implementation supports selected LaTeX templates and template-free HTML/CSS design. Word templates remain manageable and previewable, but are not selectable for generation until a structured DOCX renderer can preserve styles reliably.

## Safety and Simplicity

- Profile and Opportunity text are explicitly treated as untrusted data, not instructions.
- Writer prompts forbid unsupported facts.
- Renderer prompts forbid external commands, file writes, network access, and shell escape.
- Compilation retains the isolated, read-only document-worker boundary.
- API tokens are resolved only inside the worker and are never copied into generation snapshots.
- LangChainGo agents and tools operate inside explicit durable stages; they do not replace the job queue with an unbounded agent graph or workflow engine.
- Tools expose only scoped document operations and never grant agents arbitrary shell, filesystem, database, object-store, or network access.

## Consequences

Users get one simple Generate form and can opt into specialized models. Failures remain inspectable by stage, and completed PDFs are stable object-storage artifacts. A text-only model can still produce a downloadable PDF, with the missing visual review made explicit instead of hidden. Visual approval is model-dependent and does not yet replace deterministic clipping, text-loss, or unsupported-claim analysis. LLM-authored LaTeX increases flexibility but requires the existing sandbox and bounded repair policy.

## Revisit

Add structured DOCX rendering, schema-constrained document IR, stronger deterministic PDF checks, cancellation, or a dedicated workflow engine only when the current pipeline demonstrates a concrete need.
