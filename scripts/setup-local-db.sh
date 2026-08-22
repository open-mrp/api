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

# --- Apply core-service MySQL migration ---

info "Applying core-service MySQL migration..."

# MySQL healthcheck can pass before it's fully ready for client connections.
# Retry goose until the connection succeeds.
MAX_RETRIES=10
for i in $(seq 1 $MAX_RETRIES); do
    if GOOSE_DRIVER=mysql GOOSE_DBSTRING="root:Testing123!@tcp(localhost:3306)/openmrp?parseTime=true" \
        goose -dir shared/db/migrations up 2>/dev/null; then
        break
    fi
    if [ "$i" -eq "$MAX_RETRIES" ]; then
        error "Failed to apply core-service migration after $MAX_RETRIES attempts."
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
