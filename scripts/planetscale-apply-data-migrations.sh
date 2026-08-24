#!/usr/bin/env bash

# planetscale-apply-data-migrations.sh
# Runs inside `pscale connect --execute`: applies shared/db/data-migrations to whatever branch the
# proxy points at. Not meant to be called directly — planetscale-data-migrations.sh invokes it.
#
# Required env:
#   DATABASE_URL       set by pscale connect
#   MIGRATE_SENTINEL   file to touch on success, proving to the caller that this actually finished
# Optional env:
#   MIGRATE_DRY_RUN    when true, report status and apply nothing

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

if [ -z "${DATABASE_URL:-}" ]; then
    echo "DATABASE_URL is not set. This script runs under 'pscale connect --execute'." >&2
    exit 1
fi

# The proxy listens on 127.0.0.1, so this reaches prod through --target prod without the script
# needing a PlanetScale hostname to recognise.
export PS_PROD_DB_URL="$DATABASE_URL"

if [ "${MIGRATE_DRY_RUN:-false}" = "true" ]; then
    ./scripts/migrate.sh data-status --target prod
else
    ./scripts/migrate.sh data-up --target prod --yes
    ./scripts/migrate.sh data-status --target prod
fi

if [ -n "${MIGRATE_SENTINEL:-}" ]; then
    touch "$MIGRATE_SENTINEL"
fi
