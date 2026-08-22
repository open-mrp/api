#!/usr/bin/env bash

# Setup E2E Test Databases
# Runs migrations and seeds against the isolated e2e database containers.
# This script expects the e2e docker compose stack to be running with healthy databases.

set -euo pipefail

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; }

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

MYSQL_CONTAINER="openmrp-mysql-e2e"
POSTGRES_CONTAINER="openmrp-postgres-e2e"

MYSQL_CMD=(mysql -uroot -pTesting123! --protocol=tcp openmrp)
PSQL_CMD=(psql -U openmrp -d openmrp_agents)

# --- Helper: extract goose Up section from a migration file ---

extract_up_sql() {
    sed -n '/^-- +goose Up$/,/^-- +goose Down$/{ /^-- +goose Up$/d; /^-- +goose Down$/d; p; }' "$1"
}

run_mysql_sql() {
    local label="$1"
    local log_file
    log_file="$(mktemp)"

    if ! docker exec -i "$MYSQL_CONTAINER" "${MYSQL_CMD[@]}" >"$log_file" 2>&1; then
        error "$label failed."
        sed 's/^/  /' "$log_file" >&2
        rm -f "$log_file"
        return 1
    fi

    rm -f "$log_file"
}

run_postgres_sql() {
    local label="$1"
    local log_file
    log_file="$(mktemp)"

    if ! docker exec -i "$POSTGRES_CONTAINER" "${PSQL_CMD[@]}" >"$log_file" 2>&1; then
        error "$label failed."
        sed 's/^/  /' "$log_file" >&2
        rm -f "$log_file"
        return 1
    fi

    rm -f "$log_file"
}

# --- Wait for containers ---

info "Verifying e2e database containers are running..."

if ! docker inspect "$MYSQL_CONTAINER" &>/dev/null; then
    error "Container $MYSQL_CONTAINER is not running. Run 'make e2e-up' first."
    exit 1
fi

if ! docker inspect "$POSTGRES_CONTAINER" &>/dev/null; then
    error "Container $POSTGRES_CONTAINER is not running. Run 'make e2e-up' first."
    exit 1
fi

# --- Apply MySQL migrations ---

info "Applying core-service MySQL migration..."

MAX_RETRIES=15
for migration_file in shared/db/migrations/*.sql; do
    [ -f "$migration_file" ] || continue
    filename="$(basename "$migration_file")"
    for i in $(seq 1 $MAX_RETRIES); do
        if extract_up_sql "$migration_file" \
            | docker exec -i "$MYSQL_CONTAINER" "${MYSQL_CMD[@]}" >/dev/null 2>&1; then
            break
        fi
        if [ "$i" -eq "$MAX_RETRIES" ]; then
            error "Failed to apply MySQL migration $filename after $MAX_RETRIES attempts."
            exit 1
        fi
        sleep 2
    done
done

info "MySQL migration complete."

# --- Seed MySQL ---

info "Seeding core-service data..."

PLAN_CODE="enterprise"
PLAN_ID="acpl_01seed000enterprise00002"

for seed_file in shared/db/seed/*.sql shared/db/seed/e2e/*.sql; do
    if [ ! -f "$seed_file" ]; then
        continue
    fi

    filename="$(basename "$seed_file")"

    sed -e "s/@plan_code/'$PLAN_CODE'/g" \
        -e "s/@plan_id/'$PLAN_ID'/g" \
        "$seed_file" \
        | run_mysql_sql "MySQL seed $filename"
done

info "Core-service seed complete."

# --- Apply PostgreSQL migrations ---

info "Applying agent-service PostgreSQL migrations..."

for migration_file in services/agent-service/db/migrations/*.sql; do
    if [ ! -f "$migration_file" ]; then
        continue
    fi

    filename="$(basename "$migration_file")"

    extract_up_sql "$migration_file" \
        | run_postgres_sql "PostgreSQL migration $filename"
done

info "Agent-service migration complete."

# --- Seed PostgreSQL ---

info "Seeding agent-service data..."

for seed_file in services/agent-service/db/seeds/*.sql; do
    if [ ! -f "$seed_file" ]; then
        continue
    fi

    filename="$(basename "$seed_file")"

    extract_up_sql "$seed_file" \
        | run_postgres_sql "PostgreSQL seed $filename"
done

info "Agent-service seed complete."

# --- Done ---

echo ""
info "E2E databases are ready!"
