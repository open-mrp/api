#!/usr/bin/env bash

# Open an interactive MySQL CLI session to the local core-service database.
# Uses DB_URL from .env (written by make local-db), with the same default as setup-local-db.sh.

set -euo pipefail

RED='\033[0;31m'
NC='\033[0m'

error() { echo -e "${RED}[ERROR]${NC} $1"; }

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

DB_URL="${DB_URL:-mysql://root:Testing123!@localhost:3306/augno}"

if echo "$DB_URL" | grep -qiE 'psdb\.cloud|planetscale|pscale'; then
    error "Refusing to connect: DB_URL points to PlanetScale. Run 'make local-db' first."
    exit 1
fi

if echo "$DB_URL" | grep -qiE '^postgres(ql)?://'; then
    error "Refusing to connect: DB_URL is PostgreSQL. Core service uses MySQL."
    exit 1
fi

if ! echo "$DB_URL" | grep -qiE '^mysql://'; then
    error "DB_URL must start with mysql://."
    exit 1
fi

if ! echo "$DB_URL" | grep -qiE '@(localhost|127\.0\.0\.1|mysql|seed-db)([:/]|$)'; then
    error "Refusing to connect: DB_URL must point to the local Docker MySQL instance."
    error "Run 'make local-db' to start it, or update DB_URL in .env."
    exit 1
fi

if ! command -v mysql >/dev/null 2>&1; then
    error "mysql client is not installed. Run: brew install mysql-client@8.4"
    exit 1
fi

MYSQL_CONN="${DB_URL#mysql://}"
MYSQL_CONN_BASE="${MYSQL_CONN%%\?*}"

MYSQL_USER="$(echo "$MYSQL_CONN_BASE" | sed -E 's/^([^:]+):.*/\1/')"
MYSQL_PASS="$(echo "$MYSQL_CONN_BASE" | sed -E 's/^[^:]+:(.+)@.*/\1/')"
MYSQL_HOSTPART="$(echo "$MYSQL_CONN_BASE" | sed -E 's/^.*@//')"
MYSQL_DB="$(echo "$MYSQL_HOSTPART" | sed -E 's|^[^/]+/||')"
MYSQL_HOSTPORT="$(echo "$MYSQL_HOSTPART" | sed -E 's|/.*||')"

if echo "$MYSQL_HOSTPORT" | grep -qE ':[0-9]+$'; then
    MYSQL_HOST="$(echo "$MYSQL_HOSTPORT" | sed -E 's/:[0-9]+$//')"
    MYSQL_PORT="$(echo "$MYSQL_HOSTPORT" | sed -E 's/.*://')"
else
    MYSQL_HOST="$MYSQL_HOSTPORT"
    MYSQL_PORT="3306"
fi

if [ -z "$MYSQL_HOST" ] || [ -z "$MYSQL_DB" ] || [ -z "$MYSQL_USER" ]; then
    error "DB_URL could not be parsed. Expected: mysql://USER:PASSWORD@HOST[:PORT]/DATABASE"
    exit 1
fi

export MYSQL_PWD="$MYSQL_PASS"

if ! mysql -u"$MYSQL_USER" -h"$MYSQL_HOST" -P"$MYSQL_PORT" --protocol=tcp "$MYSQL_DB" -e "SELECT 1;" >/dev/null 2>&1; then
    error "Failed to connect to database. Is 'make local-db' running?"
    exit 1
fi

exec mysql -u"$MYSQL_USER" -h"$MYSQL_HOST" -P"$MYSQL_PORT" --protocol=tcp "$MYSQL_DB" "$@"
