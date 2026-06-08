#!/usr/bin/env bash

# Normalize request_log.actor_id and audit_event.actor_id to the raw user_id
# (us_…) for identity_type='user' rows. Production data is mixed: some user rows
# already store the user_id, others store the account_user.id (acus_…) written by
# older code. The API now exposes the user_id directly, so this rewrites the
# acus_ rows to their user_id. API-key / internal / system / NULL identity rows
# are left untouched (their actor_id is already the exposed value).
#
# account_user.id is a globally-unique primary key, so the join needs no account
# scoping: actor_id = account_user.id pins the exact account_user, and we set
# actor_id to that row's user_id.
#
# Safe to run online and repeatedly: updates are batched (LIMIT) and idempotent —
# once a row's actor_id is a user_id it no longer matches au.id, so a second run
# reports 0 rows. Run AFTER deploying the code that reads actor_id directly.
#
# Usage:
#   DB_URL=mysql://user:pass@host:port/db ./scripts/backfill-actor-ids.sh [batchSize] [--dry-run] [--force]
#     batchSize  rows per UPDATE (default 2000)
#     --dry-run  report how many rows WOULD convert, then exit without writing
#     --force    skip the interactive confirmation (for non-interactive prod jobs)
#
# Local/Docker MySQL runs with no extra flags. To run against a remote / managed
# (PlanetScale) database — i.e. PRODUCTION — set ALLOW_REMOTE=1 explicitly:
#   ALLOW_REMOTE=1 DB_URL=mysql://…@…psdb.cloud/db ./scripts/backfill-actor-ids.sh

set -euo pipefail

GREEN='\033[0;32m'; RED='\033[0;31m'; YELLOW='\033[0;33m'; NC='\033[0m'
info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
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

BATCH_SIZE=2000
DRY_RUN=0
FORCE=0
for arg in "$@"; do
    case "$arg" in
        --dry-run) DRY_RUN=1 ;;
        --force)   FORCE=1 ;;
        ''|*[!0-9]*) error "Unknown argument: $arg"; exit 1 ;;
        *) BATCH_SIZE="$arg" ;;
    esac
done

DB_URL="${DB_URL:-}"
ALLOW_REMOTE="${ALLOW_REMOTE:-0}"

if [ -z "$DB_URL" ]; then
    error "DB_URL is not set. Export it or add it to .env."
    exit 1
fi
if ! echo "$DB_URL" | grep -qiE '^mysql://'; then
    error "DB_URL does not start with mysql://."
    exit 1
fi

# Local hosts run freely. Anything else (remote / managed / PlanetScale = prod)
# requires an explicit ALLOW_REMOTE=1 opt-in so this can't fire at a real database
# by accident.
IS_LOCAL=0
if echo "$DB_URL" | grep -qiE '@(localhost|127\.0\.0\.1|mysql|seed-db|[a-z0-9_-]*mysql[a-z0-9_-]*):'; then
    IS_LOCAL=1
fi
if [ "$IS_LOCAL" -ne 1 ] && [ "$ALLOW_REMOTE" != "1" ]; then
    error "DB_URL points at a non-local database. To migrate production data, re-run with ALLOW_REMOTE=1 (and review --dry-run output first)."
    exit 1
fi

if ! command -v mysql &> /dev/null; then
    error "mysql client is not installed."
    exit 1
fi

MYSQL_CONN="${DB_URL#mysql://}"
MYSQL_USER="$(echo "$MYSQL_CONN" | sed -E 's/(.+):(.+)@(.+):([0-9]+)\/(.+)/\1/')"
MYSQL_PASS="$(echo "$MYSQL_CONN" | sed -E 's/(.+):(.+)@(.+):([0-9]+)\/(.+)/\2/')"
MYSQL_HOST="$(echo "$MYSQL_CONN" | sed -E 's/(.+):(.+)@(.+):([0-9]+)\/(.+)/\3/')"
MYSQL_PORT="$(echo "$MYSQL_CONN" | sed -E 's/(.+):(.+)@(.+):([0-9]+)\/(.+)/\4/')"
MYSQL_DB="$(echo "$MYSQL_CONN" | sed -E 's/(.+):(.+)@(.+):([0-9]+)\/(.+)/\5/')"
MYSQL_CMD=(mysql -u"${MYSQL_USER}" -p"${MYSQL_PASS}" -h"${MYSQL_HOST}" -P"${MYSQL_PORT}" --protocol=tcp -N "${MYSQL_DB}")

# remaining <table> <alias> <account-col>: rows still holding an account_user.id.
remaining() {
    local table="$1" alias="$2"
    "${MYSQL_CMD[@]}" -e \
        "SELECT COUNT(*) FROM ${table} ${alias} JOIN account_user au ON ${alias}.actor_id = au.id WHERE ${alias}.identity_type = 'user';"
}

RL_PENDING="$(remaining request_log rl)"
AE_PENDING="$(remaining audit_event ae)"
info "Rows holding an account_user.id to convert → request_log: ${RL_PENDING}, audit_event: ${AE_PENDING} (db: ${MYSQL_DB}@${MYSQL_HOST})"

if [ "$DRY_RUN" -eq 1 ]; then
    info "Dry run — no rows written."
    exit 0
fi
if [ $((RL_PENDING + AE_PENDING)) -eq 0 ]; then
    info "Nothing to migrate. Exiting."
    exit 0
fi

if [ "$IS_LOCAL" -ne 1 ] && [ "$FORCE" -ne 1 ]; then
    warn "About to rewrite actor_id on ${MYSQL_DB}@${MYSQL_HOST} (remote)."
    read -r -p "Type the database name (${MYSQL_DB}) to proceed: " CONFIRM
    if [ "$CONFIRM" != "$MYSQL_DB" ]; then
        error "Confirmation did not match. Aborted."
        exit 1
    fi
fi

# backfill <label> <update-sql-without-trailing-semicolon>
# Loops the batched UPDATE until it affects zero rows, summing the total.
backfill() {
    local label="$1" update_sql="$2"
    local total=0 affected=1
    while [ "${affected}" -gt 0 ]; do
        affected="$("${MYSQL_CMD[@]}" -e "${update_sql} LIMIT ${BATCH_SIZE}; SELECT ROW_COUNT();" | tail -1)"
        affected="${affected:-0}"
        total=$((total + affected))
        if [ "${affected}" -gt 0 ]; then
            info "${label}: converted ${affected} (running total ${total})"
        fi
    done
    info "${label}: done — ${total} row(s) rewritten from account_user.id to user_id."
}

info "Normalizing actor_id → user_id (batch size ${BATCH_SIZE})..."

backfill "request_log" \
    "UPDATE request_log rl JOIN account_user au ON rl.actor_id = au.id SET rl.actor_id = au.user_id WHERE rl.identity_type = 'user'"

backfill "audit_event" \
    "UPDATE audit_event ae JOIN account_user au ON ae.actor_id = au.id SET ae.actor_id = au.user_id WHERE ae.identity_type = 'user'"

info "Backfill complete."
