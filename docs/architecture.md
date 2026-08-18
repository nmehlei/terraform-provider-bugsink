# Proposed architecture

## Technology

- Go, current stable release.
- HashiCorp Terraform Plugin Framework (protocol v6).
- Standard-library `net/http` client with a small handwritten Bugsink API
  client; avoid generated client code until its license and compatibility are
  explicitly agreed.
- `terraform-plugin-testing` acceptance tests and `tfplugindocs` generated
  Registry documentation.

## Intended layout

```text
src/
  main.go
  provider/
    provider.go
    client.go
    team_resource.go
    project_resource.go
    team_data_source.go
    project_data_source.go
tests/
  acceptance/
  unit/
docs/
examples/
.github/workflows/
```

## Resource behavior

Each resource implements create, read, update, delete, and import. Deletion
must call a documented remote delete endpoint. A 404 during read/delete removes
the resource from state; other errors preserve state and return actionable
diagnostics. Retry only documented transient failures, respecting `Retry-After`.

## Release and distribution

Use GoReleaser plus GitHub Actions to build darwin/linux/windows artifacts for
amd64 and arm64, produce checksums, sign releases, generate documentation, and
publish semver tags to the Terraform Registry. The final public namespace and
repository owner require agreement with Bugsink; do not assume use of the
`bugsink` namespace.
