# API contract research (Bugsink 2.5)

## Canonical endpoints

| Object | Supported operations | Fields relevant to Terraform |
| --- | --- | --- |
| Team | list, create, retrieve, PATCH | `id`, `name`, `visibility` |
| Project | list, create, retrieve, PATCH | `id`, `team`, `name`, `visibility`, alert flags, `retention_max_event_count`, computed `slug`, `dsn`, counts, grouping mechanism |
| Release | list, create, retrieve | `id`, `project`, `version`, timestamp |

Endpoints are versioned under `/api/canonical/0/`. Use cursor pagination and
follow URLs returned by the server, rather than constructing cursor values.

## Provider configuration

```hcl
provider "bugsink" {
  endpoint = "https://bugsink.example.com"
  # token is sensitive; prefer BUGSINK_TOKEN in CI.
}
```

Use bearer authentication. The provider must normalize endpoint URLs, set a
bounded HTTP timeout, redact Authorization headers in diagnostics, and reject
plain HTTP except for explicit local-development endpoints.

## Terraform state model

- Use Bugsink's remote numeric project ID / UUID team ID as the Terraform ID.
- Make API token sensitive and never persist it in state.
- Treat the DSN as computed+sensitive pending an upstream security statement.
- Read after every create/update; API defaults must be represented correctly.
- Implement import before v1 release.

## Compatibility rules

Fetch and validate the API schema/version during provider configuration or
resource operations. Start with Bugsink 2.5 as the only supported version;
expand the matrix only through acceptance tests. The OpenAPI schema at
`/api/canonical/0/schema/` reports `info.version`; it's served as YAML by
default and only returns JSON when the request sends `Accept:
application/json` explicitly.

## Confirmed against a live instance (Bugsink 2.5.0, `bugsink/bugsink@sha256:b697...`)

These aren't documented anywhere upstream; found by exercising the running
API directly while building the client in `src/bugsink`.

- **Project `id` round-trips as a different JSON type per endpoint.** The
  create response returns it as a JSON string (`"id": "2"`); retrieve and
  list return it as a bare JSON number (`"id": 2`). Team `id` (a UUID) does
  not have this problem — it's a JSON string everywhere. A future
  `bugsink_project` resource's ID handling needs to tolerate both forms;
  `src/bugsink`'s `ProjectID` type does this today.
- **The create response for a project is a strict subset of the retrieve
  response.** `slug`, `dsn`, `grouping_mechanism`, and the event-count
  fields are absent from `POST /projects/`'s response body and only appear
  on `GET`. This is exactly why "read after every create/update" (above) is
  a hard requirement, not just good practice — the DSN a resource needs to
  export genuinely isn't available until the follow-up read.
- **`alert_on_unmute` exists** alongside `alert_on_new_issue` and
  `alert_on_regression` on projects; not previously listed here.
- Confirmed: `AllowInsecure`-style HTTPS enforcement, bearer auth, and PATCH
  partial-update semantics all behave as expected end-to-end.

Sources:

- https://www.bugsink.com/docs/api/
- https://www.bugsink.com/docs/api-documentation/
