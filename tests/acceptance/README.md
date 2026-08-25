# Disposable Bugsink 2.5 acceptance environment

A `docker compose` environment used only by acceptance tests. It is never
shared and never points at a production instance (see AGENTS.md).

## Pinned image

`docker-compose.yml` pins `bugsink/bugsink` by digest:

```
bugsink/bugsink@sha256:b697016950d8aa7d94020def15bc2727e8fc40148f079884ae4cdacf7ea1e9a3
```

This digest was resolved from Docker Hub's `2.5.0` tag (which at the time of
writing is identical to the `2.5`, `2`, and `latest` tags — Bugsink had not
yet published a `2.5.x` patch release). Re-verify before bumping:

```bash
curl -s "https://hub.docker.com/v2/repositories/bugsink/bugsink/tags?page_size=100" \
  | python3 -c "import json,sys; d=json.load(sys.stdin); \
      print([r['name'] for r in d['results'] if r['name'].startswith('2.5')])"
```

Only bump the pinned digest to a newer `2.5.x` patch; a `2.6` or later minor
is a different supported-version decision (see `docs/api-contract.md`'s
compatibility rule) and needs `supportedAPIMajorMinor` in
`src/bugsink/client.go` revisited too.

## Usage

```bash
./up.sh     # start, wait for health, provision a fresh auth token
# token is now in ./.token (gitignored) and BUGSINK_ACCEPTANCE tests can read it
./down.sh   # stop and remove all state
```

`up.sh` provisions the token via `bugsink-manage create_auth_token` inside
the running container. As documented in `docs/implementation-brief.md`,
Bugsink's tokens are currently installation-wide (not scoped per
team/project) — this is exactly the gap tracked in
`docs/upstream-proposal.md`. The token this script creates is scoped only to
this disposable instance's lifetime, never reused across runs.

## What starts

- `db`: Postgres 17, tmpfs-backed data directory (no named volume — nothing
  survives `down.sh` by design).
- `web`: Bugsink 2.5 at `http://localhost:8000`, healthchecked against
  `/health/ready` before `up.sh` returns (`docker compose up --wait`).
