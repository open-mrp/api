#!/usr/bin/env bash

# planetscale-data-migrations.sh
# Applies the core-service data migrations (shared/db/data-migrations) to the PlanetScale prod branch.
#
# These are backfills — copy a column into another, seed a lookup row, reshape existing rows — and
# they run *after* the schema deploy request has landed and *before* the new service images roll, so
# code that expects a backfilled column never meets an unbackfilled one.
#
# Running DML straight at prod is deliberate and safe: a PlanetScale deploy request diffs schema only,
# so DML written into a schema migration would apply to the dev branch and silently never reach prod.
# Safe migrations blocks DDL, not DML, so this path is open — and is the only one that works.
#
# Required env:
#   PLANETSCALE_SERVICE_TOKEN_ID
#   PLANETSCALE_SERVICE_TOKEN
# Optional env:
#   PS_ORG          default augno-inc
#   PS_DATABASE     default augno_core
#   PS_PROD_BRANCH  default prod
#   DRY_RUN         when true, report pending migrations and apply nothing

set -euo pipefail

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1" >&2; }

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

PS_ORG="${PS_ORG:-augno-inc}"
PS_DATABASE="${PS_DATABASE:-augno_core}"
PS_PROD_BRANCH="${PS_PROD_BRANCH:-prod}"
DML_DIR="shared/db/data-migrations"

for var in PLANETSCALE_SERVICE_TOKEN_ID PLANETSCALE_SERVICE_TOKEN; do
    if [ -z "${!var:-}" ]; then
        error "$var is required."
        exit 1
    fi
done

if [ ! -d "$DML_DIR" ] || [ -z "$(ls -A "$DML_DIR"/*.sql 2>/dev/null)" ]; then
    info "No data migrations to apply."
    exit 0
fi

info "Data migrations present; applying to $PS_DATABASE/$PS_PROD_BRANCH."

SENTINEL="$(mktemp)"
rm -f "$SENTINEL"
export MIGRATE_SENTINEL="$SENTINEL"
export MIGRATE_TARGET_LABEL="prod"
export MIGRATE_DRY_RUN="${DRY_RUN:-false}"

pscale connect "$PS_DATABASE" "$PS_PROD_BRANCH" --org "$PS_ORG" \
    --execute-protocol mysql \
    --execute "$SCRIPT_DIR/planetscale-apply-data-migrations.sh"

# pscale owns the exit status of --execute; the sentinel is the independent proof that goose ran to
# completion rather than a non-zero being swallowed into a clean-looking run.
if [ ! -f "$SENTINEL" ]; then
    error "Data migrations did not complete successfully."
    exit 1
fi
rm -f "$SENTINEL"

info "Data migrations applied."
