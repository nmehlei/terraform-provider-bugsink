# bugsink-api-client Specification

## Purpose

Defines the observable behavior of the hand-written Go client used to talk to
a Bugsink instance's `/api/canonical/0/` API on behalf of the Terraform
provider.

## Requirements

### Requirement: Bearer authentication
The client SHALL authenticate every request with a bearer token supplied by
the caller and SHALL NOT read credentials from the environment itself.

#### Scenario: Request carries bearer token
- **WHEN** the client issues any request to `/api/canonical/0/`
- **THEN** the request includes an `Authorization: Bearer <token>` header set from the caller-supplied token

### Requirement: Credentials never appear in errors or diagnostics
The client SHALL NOT include request headers, the bearer token, or any
credentials embedded in the configured endpoint (e.g. `https://user:pass@host/`
userinfo) in any error message, log line, or diagnostic it produces.

#### Scenario: Failed request error omits token
- **WHEN** a request fails (network error or non-2xx response) and the client returns/logs a diagnostic
- **THEN** the diagnostic text does not contain the bearer token value

#### Scenario: Endpoint credentials omitted from errors
- **WHEN** the configured endpoint contains embedded userinfo
- **THEN** no diagnostic the client produces contains that userinfo

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
block indefinitely on a hung connection, regardless of whether the caller
supplies its own underlying HTTP client.

#### Scenario: Request exceeds timeout
- **WHEN** the server does not respond within the configured timeout
- **THEN** the client aborts the request and returns a timeout error

### Requirement: Cursor-following pagination, restricted to the configured host
The client SHALL follow pagination cursor URLs returned by the server and
SHALL NOT construct or guess cursor values itself. It SHALL refuse to follow
a pagination URL that targets a different host than the configured endpoint
or that embeds its own credentials, since blindly following a
server-controlled URL would let a compromised or misconfigured Bugsink
instance redirect the client's bearer token to an arbitrary host.

#### Scenario: Listing follows server-provided next-page URL
- **WHEN** a list response includes a next-page cursor URL on the same host as the configured endpoint
- **THEN** the client requests exactly that URL to fetch the next page

#### Scenario: Cross-host pagination refused
- **WHEN** a list response's next-page URL targets a different host than the configured endpoint
- **THEN** the client returns a configuration error and does not request that URL

#### Scenario: Pagination URL with embedded credentials refused
- **WHEN** a list response's next-page URL embeds its own credentials
- **THEN** the client returns a configuration error and does not request that URL

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

### Requirement: Project ID tolerates both wire representations
Bugsink's project `id` is a JSON string on create and a bare JSON number on
retrieve/list (confirmed empirically). The client SHALL accept either form
and expose it as a string-like value to callers.

#### Scenario: Retrieve after create
- **WHEN** a project created via the client is subsequently retrieved
- **THEN** the client successfully decodes the id from both the create response's string form and the retrieve response's numeric form
