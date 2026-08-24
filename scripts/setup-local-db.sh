#!/bin/bash

# Setup Local Development Databases
# Starts MySQL and PostgreSQL via Docker Compose, waits for health, and applies migrations.

set -euo pipefail

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Ensure we're in the repo root (where docker-compose.yml lives)
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

# --- Connection strings ---

DB_URL="mysql://root:Testing123!@localhost:3306/openmrp"
AGENT_DB_URL="postgres://openmrp@localhost:5432/openmrp_agents?sslmode=disable"

# --- Write .env ---

update_env() {
    local key="$1" value="$2" file="$REPO_ROOT/.env"
    if [ ! -f "$file" ]; then
        echo "$key=$value" > "$file"
    elif grep -q "^$key=" "$file"; then
        sed -i '' "s|^$key=.*|$key=$value|" "$file"
    else
        echo "$key=$value" >> "$file"
    fi
}

update_env "DB_URL" "$DB_URL"
update_env "AGENT_DB_URL" "$AGENT_DB_URL"
info "Updated .env with DB_URL and AGENT_DB_URL."

# --- Start containers ---

info "Starting database containers..."
docker compose up -d

# --- Wait for healthy ---

info "Waiting for MySQL to be healthy..."
until docker inspect --format='{{.State.Health.Status}}' openmrp-mysql 2>/dev/null | grep -q "healthy"; do
    sleep 1
done
info "MySQL is ready."

info "Waiting for PostgreSQL to be healthy..."
until docker inspect --format='{{.State.Health.Status}}' openmrp-postgres 2>/dev/null | grep -q "healthy"; do
    sleep 1
done
info "PostgreSQL is ready."

# --- Check for goose ---

if ! command -v goose &> /dev/null; then
    error "goose is not installed. Run 'make install-tools' first."
    exit 1
fi

# --- Ensure MySQL database exists ---

# MYSQL_DATABASE is only applied when the data volume is first initialized, so reused volumes can be healthy with no schema for goose to connect to.
info "Ensuring core-service MySQL database exists..."
if ! docker exec openmrp-mysql mysql -uroot -p'Testing123!' -e "CREATE DATABASE IF NOT EXISTS openmrp;" >/dev/null 2>&1; then
    error "Failed to ensure core-service MySQL database exists."
    exit 1
fi

# --- Ensure PostgreSQL role and database exist ---

# POSTGRES_USER / POSTGRES_DB are only applied on first volume init. pg_isready reports healthy even when the role or database is missing.
info "Ensuring agent-service PostgreSQL role and database exist..."
PG_ADMIN=""
for candidate in openmrp postgres augno; do
    if docker exec openmrp-postgres psql -U "$candidate" -d postgres -c 'SELECT 1' >/dev/null 2>&1; then
        PG_ADMIN="$candidate"
        break
    fi
done
if [ -z "$PG_ADMIN" ]; then
    error "Could not connect to PostgreSQL as a superuser to create role openmrp."
    exit 1
fi
docker exec -i openmrp-postgres psql -U "$PG_ADMIN" -d postgres -v ON_ERROR_STOP=1 >/dev/null <<'SQL'
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'openmrp') THEN
    CREATE ROLE openmrp WITH LOGIN SUPERUSER;
  END IF;
END
$$;
SQL
if ! docker exec openmrp-postgres psql -U "$PG_ADMIN" -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname = 'openmrp_agents'" | grep -qx 1; then
    docker exec openmrp-postgres psql -U "$PG_ADMIN" -d postgres -v ON_ERROR_STOP=1 \
        -c "CREATE DATABASE openmrp_agents OWNER openmrp;" >/dev/null
fi

# --- Apply core-service MySQL migration ---

info "Applying core-service MySQL migration..."

# MySQL healthcheck can pass before it's fully ready for client connections.
# Retry goose until the connection succeeds.
MAX_RETRIES=10
GOOSE_LOG="$(mktemp)"
for i in $(seq 1 $MAX_RETRIES); do
    if GOOSE_DRIVER=mysql GOOSE_DBSTRING="root:Testing123!@tcp(localhost:3306)/openmrp?parseTime=true" \
        goose -dir shared/db/migrations up >"$GOOSE_LOG" 2>&1; then
        cat "$GOOSE_LOG"
        rm -f "$GOOSE_LOG"
        break
    fi
    if [ "$i" -eq "$MAX_RETRIES" ]; then
        error "Failed to apply core-service migration after $MAX_RETRIES attempts."
        sed 's/^/  /' "$GOOSE_LOG" >&2
        rm -f "$GOOSE_LOG"
        exit 1
    fi
    sleep 2
done

info "Core-service migration complete."

# --- Seed core-service data ---

info "Seeding core-service data..."
./scripts/seed-core-db.sh
info "Core-service seed complete."

# --- Apply agent-service PostgreSQL migration ---

info "Applying agent-service PostgreSQL migration..."
GOOSE_DRIVER=postgres GOOSE_DBSTRING="postgres://openmrp@localhost:5432/openmrp_agents?sslmode=disable" \
    goose -dir services/agent-service/db/migrations up

info "Agent-service migration complete."

# --- Done ---

echo ""
info "Local databases are ready!"
echo ""
echo "  MySQL:      mysql -u root -p'Testing123!' -h 127.0.0.1 openmrp"
echo "  PostgreSQL: psql postgres://openmrp@localhost:5432/openmrp_agents"
echo ""
echo "  To re-seed core data:"
echo "    make seed-core"
echo ""
echo "  To seed Stripe subscription:"
echo "    make seed-stripe"
echo ""
echo "  To tear down:"
echo "    make local-db-down"
