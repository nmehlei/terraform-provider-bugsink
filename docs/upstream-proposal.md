# Upstream issue: Terraform-friendly lifecycle API

Filed as [bugsink/bugsink#495](https://github.com/bugsink/bugsink/issues/495)
on 2026-08-26.

## Context

This provider is built and published
([registry.terraform.io/providers/nmehlei/bugsink](https://registry.terraform.io/providers/nmehlei/bugsink)).
`bugsink_team` and `bugsink_project` support create/read/update/import/drift
detection. What's still missing upstream is a documented `DELETE` — per
AGENTS.md's "Delete-endpoint policy," `Delete` on both resources fails on
purpose with an actionable diagnostic instead of silently removing state,
rather than blocking the provider's existence on this entirely. The issue
below is what would remove that limitation.

## Request

1. `DELETE` endpoints for projects and teams. Specifically, documented
   behavior for:
   - what happens to stored events/issues when a project is deleted,
   - what happens to team members when a team is deleted,
   - whether deleting a non-empty team (one that still owns projects) is
     allowed, and if so what happens to those projects.
2. Release lifecycle semantics: either a delete endpoint, or an explicit,
   documented append-only contract we can rely on (i.e., confirmation that
   releases are intentionally immutable rather than just currently
   unsupported).
3. Scoped, revocable API tokens for automation — at minimum scoped per
   team/project rather than installation-wide, so a provider's default
   credential isn't over-privileged.
4. A documented compatibility/versioning policy for `/api/canonical/0/` and
   its OpenAPI schema, so we know when a schema change is a breaking change
   versus additive.
5. Idempotent create behavior for POST, or an idempotency-key mechanism, so
   retries after a network failure don't create duplicate teams/projects.

## Proposed collaboration

The provider is a separately licensed, independent Go project using the
public HTTP API — not a fork or embedding of Bugsink server code. We'd
welcome upstream review of the API mappings and compatibility tests, and
guidance on permitted trademark/provider-namespace use (not assuming the
`bugsink` Terraform namespace). Open to sponsorship, adoption, or a
recommendation from Bugsink once these gaps are addressed — no expectation
either way.
