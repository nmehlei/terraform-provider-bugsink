## 1. Go module setup

- [x] 1.1 Initialize Go module under `src/` (current stable Go release, per `docs/architecture.md`)
- [x] 1.2 Create `src/bugsink/` package skeleton (`client.go`, `errors.go`)

## 2. Client core

- [x] 2.1 Implement `Config{Endpoint, Token, Timeout, AllowInsecure}` and `New(cfg) (*Client, error)`
- [x] 2.2 Enforce HTTPS-only endpoints unless `AllowInsecure` is set explicitly
- [x] 2.3 Apply a bounded default timeout to the underlying `http.Client`
- [x] 2.4 Attach `Authorization: Bearer <token>` to every outgoing request
- [x] 2.5 Implement `*APIError` type that never includes header values, wrap all request-error paths in it
- [x] 2.6 Implement schema/version fetch in `New` and fail closed on non-Bugsink-2.5 version

## 3. Team and project operations

- [x] 3.1 Implement list (cursor-following, no manual cursor construction), create, retrieve, PATCH for teams
- [x] 3.2 Implement list (cursor-following), create, retrieve, PATCH for projects
- [x] 3.3 Confirm no delete method exists on the client's public API (spec requirement, not just an omission)

## 4. Unit tests

- [x] 4.1 Test bearer header is set on every request (`httptest.Server`)
- [x] 4.2 Test `Authorization` value is absent from error/diagnostic strings
- [x] 4.3 Test plain-HTTP rejected by default and accepted with `AllowInsecure`
- [x] 4.4 Test timeout aborts a hung request
- [x] 4.5 Test pagination follows a server-provided next-page URL verbatim
- [x] 4.6 Test version validation accepts Bugsink 2.5 and rejects other reported versions
- [x] 4.7 Test PATCH only changes specified fields (mock server asserts partial body, returns merged state)

## 5. Disposable acceptance environment

- [x] 5.1 Add `tests/acceptance/docker-compose.yml` pinned to a specific Bugsink 2.5 image digest
- [x] 5.2 Add a startup script/Makefile target that brings the instance up and provisions a fresh, run-scoped automation token
- [x] 5.3 Add a teardown script/target that leaves no persistent state for the next run
- [x] 5.4 Document the pinned digest and local run instructions in `tests/acceptance/README.md`

## 6. Acceptance test

- [x] 6.1 Add `tests/acceptance/` Go test gated behind `BUGSINK_ACCEPTANCE=1`
- [x] 6.2 Exercise create → retrieve → PATCH → retrieve for a team and a project against the docker-compose instance
- [x] 6.3 Assert the client's version check passes against the running pinned instance

## 7. CI wiring

- [x] 7.1 Add a workflow job (or extend an existing one) that runs unit tests unconditionally
- [x] 7.2 Add a workflow job that brings up docker-compose, sets `BUGSINK_ACCEPTANCE=1`, runs the acceptance test, and tears the environment down
