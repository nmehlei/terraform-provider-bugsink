## Purpose

Defines the observable behavior of the hand-written Go client used to talk to
a Bugsink instance's `/api/canonical/0/` API on behalf of the future
Terraform provider.

## ADDED Requirements

### Requirement: Bearer authentication
The client SHALL authenticate every request with a bearer token supplied by
the caller and SHALL NOT read credentials from the environment itself.

#### Scenario: Request carries bearer token
- **WHEN** the client issues any request to `/api/canonical/0/`
- **THEN** the request includes an `Authorization: Bearer <token>` header set from the caller-supplied token

### Requirement: Credential redaction
The client SHALL redact the `Authorization` header value from any error
message, log line, or diagnostic it produces.

#### Scenario: Failed request error omits token
- **WHEN** a request fails (network error or non-2xx response) and the client returns/logs a diagnostic
- **THEN** the diagnostic text does not contain the bearer token value

### Requirement: Endpoint scheme enforcement
The client SHALL reject plain-HTTP endpoints unless the caller explicitly
marks the endpoint as local development.

#### Scenario: Plain HTTP rejected by default
- **WHEN** the client is configured with an `http://` endpoint and no explicit local-development override
- **THEN** the client returns a configuration error and issues no request

#### Scenario: Plain HTTP allowed with explicit override
- **WHEN** the client is configured with an `http://` endpoint and the caller explicitly sets the local-development override
- **THEN** the client accepts the configuration and issues requests

### Requirement: Bounded request timeout
The client SHALL apply a bounded timeout to every HTTP request and SHALL NOT
block indefinitely on a hung connection.

#### Scenario: Request exceeds timeout
- **WHEN** the server does not respond within the configured timeout
- **THEN** the client aborts the request and returns a timeout error

### Requirement: Cursor-following pagination
The client SHALL follow pagination cursor URLs returned by the server and
SHALL NOT construct or guess cursor values itself.

#### Scenario: Listing follows server-provided next-page URL
- **WHEN** a list response includes a next-page cursor URL
- **THEN** the client requests exactly that URL to fetch the next page

### Requirement: API version validation
The client SHALL fetch and validate the remote API schema/version during
configuration and SHALL fail closed when the remote version is not the
supported Bugsink 2.5 version.

#### Scenario: Unsupported version rejected
- **WHEN** the remote instance reports an API version other than the supported Bugsink 2.5 version
- **THEN** the client returns a configuration error and issues no further requests

#### Scenario: Supported version accepted
- **WHEN** the remote instance reports the supported Bugsink 2.5 version
- **THEN** the client completes configuration and accepts subsequent requests

### Requirement: Team and project read/write operations
The client SHALL support list, create, retrieve, and PATCH operations for
teams and projects against `/api/canonical/0/`, and SHALL NOT expose a
delete operation for any resource.

#### Scenario: Create then retrieve a team
- **WHEN** the caller creates a team and then retrieves it by ID
- **THEN** the retrieved team matches the fields that were set on create

#### Scenario: Partial update via PATCH
- **WHEN** the caller PATCHes a subset of a project's mutable fields
- **THEN** only those fields change on subsequent retrieve, and unspecified fields retain their prior values

#### Scenario: No delete operation exposed
- **WHEN** a caller inspects the client's public API for teams, projects, or releases
- **THEN** no delete operation exists
