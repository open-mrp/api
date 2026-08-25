#!/usr/bin/env bash

# planetscale-postgres-migrate.sh
# Applies the agent-service goose migrations to the PlanetScale Postgres branch.
#
# Postgres has no deploy requests — PlanetScale's own guidance is that "Postgres branches apply DDL
# directly", and `deploy-request`/`connect`/`password` are Vitess-only commands. So unlike the MySQL
# side there is nothing to stage for review: this runs at release time and applies the migrations to
# the production branch, and the release is gated on it succeeding.
#
# Access is a short-lived role created for this run and deleted afterwards, with a TTL as the backstop
# if the run dies before cleanup.
#
# Required env:
#   PLANETSCALE_SERVICE_TOKEN_ID
#   PLANETSCALE_SERVICE_TOKEN
# Optional env:
#   PG_ORG        default augno-inc
#   PG_DATABASE   default agent-db
#   PG_BRANCH     default main
#   MIGRATIONS    schema | data | both   (default schema)
#   DRY_RUN       when true, report pending migrations and apply nothing

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

PG_ORG="${PG_ORG:-augno-inc}"
PG_DATABASE="${PG_DATABASE:-agent-db}"
PG_BRANCH="${PG_BRANCH:-main}"
MIGRATIONS="${MIGRATIONS:-schema}"
DRY_RUN="${DRY_RUN:-false}"

SCHEMA_DIR="services/agent-service/db/migrations"
DATA_DIR="services/agent-service/db/data-migrations"
DATA_TABLE="goose_db_version_data"

for var in PLANETSCALE_SERVICE_TOKEN_ID PLANETSCALE_SERVICE_TOKEN; do
    if [ -z "${!var:-}" ]; then
        error "$var is required."
        exit 1
    fi
done

case "$MIGRATIONS" in
    schema|data|both) ;;
    *)
        error "MIGRATIONS must be schema, data, or both (got '$MIGRATIONS')."
        exit 1
        ;;
esac

if ! command -v goose >/dev/null 2>&1; then
    error "goose is not installed."
    exit 1
fi

pscale_cmd() {
    pscale "$@" --org "$PG_ORG"
}

# --- Short-lived role ---

ROLE_NAME="ci-migrate-${GITHUB_RUN_ID:-local}-${GITHUB_RUN_ATTEMPT:-1}"
ROLE_ID=""

cleanup() {
    if [ -n "$ROLE_ID" ]; then
        info "Deleting migration role $ROLE_NAME"
        pscale_cmd role delete "$PG_DATABASE" "$PG_BRANCH" "$ROLE_ID" --force >/dev/null 2>&1 || \
            warn "Could not delete role $ROLE_ID; it expires on its own TTL."
    fi
}
trap cleanup EXIT

info "Creating a short-lived admin role on $PG_DATABASE/$PG_BRANCH..."

# Under --format json pscale writes its error as a JSON object to stdout and exits non-zero. Left to
# `set -e` the failing assignment would abort with a bare "exit code 2" and the reason — most often
# the service token lacking access to this database — captured and thrown away. Surface it instead.
if ! ROLE_JSON="$(pscale_cmd role create "$PG_DATABASE" "$PG_BRANCH" "$ROLE_NAME" \
    --inherited-roles postgres \
    --ttl 30m \
    --format json)"; then
    error "Could not create the migration role on $PG_DATABASE/$PG_BRANCH:"
    echo "$ROLE_JSON" | jq -r '.error // .issues[].message // .' 2>/dev/null >&2 || echo "$ROLE_JSON" >&2
    exit 1
fi

ROLE_ID="$(echo "$ROLE_JSON" | jq -r '.id // empty')"
DATABASE_URL="$(echo "$ROLE_JSON" | jq -r '.database_url // empty')"

if [ -z "$DATABASE_URL" ]; then
    error "Could not read a connection string from the created role."
    # The role JSON carries a password; print only the field names so a failure here cannot leak it.
    error "Fields returned: $(echo "$ROLE_JSON" | jq -r 'keys | join(", ")' 2>/dev/null || echo unknown)"
    exit 1
fi

# --- Apply ---

run_goose() {
    local dir="$1"
    shift
    local table_args=()
    if [ "$dir" = "$DATA_DIR" ]; then
        table_args=(-table "$DATA_TABLE")
    fi

    GOOSE_DRIVER=postgres GOOSE_DBSTRING="$DATABASE_URL" \
        goose ${table_args[@]+"${table_args[@]}"} -dir "$dir" "$@"
}

apply_dir() {
    local dir="$1" label="$2"

    if [ ! -d "$dir" ] || [ -z "$(ls -A "$dir"/*.sql 2>/dev/null)" ]; then
        info "No $label migrations to apply."
        return 0
    fi

    if [ "$DRY_RUN" = "true" ]; then
        info "Pending $label migrations:"
        run_goose "$dir" status
        return 0
    fi

    info "Applying $label migrations..."
    run_goose "$dir" up
    run_goose "$dir" status
}

if [ "$MIGRATIONS" = "schema" ] || [ "$MIGRATIONS" = "both" ]; then
    apply_dir "$SCHEMA_DIR" "agent-service schema"
fi

if [ "$MIGRATIONS" = "data" ] || [ "$MIGRATIONS" = "both" ]; then
    apply_dir "$DATA_DIR" "agent-service data"
fi

info "Done."
