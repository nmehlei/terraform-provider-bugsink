## Context

See proposal.md - Why, and AGENTS.md's "Delete-endpoint policy" (amended
2026-08-24) for the policy this implements. Builds directly on the
`src/bugsink` client from `acceptance-spike` (already handles auth, HTTPS
enforcement, version-checking, pagination, the `ProjectID` string/number
inconsistency, and read-after-create semantics for computed fields) — this
change adds the Terraform layer on top and does not modify the client.

## Goals / Non-Goals

**Goals:**
- `bugsink_team` and `bugsink_project` usable end-to-end in real Terraform:
  apply, plan (no spurious diffs), update, import, drift detection.
- `terraform destroy` fails loudly and informatively on both resources,
  never silently.

**Non-Goals:**
- `bugsink_release` — stays out of v1 regardless of this change.
- Data sources — not needed now that resources exist.
- Registry publication, GoReleaser, signed releases — separate work, comes
  after these resources are proven via acceptance tests.
- Retry/backoff policy beyond what `src/bugsink` already does — no evidence
  yet of what's worth retrying against a real instance.

## Decisions

- **Terraform Plugin Framework, protocol v6**, per `docs/architecture.md`.
  Add `github.com/hashicorp/terraform-plugin-framework`,
  `terraform-plugin-go`, and `terraform-plugin-log` as real dependencies
  (the module has had zero dependencies until now); add
  `terraform-plugin-testing` as a test-only dependency for the acceptance
  tests in `tests/acceptance/`.
- **Package layout**: `src/provider/provider.go` (the `provider.Provider`
  implementation and `Configure`), `src/provider/team_resource.go`,
  `src/provider/project_resource.go`, and `src/main.go` calling
  `providerserver.Serve`. Mirrors `docs/architecture.md`'s intended layout.
- **Client lifecycle**: `Configure` builds one `*bugsink.Client` via
  `bugsink.New` (which itself performs the version check) and stores it as
  `resp.ResourceData`; both resources retrieve it in their own `Configure`
  hook. A `bugsink.New` failure becomes a provider-configuration diagnostic
  — satisfies the "surfaces as configuration error" spec requirement for
  free, no separate error-translation layer needed.
- **Delete needs no special "don't remove state" logic**: the Plugin
  Framework only removes a resource from state after `Delete` returns
  *without* diagnostics-level errors. So `Delete` simply calls
  `resp.Diagnostics.AddError(...)` with the actionable message and returns
  — the framework's own contract already leaves state untouched on error.
  This was the key design question and it resolves to "do nothing special,"
  which is worth writing down so a future reader doesn't add unnecessary
  state-manipulation code here.
- **Read-after-write for computed project fields**: `Create` and `Update`
  on `bugsink_project` call the client's create/update, then immediately
  call the same internal `read` helper `Read` uses, before returning —
  because `slug`/`dsn`/`grouping_mechanism` aren't in the create/update
  response (confirmed empirically, `docs/api-contract.md`). `bugsink_team`
  doesn't need this: its create response is already complete.
- **`team` change forces replacement**: `docs/api-contract.md` documents no
  "move project to another team" operation, and guessing at one is worse
  than an honest `RequiresReplace()` on that attribute.
- **Import via ID passthrough**: both resources use
  `resource.ImportStatePassthroughID` against their `id` attribute — the
  remote ID is already exactly what a user would supply to `terraform
  import`, no parsing needed.
- **Superseded during implementation — `terraform-exec` + dev_overrides,
  not `terraform-plugin-testing`'s `resource.Test`, and no `TF_ACC`**:
  `resource.Test` always attempts an automatic final `destroy` and treats a
  failing destroy as a fatal test error — exactly what `bugsink_team`'s and
  `bugsink_project`'s `Delete` is supposed to do by design, so the harness
  fights the feature under test. Provider acceptance tests instead build
  the provider binary, point the real `terraform` CLI at it via a
  `dev_overrides` CLI config, and drive `apply`/`plan`/`destroy`/`state rm`
  directly through `github.com/hashicorp/terraform-exec`, giving full
  control over when destroy is expected to fail. `terraform init` is
  skipped entirely — Terraform's own CLI warns that init "may error
  unexpectedly" under dev_overrides, and confirmed empirically to fail for
  exactly that reason; `plan`/`apply` work without it for a
  single-provider, default-local-backend config. Gating stays the existing
  `BUGSINK_ACCEPTANCE=1` convention from `acceptance-spike` (no separate
  `TF_ACC` gate needed since `resource.Test` isn't used). Provider
  acceptance tests live alongside the existing client acceptance test in
  `tests/acceptance/`, reusing the same docker-compose environment and
  `.token` file.
- **Sensitive attributes**: `token` (provider config) and `dsn` (project,
  computed) are marked `Sensitive: true` in their schemas — this is a
  framework-level guarantee (CLI masking, not persisted specially in state,
  which Terraform state files are not encrypted regardless) and matches
  `docs/api-contract.md`'s "treat DSN as computed+sensitive."

## Risks / Trade-offs

- [`terraform destroy` fails by design on these resources] → This is the
  entire point of the policy change, not a bug, but it's a sharp edge for
  anyone who doesn't read the docs first. Mitigation: prominent
  documentation (tasks below), not a code workaround.
- [Drift on computed alert fields Bugsink defaults without the caller
  asking] → `docs/api-contract.md`'s "API defaults must be represented
  correctly" rule means `Read` must reflect whatever the server actually
  has, even on first create — the read-after-write step handles this by
  construction rather than trusting the create response's echoed values.
- [Acceptance tests require Docker + TF_ACC, so most local `go test ./...`
  runs never exercise this code end-to-end] → Same trade-off already
  accepted in `acceptance-spike`; unit-testable pieces (schema validation,
  diagnostic message construction) get plain unit tests independent of the
  acceptance suite.
