# Terraform Provider for Bugsink

## Mission

Build a public, independent Terraform provider for Bugsink only after its API
supports safe Terraform lifecycle semantics. Use Go and HashiCorp's Terraform
Plugin Framework; do not use SDKv2 for new code.

## First action

Read `docs/implementation-brief.md`, `docs/api-contract.md`, and
`docs/upstream-proposal.md`. Do not implement `bugsink_project`,
`bugsink_team`, or `bugsink_release` as Terraform resources while their delete
operations are unavailable. A no-op delete or state-only removal is not an
acceptable public-provider behavior.

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
