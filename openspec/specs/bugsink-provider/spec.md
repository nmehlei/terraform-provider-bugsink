# bugsink-provider Specification

## Purpose

Defines the Terraform Plugin Framework provider's configuration and
authentication behavior: how it's configured, how credentials are sourced
and protected, and how it connects to a Bugsink instance.

## Requirements

### Requirement: Provider configuration
The provider SHALL accept `endpoint` (string) and `token` (sensitive
string) as configuration attributes, matching the shape in
`docs/api-contract.md`.

#### Scenario: Configured via HCL
- **WHEN** a user sets `provider "bugsink" { endpoint = "..." token = "..." }`
- **THEN** the provider configures its API client with those values

### Requirement: Token falls back to environment variable
The provider SHALL read `BUGSINK_TOKEN` from the environment when `token` is
not set in configuration, and SHALL fail configuration with an actionable
diagnostic when neither is present.

#### Scenario: Token from environment
- **WHEN** `token` is omitted from provider configuration and `BUGSINK_TOKEN` is set in the environment
- **THEN** the provider uses the environment variable's value

#### Scenario: No token anywhere
- **WHEN** `token` is omitted from provider configuration and `BUGSINK_TOKEN` is unset
- **THEN** the provider configuration fails with a diagnostic explaining how to supply a token

### Requirement: Token is never exposed
The provider SHALL mark `token` as a sensitive schema attribute and SHALL
NOT log or include it in any diagnostic it produces.

#### Scenario: Token absent from provider diagnostics
- **WHEN** provider configuration fails for a reason unrelated to the token (e.g. an invalid endpoint)
- **THEN** the resulting diagnostic does not contain the token value

### Requirement: Client version/compatibility failure surfaces as configuration error
The provider SHALL surface the underlying client's version-check failure
(unsupported Bugsink API version) as a provider configuration error with an
actionable message, not a generic or swallowed failure.

#### Scenario: Unsupported Bugsink version
- **WHEN** the configured endpoint reports an unsupported API version
- **THEN** `terraform plan`/`apply` fails during provider configuration with a diagnostic naming the unsupported version

### Requirement: Optional plain-HTTP override for local development
The provider SHALL accept an optional `insecure` (boolean) configuration
attribute that permits a plain-HTTP `endpoint`, for local development
against a disposable instance only.

#### Scenario: Plain HTTP endpoint requires the override
- **WHEN** `endpoint` is `http://` and `insecure` is not set
- **THEN** provider configuration fails, matching the underlying client's default HTTPS enforcement

#### Scenario: Plain HTTP endpoint accepted with the override
- **WHEN** `endpoint` is `http://` and `insecure` is `true`
- **THEN** provider configuration succeeds
