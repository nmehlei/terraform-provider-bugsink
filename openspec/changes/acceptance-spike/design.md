## Context

See proposal.md - Why. Constraints from `docs/architecture.md` and
`docs/api-contract.md`: standard-library `net/http` only (no generated
client), Terraform Plugin Framework will eventually wrap this client but is
not built yet, `/api/canonical/0/` uses cursor pagination via server-returned
URLs, and Bugsink 2.5 is the only version this spike targets.

## Goals / Non-Goals

**Goals:**
- A `src/` Go module with a client package usable, unmodified, by the future
  provider's resources.
- A `tests/` docker-compose environment that any contributor can bring up
  locally or in CI without touching a shared instance.

**Non-Goals:**
- No `provider.go`, resource, or data-source scaffolding — the client is a
  standalone package with no Plugin Framework dependency yet.
- No delete support of any kind — none exists upstream.
- No multi-version compatibility matrix — pin Bugsink 2.5 only; the matrix
  is future MVP/v1 work per `docs/implementation-brief.md`.

## Decisions

- **Client package location**: `src/bugsink/client.go` (package `bugsink`),
  separate from any future `src/provider/` package, so it can be imported
  without pulling in Plugin Framework types. Mirrors the intended layout in
  `docs/architecture.md`.
- **Config struct over functional options**: a plain `Config{Endpoint,
  Token, Timeout, AllowInsecure}` struct passed to `New(cfg)`. Simpler to
  unit-test than functional options, and the provider layer (not yet built)
  is the only caller — no need for the extensibility functional options buy.
- **Version check as part of `New`**: `New(cfg)` performs the schema/version
  fetch and returns an error immediately on mismatch, rather than lazily on
  first request. Matches the spec's "fail closed at configuration" behavior
  and gives callers one place to handle the error.
- **Never-include over redaction, via a custom error type**: request errors
  wrap an `*APIError` that formats without ever including header values,
  instead of string-scrubbing arbitrary error text for a token that might be
  in it. Deterministic and easy to unit-test (assert the header is
  structurally absent, not "grep the message"). Considered
  logging-middleware-based scrubbing; rejected because it depends on every
  call site remembering to route errors through it.
- **Credentials stripped/refused, not just headers omitted**: initial review
  found the header-omission property alone was insufficient - an endpoint
  URL's own userinfo (`https://user:pass@host/`) still reached error
  messages, and a server-controlled pagination `next` URL could point at any
  host with the bearer token attached. `New` now strips endpoint userinfo
  before it's ever used to build a request URL, and `resolve` refuses to
  follow a pagination URL that targets a different host or embeds its own
  credentials.
- **docker-compose for the acceptance environment**: matches AGENTS.md's
  explicit instruction ("disposable Docker Bugsink instances") and
  `docs/architecture.md`'s `tests/acceptance/` layout. A single
  `docker-compose.yml` pinned to a specific Bugsink 2.5 image tag/digest,
  started with a Makefile/script target that also provisions a fresh
  automation token via the instance's setup flow before tests run.
- **Acceptance test gating**: the acceptance test is skipped unless an
  environment variable (e.g. `BUGSINK_ACCEPTANCE=1`) is set, so `go test
  ./...` stays fast and hermetic by default; CI sets the variable and brings
  up docker-compose first.

## Risks / Trade-offs

- [Bugsink 2.5 image drifts or gets unpublished] → Pin by digest, not just
  tag, and record the pinned digest in `tests/acceptance/README.md`.
- [Docker unavailable in some contributor/CI environments] → Acceptance test
  stays opt-in via the gating env var; unit tests never require Docker.
- [Hand-written client drifts from the real OpenAPI schema over time] → The
  version-validation requirement fails closed on mismatch, surfacing drift
  immediately instead of silently misbehaving.
- [Provisioning a fresh token per run adds startup latency/complexity] →
  Acceptable for a spike; MVP work can revisit once upstream's scoped-token
  proposal (see `docs/upstream-proposal.md`) lands.
