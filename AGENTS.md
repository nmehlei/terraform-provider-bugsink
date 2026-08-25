# Terraform Provider for Bugsink

## Mission

Build a public, independent Terraform provider for Bugsink only after its API
supports safe Terraform lifecycle semantics. Use Go and HashiCorp's Terraform
Plugin Framework; do not use SDKv2 for new code.

## First action

Read `docs/implementation-brief.md`, `docs/api-contract.md`, and
`docs/upstream-proposal.md`.

## Delete-endpoint policy (amended 2026-08-24)

Bugsink's canonical API still has no documented delete endpoint for teams,
projects, or releases. The original policy here was to build nothing until
that changed. That's been revised: `bugsink_team` and `bugsink_project` may
be implemented as Terraform resources with full create/read/update, on the
condition that `Delete` never silently removes the resource from state. A
no-op delete or silent state-only removal is still not acceptable — that's
the exact state/reality drift Terraform resources exist to prevent. Instead,
`Delete` MUST fail with a diagnostic that explains the remote object cannot
be deleted through the API and tells the user how to detach it from state
themselves (`terraform state rm`) if they want to stop managing it.

Known consequence, and it must be documented prominently wherever these
resources are documented: any workflow that assumes `terraform destroy`
succeeds unconditionally (ephemeral environments, CI teardown) will get
stuck on these resources. This makes them a poor fit for throwaway
infrastructure and a reasonable fit for long-lived, manually-tended teams
and projects.

`bugsink_release` is unaffected by this change and stays out of v1 scope
regardless (see "Candidate v1 scope" in `docs/implementation-brief.md`).

## Security and testing

- Accept API credentials only through a sensitive provider attribute or
  `BUGSINK_TOKEN`; never log, return, or write them to fixtures.
- Treat DSNs as sensitive state until upstream documents otherwise.
- Use disposable Docker Bugsink instances for acceptance tests. Never point
  tests at a shared or production instance.
- Pin and test every supported Bugsink version. The installed OpenAPI schema is
  authoritative for that version.

## Repository conventions

- Put Go source under `src/` and tests under `tests/` unless the Terraform
  framework requires a conventional location.
- Use semantic versioning, generated Registry documentation, signed releases,
  and a public GitHub release workflow.
- Keep this provider independent of Bugsink server code. Do not copy server
  code or generated artifacts without upstream permission.

## Completion gate for v1

Do not publish until provider resources have tested create/read/update/delete,
import, drift detection, retry/error handling, and acceptance tests.
