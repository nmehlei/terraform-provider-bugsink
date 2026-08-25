## Why

`docs/implementation-brief.md` blocks Terraform resource CRUD until Bugsink
ships delete endpoints, but the delivery plan's "Spike" milestone (a Go HTTP
client validated against a real Bugsink 2.5 instance) is not blocked and de-risks
the eventual MVP. We need working client code and a disposable test
environment now so the MVP can start immediately once upstream unblocks
delete semantics, instead of starting from zero.

## What Changes

- Add a Go module under `src/` with a hand-written `net/http` Bugsink API
  client (no generated/SDK code) supporting list, create, retrieve, and PATCH
  for teams and projects against `/api/canonical/0/`.
- Client enforces: bearer auth from a token argument (sourced from
  `BUGSINK_TOKEN` or a sensitive provider attribute by the caller, never read
  directly by the client from the environment), a bounded HTTP timeout,
  rejection of plain HTTP except an explicit local-development allowance,
  redaction of the `Authorization` header from any error/diagnostic output,
  and cursor-following pagination (never constructing cursor values).
- Client fetches and validates the remote API schema/version during
  configuration and fails closed on an unsupported version; Bugsink 2.5 is
  the only supported version for this spike.
- Add a disposable `docker-compose` Bugsink 2.5 environment under `tests/`
  for exercising the client against a real instance, isolated from any
  shared or production instance.
- Add unit tests for the client (request construction, redaction, pagination,
  version validation) and one acceptance/smoke test that runs the client
  against the docker-compose instance.
- Explicitly out of scope: no `bugsink_project`/`bugsink_team`/`bugsink_release`
  Terraform resources or data sources, no `provider.go` scaffolding, no
  delete operation of any kind (none exists upstream yet). This is a client
  and environment spike only, per AGENTS.md's "First action" gate.

## Capabilities

### New Capabilities
- `bugsink-api-client`: the Go HTTP client's request/auth/redaction/pagination/version-validation behavior against `/api/canonical/0/`.
- `acceptance-test-environment`: the disposable Bugsink 2.5 docker-compose environment and how tests provision/target it.

### Modified Capabilities
(none — first change in this repo)

## Impact

- New code: `src/` (Go module, `go.mod`, client package), `tests/acceptance/`
  and `tests/unit/` (per `docs/architecture.md`'s intended layout).
- New dependency: Docker/`docker-compose` for local and CI acceptance runs.
- No impact on existing Terraform state, provider surface, or published
  artifacts — nothing here is released or wired into a `provider.go` yet.
