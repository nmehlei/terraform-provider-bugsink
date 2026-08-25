# terraform-provider-bugsink

An independent Terraform provider for [Bugsink](https://www.bugsink.com/).

`bugsink_team` and `bugsink_project` are usable today: create, read, update,
import, and drift detection all work against a real instance. There's a real,
deliberate limitation on delete — see Caveats below before you adopt either
resource.

## What's supported

| Entity | List / Get | Create | Update | Delete | As a Terraform resource |
| --- | --- | --- | --- | --- | --- |
| Team | yes | yes | yes | fails by design (`terraform state rm` to detach) | `bugsink_team` |
| Project | yes | yes | yes | fails by design (`terraform state rm` to detach) | `bugsink_project` |
| Release | not implemented | not implemented | n/a | not possible (no upstream API) | not planned |
| Issues, events, alerts, users, sourcemaps | out of scope for v1 | — | — | — | not planned |

## Using the provider

```hcl
terraform {
  required_providers {
    bugsink = {
      source = "nmehlei/bugsink"
    }
  }
}

provider "bugsink" {
  endpoint = "https://bugsink.example.com"
  # token is sensitive - prefer the BUGSINK_TOKEN environment variable over
  # hardcoding it here.
}

resource "bugsink_team" "platform" {
  name       = "platform"
  visibility = "discoverable"
}

resource "bugsink_project" "checkout" {
  team                      = bugsink_team.platform.id
  name                      = "checkout-service"
  alert_on_new_issue        = true
  retention_max_event_count = 10000
}

output "checkout_dsn" {
  value     = bugsink_project.checkout.dsn
  sensitive = true
}
```

Full docs: [`docs/index.md`](docs/index.md),
[`docs/resources/team.md`](docs/resources/team.md),
[`docs/resources/project.md`](docs/resources/project.md). More examples in
[`examples/`](examples/).

## Caveats

**`terraform destroy` fails by design on `bugsink_team` and
`bugsink_project`.** Bugsink's canonical API (`/api/canonical/0/`) has no
`DELETE` endpoint for teams or projects. Rather than block on that
indefinitely, or silently drop resources from state while they still exist
remotely (the kind of state/reality drift Terraform exists to prevent),
`Delete` on both resources always fails with a diagnostic explaining why,
and how to detach the resource from state yourself:

```
terraform state rm bugsink_team.platform
```

The remote team or project is untouched either way. See each resource's
generated documentation (`docs/resources/`) for the exact message.

Don't use these resources anywhere that assumes `terraform destroy`
succeeds unconditionally — ephemeral environments, CI-provisioned
sandboxes, anything with automated teardown. They're a good fit for
long-lived, manually-tended teams and projects; a poor fit for throwaway
infrastructure. Full reasoning: `AGENTS.md`'s "Delete-endpoint policy" and
`docs/implementation-brief.md`.

Other things worth knowing:

- `bugsink_release` is out of scope for now regardless of the above —
  releases don't have create/update semantics documented either.
- `dsn` is computed and sensitive; it's only populated after `apply`
  finishes (Bugsink's create response doesn't include it - the provider
  reads it back immediately).
- Changing `team` on an existing `bugsink_project` forces replacement:
  Bugsink has no API to move a project between teams.
- Not yet published to the Terraform Registry - see `TASKS.md` for what's
  left before that.

## Developing

```bash
go build ./...
go test ./src/... ./tests/unit/...   # fast, no external deps

# acceptance tests against a real (disposable, local) Bugsink 2.5 - client
# and provider both, driven through a real terraform CLI for the provider:
cd tests/acceptance
./up.sh
BUGSINK_ACCEPTANCE=1 go test -v ./...
./down.sh
```

The acceptance environment is `docker compose`, pinned to a specific Bugsink
image digest — see [`tests/acceptance/README.md`](tests/acceptance/README.md)
for details and how to bump it. Provider acceptance tests need a `terraform`
CLI on `PATH`.

Regenerate the docs after changing a schema's `MarkdownDescription`:

```bash
go generate ./src/...
```

(This needs a `terraform` CLI on `PATH` too; see the `go:generate` directive
in `src/main.go` for how it resolves the provider's own binary via
dev_overrides instead of the Registry.)

## Docs

- [`docs/implementation-brief.md`](docs/implementation-brief.md) — the
  current plan, and the delete-endpoint policy these resources implement.
- [`docs/api-contract.md`](docs/api-contract.md) — what we've confirmed
  about the API, including quirks found by exercising a live instance.
- [`docs/architecture.md`](docs/architecture.md) — intended package layout
  and release process.
- [`docs/upstream-proposal.md`](docs/upstream-proposal.md) — the issue
  drafted for Bugsink (delete endpoints, scoped tokens, compatibility
  policy), not yet filed.
- [`TASKS.md`](TASKS.md) — the running kickoff checklist.

## Contributing

`bugsink_team`/`bugsink_project` create/read/update/import are implemented;
`bugsink_release` stays out of scope, and delete stays a hard failure until
upstream ships a real delete API (see `AGENTS.md`'s delete-endpoint policy
for what would change that). Bug reports and improvements elsewhere are
welcome.

## License

Apache 2.0, see [`LICENSE`](LICENSE). Not affiliated with or endorsed by
Bugsink.
