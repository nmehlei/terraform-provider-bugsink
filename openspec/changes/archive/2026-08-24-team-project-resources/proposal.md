## Why

The client spike (`acceptance-spike`) proved the API client works against a
real Bugsink 2.5 instance, but the repo still has no usable Terraform
provider — nobody can actually manage anything with it. Bugsink still has no
delete endpoint for teams or projects, but per AGENTS.md's amended
"Delete-endpoint policy" (2026-08-24), that no longer blocks create/read/update
resources as long as `Delete` fails loudly instead of silently untracking a
resource that still exists remotely. This change makes the provider usable.

## What Changes

- Add `src/provider/provider.go`: a Terraform Plugin Framework provider
  configured via `endpoint` and a sensitive `token` (falling back to
  `BUGSINK_TOKEN`), wrapping the existing `src/bugsink` client.
- Add `bugsink_team` resource: create/read/update/import. `Delete` returns
  an error diagnostic (Bugsink has no delete endpoint) instructing the user
  to run `terraform state rm` instead of silently removing state.
- Add `bugsink_project` resource: create/read/update/import, sensitive
  computed `dsn`, same hard-fail `Delete` pattern. Populates computed fields
  (`slug`/`dsn`/`grouping_mechanism`) via a read-after-write, since the
  create response doesn't include them (confirmed empirically against a
  live instance, see `docs/api-contract.md`).
- Add `main.go` wiring the provider for the Plugin Framework server.
- Add acceptance tests (`terraform-plugin-testing`) against the existing
  disposable `tests/acceptance/` Bugsink 2.5 environment: create, in-place
  update, import, drift detection, and the hard-fail delete/`state rm` flow.
- Add `examples/` and `tfplugindocs`-generated docs, with the
  destroy-will-fail behavior documented prominently in both resources' docs.
- **Important adoption caveat, not a code break** (no prior release exists):
  `terraform destroy` on these resources is expected to fail by design.
  Anyone adopting these resources needs to know that before they do, so it
  must be prominent in the docs, not a footnote.

## Capabilities

### New Capabilities
- `bugsink-provider`: the Plugin Framework provider itself — configuration,
  auth, and how it's wired to the `src/bugsink` client.
- `bugsink-team-resource`: `bugsink_team`'s create/read/update/import/delete
  behavior.
- `bugsink-project-resource`: `bugsink_project`'s create/read/update/import/delete
  behavior, including computed/sensitive field handling.

### Modified Capabilities
(none — `bugsink-api-client` from `acceptance-spike` is consumed as-is, not changed)

## Impact

- New code: `src/provider/` (provider, both resources), `src/main.go`,
  `tests/acceptance/` additions (provider-level acceptance tests alongside
  the existing client-level ones), `examples/`, generated `docs/resources/`.
- No impact on `src/bugsink`: this change is purely a consumer of that
  package.
- Policy impact already recorded in `AGENTS.md` and
  `docs/implementation-brief.md` (the delete-endpoint policy amendment);
  this change implements what that amendment authorized.
