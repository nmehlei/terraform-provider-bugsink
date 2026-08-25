## Purpose

Defines `bugsink_team`'s Terraform lifecycle: what create/read/update/import
do, and — critically — what `Delete` does given Bugsink has no delete
endpoint for teams.

## ADDED Requirements

### Requirement: Create
Applying a `bugsink_team` resource block SHALL create a team via the API
client and populate state with the server's representation of it, including
computed fields.

#### Scenario: First apply creates a team
- **WHEN** a user applies a new `bugsink_team` resource with a `name`
- **THEN** a team is created remotely and its `id` is written to state

### Requirement: Read reflects remote state, including drift
Refreshing the resource SHALL re-fetch the team from the API and update
state to match, so that out-of-band changes are reported as drift on the
next `plan`.

#### Scenario: Out-of-band rename is detected as drift
- **WHEN** a team's `name` is changed directly through the API (not through Terraform) and the user runs `terraform plan`
- **THEN** the plan shows a diff reconciling state with the actual remote name

### Requirement: In-place update
Changing `name` or `visibility` on an existing `bugsink_team` resource SHALL
update the resource in place via the API client's partial-update operation,
without replacing (destroying and recreating) it.

#### Scenario: Rename does not force replacement
- **WHEN** a user changes only `name` on an existing `bugsink_team` and applies
- **THEN** the plan shows an in-place update, not a destroy/create pair

### Requirement: Import by remote ID
The resource SHALL support `terraform import` using the team's remote ID.

#### Scenario: Import an existing team
- **WHEN** a user runs `terraform import bugsink_team.example <team-id>`
- **THEN** the resource is added to state with attributes matching the remote team

### Requirement: Delete fails with an actionable diagnostic instead of removing state
Because Bugsink has no delete endpoint for teams, `Delete` SHALL NOT
silently remove the resource from state. It SHALL return an error
diagnostic that states the remote object cannot be deleted through the API
and that the user must run `terraform state rm` if they want Terraform to
stop managing it (leaving the remote team intact).

#### Scenario: Destroy fails loudly
- **WHEN** a user runs `terraform destroy` (or otherwise plans a delete) on a `bugsink_team` resource
- **THEN** the operation fails with a diagnostic explaining Bugsink has no delete endpoint and how to detach the resource from state with `terraform state rm`
- **AND** the team still exists remotely afterward

#### Scenario: Manual state removal leaves the remote object intact
- **WHEN** a user runs `terraform state rm` on a `bugsink_team` resource
- **THEN** Terraform stops tracking the resource
- **AND** the team is not deleted remotely
