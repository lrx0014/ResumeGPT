# ADR-010: Workspace Tenancy and Authorization

- Status: Proposed
- Date: 2026-09-13
- Decision owners: TBD

## Context

Even if the MVP is single-user, future family, consultant, and team sharing affects every table, object path, cache key, vector filter, and audit event. Retrofitting tenant IDs is risky and commonly creates data leaks.

## Decision

Introduce `Workspace` as the security tenant boundary from day one. Registration automatically creates a personal workspace. A profile is a business object inside a workspace, not a security tenant.

### Authorization Model

- `owner`: manage members, billing, deletion, provider policy, and all content.
- `admin`: manage members and content but not transfer ownership or perform selected billing actions.
- `editor`: create and edit profiles, jobs, applications, and artifacts.
- `viewer`: read and download as policy permits.

Use RBAC for stable roles and add resource relationships only where necessary, such as a private profile visible only to its owner or explicitly named members. Do not introduce a general-purpose ABAC policy language in the first release.

### Defense in Depth

- Every tenant-owned table has a non-null `workspace_id`; compound uniqueness includes workspace scope.
- Repository/API code derives workspace context from the authenticated session. A model or client cannot freely select a privileged workspace.
- Application queries always apply scope; PostgreSQL Row-Level Security is a second line of defense.
- The runtime database role is not a superuser, does not have `BYPASSRLS`, and is separate from the table owner. Evaluate `FORCE ROW LEVEL SECURITY`.
- Object keys, cache keys, jobs, outbox events, audit events, and vector payloads include workspace ID.
- If a vector index is introduced, repositories inject its workspace/profile filter; callers cannot remove it.
- Background-job payloads include workspace ID and revalidate resource ownership at execution.

### Invitations and Lifecycle

- Invitation tokens are single-use, short-lived, stored as hashes, and bound to workspace, role, and email.
- A workspace always has at least one owner. Ownership transfer requires reauthentication and audit.
- Workspace deletion enters a recoverable grace period, then cascades across stores and creates a completion record.
- Audit records include actor, workspace, action, resource, result, and trace ID, but not document bodies.

## Alternatives

- **Use user ID as tenant:** Simpler initially, but makes team sharing and migration expensive.
- **Database per tenant:** Strong isolation but excessive operational cost for a small SaaS; retain as a high-compliance enterprise option.
- **Application filtering only:** One omitted condition could leak data and is rejected.

## Consequences

All code carries workspace context from the start, but future collaboration and enterprise functionality avoid a primary-key redesign. RLS is defense in depth, not a substitute for application authorization testing. Administrator and migration paths require separate design.

## References

- [PostgreSQL Row Security Policies](https://www.postgresql.org/docs/current/ddl-rowsecurity.html)
- [Kubernetes multi-tenancy](https://kubernetes.io/docs/concepts/security/multi-tenancy/)

## Validation

- Run two-workspace isolation tests for every repository.
- Test database privileges for runtime, worker, migration, and support roles.
- Perform dedicated security tests for IDOR, cache keys, signed URLs, background jobs, and vector filters if vector retrieval is introduced.

## Revisit

Review when adding enterprise customers, external collaborators, profile-level sharing, or region-specific deployments.
