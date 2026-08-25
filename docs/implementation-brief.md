# Implementation brief

## Objective

Publish an Apache-2.0 licensed, community Terraform provider that declaratively
manages Bugsink teams and projects, with a stable provider address and signed
cross-platform releases.

## Current conclusion — create/update yes, delete still blocked

Bugsink 2.5's canonical API provides list, create, retrieve, and partial-update
operations for teams and projects. It does not document delete operations for
teams, projects, or releases.

See AGENTS.md's "Delete-endpoint policy" (amended 2026-08-24):
`bugsink_team` and `bugsink_project` may be implemented with full
create/read/update. `Delete` must fail with an actionable diagnostic
instead of silently removing the resource from state - silently untracking
a resource that still exists remotely is the state/reality drift Terraform
is supposed to prevent, and remains unacceptable regardless of this policy
change. This is a deliberate, documented trade-off, not the ideal end
state: these resources are a poor fit for any workflow that assumes
`terraform destroy` always succeeds (ephemeral environments, CI teardown).
The upstream ask in `docs/upstream-proposal.md` still stands - a real
delete endpoint removes this limitation entirely.

Tokens currently have installation-wide access. That is too broad for the
default automation credential of a public provider.

## Required upstream contract

Before provider resource implementation, obtain upstream agreement on:

1. Delete endpoints for projects and teams, with explicit behavior for project
   data, team membership, and non-empty teams.
2. Delete or explicitly immutable/append-only semantics for releases.
3. Scoped, revocable automation tokens, at least by team/project.
4. A documented compatibility policy for `/api/canonical/0/`.
5. Idempotency behavior for POST creation, or an idempotency-key mechanism.

Open `docs/upstream-proposal.md` as a GitHub issue in `bugsink/bugsink` before
starting the provider. Do not claim upstream endorsement unless it is explicit.

## Candidate v1 scope

- Provider configuration: `endpoint`, `token`.
- Resources: `bugsink_team`, `bugsink_project`.
- Data sources: `bugsink_team`, `bugsink_project` lookups.
- Import by remote ID.
- Computed properties: project slug and DSN. Store DSN as sensitive state.

Exclude issue triage, events, comments, releases, alerts, users, and
sourcemaps from v1.

## Delivery plan

| Milestone | Outcome |
| --- | --- |
| Upstream contract | Delete/auth/versioning decisions agreed and documented |
| Spike | Go HTTP client plus a disposable Bugsink 2.5 acceptance environment |
| MVP | Team/project CRUD, import, drift handling, generated docs, acceptance tests |
| Public v1 | Signed multi-platform releases, Registry publication, compatibility matrix, support policy |

Estimated work after upstream API support: 2–3 person-weeks for MVP and 4–6
person-weeks for a public v1.
