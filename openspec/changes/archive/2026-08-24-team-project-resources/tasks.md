## 1. Dependencies and scaffold

- [x] 1.1 Add `terraform-plugin-framework`, `terraform-plugin-go`, `terraform-plugin-log` to `go.mod`
- [x] 1.2 Add `terraform-plugin-testing` as a test-only dependency
- [x] 1.3 Create `src/provider/` package
- [x] 1.4 Create `src/main.go` calling `providerserver.Serve`

## 2. Provider configuration

- [x] 2.1 Implement `provider.Provider` (`Metadata`, `Schema`, `Configure`, `Resources`, `DataSources` returning none)
- [x] 2.2 Schema: `endpoint` (string, required), `token` (string, optional, sensitive) — also added `insecure` (bool, optional), not in the original spec text but required for the client's existing `AllowInsecure` gate to be reachable at all (needed by the disposable-instance acceptance tests in section 6)
- [x] 2.3 `Configure`: read `BUGSINK_TOKEN` when `token` unset, fail with actionable diagnostic if both are absent
- [x] 2.4 `Configure`: build `*bugsink.Client` via `bugsink.New`, surface its error (incl. unsupported-version) as a configuration diagnostic
- [x] 2.5 Store the client in `resp.ResourceData` for resources to retrieve

## 3. `bugsink_team` resource

- [x] 3.1 Schema: `id` (computed), `name` (required), `visibility` (optional/computed)
- [x] 3.2 `Create`: call `client.CreateTeam`, write full response to state
- [x] 3.3 `Read`: call `client.GetTeam`, update state (drift detection)
- [x] 3.4 `Update`: call `client.UpdateTeam` with only changed fields, write response to state
- [x] 3.5 `Delete`: `resp.Diagnostics.AddError` with the actionable message from the spec; no state manipulation
- [x] 3.6 `ImportState`: `resource.ImportStatePassthroughID` on `id`

## 4. `bugsink_project` resource

- [x] 4.1 Schema: `id` (computed), `team` (required, `RequiresReplace`), `name` (required), `visibility`, `alert_on_new_issue`, `alert_on_regression`, `alert_on_unmute`, `retention_max_event_count` (all optional/computed), `slug` (computed), `dsn` (computed, sensitive), `grouping_mechanism` (computed)
- [x] 4.2 Shared internal `read` helper used by `Create`/`Update`/`Read` to populate computed fields via `client.GetProject` after every write
- [x] 4.3 `Create`: call `client.CreateProject`, then the shared read helper
- [x] 4.4 `Read`: call the shared read helper directly
- [x] 4.5 `Update`: call `client.UpdateProject` with only changed fields, then the shared read helper
- [x] 4.6 `Delete`: `resp.Diagnostics.AddError` with the actionable message from the spec; no state manipulation
- [x] 4.7 `ImportState`: `resource.ImportStatePassthroughID` on `id`

## 5. Unit tests

- [x] 5.1 Test `Configure` fails with actionable diagnostic when `token` and `BUGSINK_TOKEN` are both absent
- [x] 5.2 Test `Configure` falls back to `BUGSINK_TOKEN` when `token` unset
- [x] 5.3 Test `bugsink_team` and `bugsink_project` `Delete` diagnostics contain the `terraform state rm` guidance and do not remove state on error (framework contract, but assert the diagnostic text itself)
- [x] 5.4 Test `bugsink_project`'s `team` attribute is configured with `RequiresReplace`

## 6. Acceptance tests (against the existing disposable Bugsink 2.5 environment)

- [x] 6.1 `bugsink_team`: create, in-place update (assert no replace), import, drift detection (rename out-of-band via the client, assert next plan shows a diff) — implemented via `terraform-exec` + dev_overrides (see design.md); ran against a live instance and passed
- [x] 6.2 `bugsink_team`: destroy fails with the expected diagnostic; team still exists remotely afterward; `terraform state rm` detaches without deleting remotely — ran against a live instance and passed
- [x] 6.3 `bugsink_project`: create (assert `dsn` is populated after apply, not empty), in-place update of a non-`team` attribute (assert no replace), import — ran against a live instance and passed. **Found and fixed a real bug in the process**: `visibility`/alert flags/`retention_max_event_count` were `Optional+Computed` with no `UseStateForUnknown` plan modifier, so an unconfigured attribute planned as unknown and `Update` sent an empty string to the API on any update, breaking `bugsink_team`'s rename step (`400: "" is not a valid choice`). Fixed on both resources.
- [x] 6.4 `bugsink_project`: changing `team` forces replacement — ran against a live instance and passed (asserted via the plan's `Actions.Replace()`, not applied, to avoid an orphaned project)
- [x] 6.5 `bugsink_project`: drift detection (change `retention_max_event_count` out-of-band via the client, assert next plan shows a diff) — ran against a live instance and passed
- [x] 6.6 `bugsink_project`: destroy fails with the expected diagnostic; project still exists remotely afterward; `terraform state rm` detaches without deleting remotely — ran against a live instance and passed
- [x] 6.7 Superseded: no `TF_ACC` gate needed (design.md - `resource.Test` isn't used). `BUGSINK_ACCEPTANCE=1` alone gates both the client and provider acceptance tests; CI's `acceptance` job already sets it (unchanged from `acceptance-spike`) and now also needs a `terraform` CLI available - see CI update below

## 7. Docs and examples

- [x] 7.1 Add `examples/provider/provider.tf` and `examples/resources/bugsink_team/`, `examples/resources/bugsink_project/` (resource.tf + import.sh)
- [x] 7.2 Wire `tfplugindocs` (generate into `docs/resources/`, `docs/index.md`) - added `tools/tools.go` + a `go:generate` directive in `src/main.go`; actually ran it (required extracting a `terraform providers schema -json` via dev_overrides and remapping the schema key, since `--providers-schema` only auto-matches `hashicorp/*` providers - documented in the generation command). `docs/index.md`, `docs/resources/team.md`, `docs/resources/project.md` are real generated output, not placeholders.
- [x] 7.3 Write the "destroy will fail" caveat prominently in both resources' schema `MarkdownDescription` (shows up at the top of generated docs, not buried) - confirmed in the generated docs/index.md and docs/resources/*.md, it's the first thing after the one-line description
- [x] 7.4 Update root `README.md`'s status/support table now that create/update/import/drift are real, keeping the destroy caveat equally prominent there
