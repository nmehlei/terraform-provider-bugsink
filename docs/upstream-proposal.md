# Draft upstream issue: Terraform-friendly lifecycle API

## Context

We would like to build an independent, public Terraform provider for Bugsink.
The canonical API already supports listing, creating, retrieving, and updating
teams and projects, which is an excellent foundation.

## Request

Could Bugsink document or add the following before a provider treats these
objects as Terraform resources?

1. `DELETE` endpoints for projects and teams, including behavior for stored
   events, team members, and non-empty teams.
2. Release lifecycle semantics: delete endpoint or an explicit append-only
   contract.
3. Scoped, revocable API tokens for automation.
4. A compatibility policy for `/api/canonical/0/` and the OpenAPI schema.
5. Create idempotency behavior or support for an idempotency key.

## Proposed collaboration

The provider would be a separately licensed, independent Go project using the
public HTTP API. We would welcome upstream review of API mappings, compatibility
tests, and guidance on permitted trademark/provider namespace use. Would
Bugsink be interested in sponsoring, adopting, or recommending it?
