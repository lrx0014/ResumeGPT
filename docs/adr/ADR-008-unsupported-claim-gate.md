# ADR-008: Unsupported-Claim Export Gate

- Status: Proposed
- Date: 2026-09-13
- Decision owners: TBD

## Context

Model self-review alone cannot reliably prevent fabricated CV content. The product must compare generated claims with the exact profile text explicitly saved by the user while preserving the user's ability to export content with a clear warning.

## Decision

Classify each atomic claim in the final document by risk. A high-risk claim unsupported by the saved profile snapshot cannot be exported as `Verified`. The preferred remedy is for the user to add or correct profile content and rerun validation—not merely click “ignore.”

### Risk Levels

- `blocking`: identity, organization, title, dates, education, certification, contact details, money, percentages, and other quantified outcomes.
- `warning`: skill proficiency, scope of responsibility, impact, and causal language.
- `style`: tone, repetition, and keyword density that do not change facts.

### Gate Policy

1. A `blocking` claim references a supporting excerpt in the saved profile snapshot and passes deterministic value checks.
2. Semantic-expansion checks compare the claim with saved profile content. An LLM verifier is one signal, never the sole authority.
3. When a user changes profile content, create a new generation input snapshot and rerun or rebind the claim.
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

The product clearly separates profile-supported and unverified generated content. Explicit unverified export preserves user autonomy, but must never be advertised as verified. Managed customers can choose a stricter policy.

## Validation

- Maintain adversarial tests for numbers, dates, title inflation, causal expansion, and cross-profile leakage.
- Prioritize low false negatives for blocking claims while monitoring false positives and profile-correction rates.
- The download service checks the current validation ID server-side rather than relying on a disabled UI button.

## Revisit

Review when false positives materially reduce completion, new fabrication patterns appear, or the product enters a regulated hiring context.
