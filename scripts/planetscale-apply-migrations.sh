#!/usr/bin/env bash

# planetscale-apply-migrations.sh
# Runs inside `pscale connect --execute`: applies the goose migrations to whatever branch the proxy is
# pointed at. Not meant to be called directly — planetscale-release-branch.sh invokes it.
#
# DATABASE_URL is set by pscale and points at the local proxy, so the connection is plaintext to
# 127.0.0.1 and needs no branch password.
#
# Required env:
#   DATABASE_URL       set by pscale connect
#   MIGRATE_SENTINEL   file to touch on success, proving to the caller that this actually finished

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

if [ -z "${DATABASE_URL:-}" ]; then
    echo "DATABASE_URL is not set. This script runs under 'pscale connect --execute'." >&2
    exit 1
fi

export PS_BRANCH_DB_URL="$DATABASE_URL"

# A branch freshly cut from prod carries prod's schema with no goose bookkeeping. Without this the
# baseline would read as pending, and applying it drops every table.
./scripts/migrate.sh baseline --target branch

# Prod's schema also includes every migration shipped in prior releases. planetscale-release-branch.sh
# passes those version ids in SHIPPED_MIGRATION_VERSIONS so they are recorded as applied rather than
# replayed against the branch that already carries them (non-idempotent DDL would collide otherwise).
./scripts/migrate.sh mark-shipped --target branch

./scripts/migrate.sh up --target branch
./scripts/migrate.sh status --target branch

if [ -n "${MIGRATE_SENTINEL:-}" ]; then
    touch "$MIGRATE_SENTINEL"
fi
