#!/usr/bin/env bash

# Seed Core Service Database
# Seeds static types, plans, accounts, users, items, orders, and other sample data
# into the core-service MySQL database.
#
# Usage: ./scripts/seed-core-db.sh [--plan free|starter|pro|enterprise]
#
# Requires DB_URL to be set (or .env loaded). Default plan is enterprise.
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

# --- Parse arguments ---

PLAN_CODE="enterprise"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --plan)
            PLAN_CODE="${2:-}"
            if [[ -z "$PLAN_CODE" ]]; then
                error "Usage: $0 [--plan free|starter|pro|enterprise]"
                exit 1
            fi
            shift 2
            ;;
        *)
            error "Unknown argument: $1"
            error "Usage: $0 [--plan free|starter|pro|enterprise]"
            exit 1
            ;;
    esac
done

# --- Validate plan code ---

case "$PLAN_CODE" in
    free)       PLAN_ID="acpl_01seed000free00plan000000" ;;
    starter)    PLAN_ID="acpl_01seed000starter0plan000" ;;
    pro)        PLAN_ID="acpl_01seed000pro000plan00000" ;;
    enterprise) PLAN_ID="acpl_01seed000enterprise00002" ;;
    *)
        error "Invalid plan: $PLAN_CODE. Must be one of: free, starter, pro, enterprise"
        exit 1
        ;;
esac

info "Using plan: $PLAN_CODE (ID: $PLAN_ID)"

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

for seed_file in "$SEED_DIR"/*.sql; do
    if [ ! -f "$seed_file" ]; then
        continue
    fi

    filename="$(basename "$seed_file")"
    info "Running $filename..."

    # Substitute plan variables and pipe to mysql
    sed -e "s/@plan_code/'$PLAN_CODE'/g" \
        -e "s/@plan_id/'$PLAN_ID'/g" \
        "$seed_file" \
        | $MYSQL_CMD
done

info "Seed complete."
