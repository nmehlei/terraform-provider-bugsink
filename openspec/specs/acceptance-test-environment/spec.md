# acceptance-test-environment Specification

## Purpose

Defines the disposable, isolated Bugsink 2.5 environment that acceptance
tests target, so tests never run against a shared or production instance.

## Requirements

### Requirement: Disposable, isolated instance
Acceptance tests SHALL run against a Bugsink instance started and torn down
by the test environment's own tooling, and SHALL NOT target any
shared or production endpoint.

#### Scenario: Environment is torn down after a run
- **WHEN** an acceptance test run completes (pass or fail)
- **THEN** the docker-compose environment can be torn down leaving no persistent state required for the next run

#### Scenario: No production endpoint reachable by default
- **WHEN** the acceptance test suite runs without additional configuration
- **THEN** it targets only the locally started docker-compose instance, never a remote endpoint

### Requirement: Pinned Bugsink version
The acceptance environment SHALL run the specific Bugsink 2.5 version this
spike targets, pinned by image digest.

#### Scenario: Environment reports expected version
- **WHEN** the docker-compose environment is started
- **THEN** the running Bugsink instance's reported API version matches the pinned Bugsink 2.5 version the client validates against

### Requirement: Test-only credentials
The acceptance environment SHALL provision credentials that exist only for
the lifetime of the test run and SHALL NOT reuse or require any
production credential.

#### Scenario: Fresh token per run
- **WHEN** the acceptance environment starts
- **THEN** it provisions a token scoped to that run, not a credential shared across runs or environments

### Requirement: Not exposed beyond the local machine
The acceptance environment SHALL bind its ports to loopback only, since it
runs with fixed, publicly-known credentials (see Test-only credentials) and
must never be reachable from the network.

#### Scenario: Service port is loopback-bound
- **WHEN** the docker-compose environment is started
- **THEN** the Bugsink web service is reachable only via loopback (127.0.0.1), not from other hosts on the network
