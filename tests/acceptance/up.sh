#!/usr/bin/env bash
# Brings up a disposable Bugsink 2.5 acceptance environment and provisions a
# fresh, run-scoped auth token. Never point this at a shared or production
# instance (AGENTS.md).
set -euo pipefail
cd "$(dirname "$0")"

docker compose up -d --wait

RAW_OUTPUT="$(docker compose exec -T web bugsink-manage create_auth_token)"
# Bugsink's AuthToken.token is validated server-side against
# ^[a-f0-9]{40}$ (bsmain/models.py) - pull exactly that shape out of the
# command's output instead of assuming the token is the last line, so an
# unexpected banner/warning line doesn't get written to .token as if it
# were the credential.
TOKEN="$(grep -Eo '[a-f0-9]{40}' <<< "$RAW_OUTPUT" | head -n1)"
if [[ -z "$TOKEN" ]]; then
  echo "failed to provision an auth token; bugsink-manage create_auth_token printed:" >&2
  echo "$RAW_OUTPUT" >&2
  exit 1
fi

(umask 077 && printf '%s' "$TOKEN" > .token)

echo "Bugsink acceptance environment is up at http://localhost:8000"
echo "Run-scoped auth token written to tests/acceptance/.token (gitignored, 0600)"
