#!/usr/bin/env bash
# Tears down the disposable Bugsink acceptance environment, leaving no
# persistent state for the next run.
set -euo pipefail
cd "$(dirname "$0")"

docker compose down --volumes --remove-orphans
rm -f .token
