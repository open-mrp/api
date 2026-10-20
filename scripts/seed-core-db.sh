#!/usr/bin/env bash

# Seed Core Service Database
# Seeds static types, accounts, users, items, orders, and other sample data
# into the core-service MySQL database.
#
# Usage: ./scripts/seed-core-db.sh
#
# Requires DB_URL to be set (or .env loaded).
#
# Safety: refuses to run against PlanetScale or any non-MySQL database.

set -euo pipefail

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Load .env if present
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

if [ -f ".env" ]; then
    while IFS= read -r line; do
        if [[ -n "$line" && ! "$line" =~ ^[[:space:]]*# ]]; then
            export "$line"
        fi
    done < .env
fi

if [[ $# -gt 0 ]]; then
    error "Unknown argument: $1"
    error "Usage: $0"
    exit 1
fi

# --- Validate DB_URL ---

DB_URL="${DB_URL:-}"

if [ -z "$DB_URL" ]; then
    error "DB_URL is not set. Export it or add it to .env."
    exit 1
fi

# --- Safety checks ---

# Block PlanetScale hosts
if echo "$DB_URL" | grep -qiE 'psdb\.cloud|planetscale|pscale'; then
    error "Refusing to seed: DB_URL points to a PlanetScale database."
    exit 1
fi

# Block PostgreSQL connections (core-service uses MySQL)
if echo "$DB_URL" | grep -qiE '^postgres(ql)?://'; then
    error "Refusing to seed: DB_URL looks like a PostgreSQL connection. Core service uses MySQL."
    exit 1
fi

# Require MySQL scheme
if ! echo "$DB_URL" | grep -qiE '^mysql://'; then
    error "Refusing to seed: DB_URL does not start with mysql://."
    exit 1
fi

# Block production-looking hosts
if echo "$DB_URL" | grep -qiE 'prod|production'; then
    warn "DB_URL contains 'prod' — are you sure this is a development database?"
    read -r -p "Type 'yes' to continue: " CONFIRM
    if [ "$CONFIRM" != "yes" ]; then
        error "Aborted."
        exit 1
    fi
fi

# Restrict to localhost connections only
if ! echo "$DB_URL" | grep -qiE '@(localhost|127\.0\.0\.1|mysql|seed-db):'; then
    error "Refusing to seed: DB_URL must connect to localhost, 127.0.0.1, or a Docker container."
    error "Current host does not match allowed patterns."
    exit 1
fi

# --- Verify mysql client is available ---

if ! command -v mysql &> /dev/null; then
    error "mysql client is not installed. Install MySQL client tools."
    exit 1
fi

# --- Parse connection string ---

# mysql://user:password@host:port/database -> user, password, host, port, database
MYSQL_CONN="${DB_URL#mysql://}"
MYSQL_USER="$(echo "$MYSQL_CONN" | sed -E 's/(.+):(.+)@(.+):([0-9]+)\/(.+)/\1/')"
MYSQL_PASS="$(echo "$MYSQL_CONN" | sed -E 's/(.+):(.+)@(.+):([0-9]+)\/(.+)/\2/')"
MYSQL_HOST="$(echo "$MYSQL_CONN" | sed -E 's/(.+):(.+)@(.+):([0-9]+)\/(.+)/\3/')"
MYSQL_PORT="$(echo "$MYSQL_CONN" | sed -E 's/(.+):(.+)@(.+):([0-9]+)\/(.+)/\4/')"
MYSQL_DB="$(echo "$MYSQL_CONN" | sed -E 's/(.+):(.+)@(.+):([0-9]+)\/(.+)/\5/')"

MYSQL_CMD="mysql -u${MYSQL_USER} -p${MYSQL_PASS} -h${MYSQL_HOST} -P${MYSQL_PORT} --protocol=tcp ${MYSQL_DB}"

# --- Test connection ---

info "Testing database connection..."
if ! $MYSQL_CMD -e "SELECT 1;" &> /dev/null; then
    error "Failed to connect to database. Check DB_URL."
    exit 1
fi
info "Connection successful."

# --- Run seed files ---

SEED_DIR="shared/db/seed"

if [ ! -d "$SEED_DIR" ]; then
    error "Seed directory not found: $SEED_DIR"
    exit 1
fi

# The dev/ files come last and are dev-only: they are exploration data for a human
# poking at the product, not fixtures anything asserts against. The e2e seeder
# deliberately does not load them — its own fixtures are tuned to its tests, and
# production history layered on top would move the numbers they check.
for seed_file in "$SEED_DIR"/*.sql "$SEED_DIR"/dev/*.sql; do
    if [ ! -f "$seed_file" ]; then
        continue
    fi

    filename="$(basename "$seed_file")"
    info "Running $filename..."

    $MYSQL_CMD < "$seed_file"
done

info "Seed complete."
