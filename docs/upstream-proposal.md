# Draft upstream issue: Terraform-friendly lifecycle API

Suggested issue title: "Delete/idempotency/token-scoping support needed for a
Terraform provider"

Target repo: `bugsink/bugsink`

## Context

We would like to build an independent, public Terraform provider for Bugsink
(Go, HashiCorp's Terraform Plugin Framework, Apache-2.0). Terraform's
create/read/update/delete/import model requires a safe, documented delete
path for every resource it manages — the canonical API's `/api/canonical/0/`
already covers list, create, retrieve, and partial-update for teams and
projects, which is an excellent foundation, but the gaps below currently
block us from starting resource implementation (state-only removal that
leaves a remote project or team behind is not acceptable in a public
provider).

## Request

Could Bugsink document or add the following before we treat teams, projects,
or releases as Terraform resources?

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

The provider would be a separately licensed, independent Go project using the
public HTTP API — not a fork or embedding of Bugsink server code. We would
welcome upstream review of our API mappings and compatibility tests, and
guidance on permitted trademark/provider-namespace use (we are not assuming
the `bugsink` Terraform namespace). Would Bugsink be interested in
sponsoring, adopting, or recommending it once these gaps are addressed?
