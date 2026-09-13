# ADR-008: Unsupported-Claim Export Gate

- Status: Proposed
- Date: 2026-09-13
- Decision owners: TBD

## Context

Model self-review alone cannot reliably prevent fabricated CV content. A blanket prohibition on user assertions would also reject true experience whose original evidence is unavailable. The product must distinguish source-supported, user-confirmed, and unverified statements.

## Decision

Classify each atomic claim in the final document by risk. A high-risk unsupported claim cannot be exported as `Verified`. The preferred remedy is for the user to add or confirm a fact, create a new fact version, and rerun validation—not merely click “ignore.”

### Risk Levels

- `blocking`: identity, organization, title, dates, education, certification, contact details, money, percentages, and other quantified outcomes.
- `warning`: skill proficiency, scope of responsibility, impact, and causal language.
- `style`: tone, repetition, and keyword density that do not change facts.

### Gate Policy

1. A `blocking` claim references at least one fact version in an allowed state and passes deterministic value checks.
2. Semantic-expansion checks compare the claim with evidence. An LLM verifier is one signal, never the sole authority.
3. When a user confirms a new fact, record the user, time, original input, and audit event, then regenerate or rebind the claim.
4. A normal user may export with warnings. Unresolved blocking findings allow only an explicitly acknowledged `Unverified` export with an audit record.
5. Managed workspaces may configure a hard block that disables all unverified exports.
6. Do not place a visible watermark on the CV by default, because it would harm its use. Keep verification state in product and download audit records.

### Claim Rules

- Split a bullet containing multiple assertions into atomic claims for validation.
- Compare numbers, dates, and proper nouns deterministically before semantic entailment.
- “Improved significantly” cannot follow from “worked on optimization”; wording cannot turn participation into leadership or correlation into causation.
- Rerun the gate after every manual edit.

## Alternatives

- **Block every unsupported claim:** Safest but overly restricts user autonomy and usability.
- **Warn but always permit verified export:** Undermines the product's trust claim and is rejected.
- **Use only a second LLM as reviewer:** Models can share failure modes and are not sufficient.

## Consequences

The product clearly separates source-supported, user-asserted, and unverified content. Explicit unverified export preserves user autonomy, but must never be advertised as verified. Managed customers can choose a stricter policy.

## Validation

- Maintain adversarial tests for numbers, dates, title inflation, causal expansion, and cross-profile leakage.
- Prioritize low false negatives for blocking claims while monitoring false positives and user-confirmation rates.
- The download service checks the current validation ID server-side rather than relying on a disabled UI button.

## Revisit

Review when false positives materially reduce completion, new fabrication patterns appear, or the product enters a regulated hiring context.

