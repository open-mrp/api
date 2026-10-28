#!/bin/bash

# Setup Local Development Databases
# Starts MySQL via Docker Compose, waits for health, and applies migrations.

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

# --- Write .env ---

update_env() {
    local key="$1" value="$2" file="$REPO_ROOT/.env"
    if [ ! -f "$file" ]; then
        echo "$key=$value" > "$file"
    elif grep -q "^$key=" "$file"; then
        local tmp
        tmp="$(mktemp)"
        sed "s|^$key=.*|$key=$value|" "$file" > "$tmp" && mv "$tmp" "$file"
    else
        echo "$key=$value" >> "$file"
    fi
}

update_env "DB_URL" "$DB_URL"
info "Updated .env with DB_URL."

# --- Start containers ---

info "Starting database containers..."
docker compose up -d

# --- Wait for healthy ---

info "Waiting for MySQL to be healthy..."
until docker inspect --format='{{.State.Health.Status}}' openmrp-mysql 2>/dev/null | grep -q "healthy"; do
    sleep 1
done
info "MySQL is ready."

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

# --- Done ---

echo ""
info "Local databases are ready!"
echo ""
echo "  MySQL:      mysql -u root -p'Testing123!' -h 127.0.0.1 openmrp"
echo ""
echo "  To re-seed core data:"
echo "    make seed-core"
echo ""
echo "  To seed Stripe subscription:"
echo "    make seed-stripe"
echo ""
echo "  To tear down:"
echo "    make local-db-down"
