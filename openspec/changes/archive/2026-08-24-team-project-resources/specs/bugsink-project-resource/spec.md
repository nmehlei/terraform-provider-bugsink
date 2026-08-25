## Purpose

Defines `bugsink_project`'s Terraform lifecycle, including computed and
sensitive field handling and the same hard-fail delete behavior as
`bugsink_team`.

## ADDED Requirements

### Requirement: Create
Applying a `bugsink_project` resource block SHALL create a project via the
API client, associated with the `team` it references.

#### Scenario: First apply creates a project
- **WHEN** a user applies a new `bugsink_project` resource with `team` and `name`
- **THEN** a project is created remotely under that team and its `id` is written to state

### Requirement: Computed fields are populated via read-after-write
`slug`, `dsn`, and `grouping_mechanism` are not present in the create
response (confirmed against a live Bugsink 2.5 instance). The resource
SHALL perform a read immediately after create (and after update) so these
computed fields are correctly populated in state before `apply` completes.

#### Scenario: DSN is present after apply completes
- **WHEN** a user applies a new `bugsink_project` resource
- **THEN** the resulting state includes a non-empty `dsn` once `apply` finishes, even though the create response itself omits it

### Requirement: DSN is sensitive
`dsn` SHALL be marked as a sensitive computed attribute so it does not
appear in plan/apply console output.

#### Scenario: DSN is masked in plan output
- **WHEN** `terraform plan` or `apply` would display the `dsn` attribute
- **THEN** its value is masked in the CLI output

### Requirement: Read reflects remote state, including drift
Refreshing the resource SHALL re-fetch the project from the API and update
state to match, so out-of-band changes are reported as drift on the next
`plan`.

#### Scenario: Out-of-band retention change is detected as drift
- **WHEN** `retention_max_event_count` is changed directly through the API and the user runs `terraform plan`
- **THEN** the plan shows a diff reconciling state with the actual remote value

### Requirement: In-place update
Changing any writable attribute (`name`, `visibility`, the alert flags,
`retention_max_event_count`) on an existing `bugsink_project` SHALL update
the resource in place via partial update, without replacing it. Changing
`team` (moving a project to a different team) is not supported by the
underlying API and MUST force replacement rather than silently failing or
producing an incorrect in-place update.

#### Scenario: Attribute change does not force replacement
- **WHEN** a user changes `retention_max_event_count` on an existing `bugsink_project` and applies
- **THEN** the plan shows an in-place update, not a destroy/create pair

#### Scenario: Changing team forces replacement
- **WHEN** a user changes `team` on an existing `bugsink_project` and plans
- **THEN** the plan shows the resource will be replaced (destroy + create), not silently updated

### Requirement: Import by remote ID
The resource SHALL support `terraform import` using the project's remote ID
(numeric, per the id-shape handling already implemented in the API client).

#### Scenario: Import an existing project
- **WHEN** a user runs `terraform import bugsink_project.example <project-id>`
- **THEN** the resource is added to state with attributes matching the remote project, including `dsn`

### Requirement: Delete fails with an actionable diagnostic instead of removing state
Because Bugsink has no delete endpoint for projects, `Delete` SHALL NOT
silently remove the resource from state. It SHALL return an error
diagnostic that states the remote object cannot be deleted through the API
and that the user must run `terraform state rm` if they want Terraform to
stop managing it (leaving the remote project intact).

#### Scenario: Destroy fails loudly
- **WHEN** a user runs `terraform destroy` (or otherwise plans a delete) on a `bugsink_project` resource
- **THEN** the operation fails with a diagnostic explaining Bugsink has no delete endpoint and how to detach the resource from state with `terraform state rm`
- **AND** the project still exists remotely afterward

#### Scenario: Manual state removal leaves the remote object intact
- **WHEN** a user runs `terraform state rm` on a `bugsink_project` resource
- **THEN** Terraform stops tracking the resource
- **AND** the project is not deleted remotely
