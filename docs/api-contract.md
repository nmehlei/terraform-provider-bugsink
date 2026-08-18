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
expand the matrix only through acceptance tests.

Sources:

- https://www.bugsink.com/docs/api/
- https://www.bugsink.com/docs/api-documentation/
