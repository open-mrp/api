#!/usr/bin/env bash

# Database Migrations (goose)
#
# Single entrypoint for the core-service MySQL schema. Schema changes are authored here as goose
# migrations; the Prisma schema in the dashboard repo is a downstream model definition updated by hand
# alongside them.
#
# Usage: ./scripts/migrate.sh <command> [--target local|branch|prod] [--yes]
#
# See docs/patterns/database-migrations.md for the full workflow.

set -euo pipefail

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; }

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

# .env fills in what the caller did not set; an explicitly exported value wins, so a one-off run against
# another database does not silently get redirected at the local one.
if [ -f ".env" ]; then
    while IFS= read -r line; do
        if [[ -n "$line" && ! "$line" =~ ^[[:space:]]*# ]]; then
            key="${line%%=*}"
            if [ -z "${!key:-}" ]; then
                export "$line"
            fi
        fi
    done < .env
fi

DDL_DIR="shared/db/migrations"
DML_DIR="shared/db/data-migrations"
DML_TABLE="goose_db_version_data"

# The baseline is a mysqldump of the schema as it stood at the goose cutover. It opens with DROP TABLE
# IF EXISTS for every table, so applying it to a populated database destroys that database. The
# empty-schema check in require_safe_to_apply exists to keep that from happening.
BASELINE_VERSION=1

usage() {
    cat <<'USAGE'
Usage: ./scripts/migrate.sh <command> [options]

Schema (DDL) commands - shared/db/migrations:
  create <name>       Scaffold a new schema migration
  up                  Apply pending schema migrations
  down                Roll back the most recent schema migration
  status              Show which schema migrations are applied
  version             Show the current schema version
  baseline            Record the baseline as applied without running it

Data (DML) commands - shared/db/data-migrations:
  create-data <name>  Scaffold a new data migration
  data-up             Apply pending data migrations
  data-status         Show which data migrations are applied

Options:
  --target local      Local Docker MySQL via DB_URL (default)
  --target branch     PlanetScale dev branch via PS_BRANCH_DB_URL
  --target prod       PlanetScale prod via PS_PROD_DB_URL (data migrations only)
  --yes               Skip the interactive confirmation for prod data migrations

Schema DDL never runs against prod: prod has safe migrations enabled and takes schema changes
only through a PlanetScale deploy request. See docs/patterns/database-migrations.md.
USAGE
}

# --- Parse arguments ---

COMMAND="${1:-}"
if [ $# -gt 0 ]; then
    shift
fi

NAME=""
TARGET="local"
ASSUME_YES=0

case "$COMMAND" in
    create|create-data)
        NAME="${1:-}"
        if [ -z "$NAME" ]; then
            error "Missing migration name. Usage: ./scripts/migrate.sh $COMMAND <name>"
            exit 1
        fi
        shift
        ;;
esac

while [[ $# -gt 0 ]]; do
    case "$1" in
        --target)
            TARGET="${2:-}"
            shift 2
            ;;
        --yes)
            ASSUME_YES=1
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            error "Unknown argument: $1"
            usage
            exit 1
            ;;
    esac
done

if [ -z "$COMMAND" ] || [ "$COMMAND" = "help" ]; then
    usage
    exit 0
fi

case "$TARGET" in
    local|branch|prod) ;;
    *)
        error "Invalid --target '$TARGET'. Must be local, branch, or prod."
        exit 1
        ;;
esac

# --- Connection handling ---

# Parses mysql://user:pass@host:port/db[?params] into MYSQL_* and a Go DSN for goose. goose speaks the
# go-sql-driver form (user:pass@tcp(host:port)/db) while everything else in this repo stores the URL
# form in DB_URL, so both are derived here rather than kept in two env vars that can disagree.
parse_url() {
    local url="$1"
    local conn base hostpart hostport

    if ! echo "$url" | grep -qiE '^mysql://'; then
        error "Connection string must start with mysql:// (got: ${url%%:*}://...)"
        exit 1
    fi

    conn="${url#mysql://}"
    base="${conn%%\?*}"
    QUERY=""
    if [ "$conn" != "$base" ]; then
        QUERY="${conn#*\?}"
    fi

    MYSQL_USER="$(echo "$base" | sed -E 's/^([^:]+):.*/\1/')"
    MYSQL_PASS="$(echo "$base" | sed -E 's/^[^:]+:(.+)@[^@]*$/\1/')"
    hostpart="$(echo "$base" | sed -E 's/^.*@//')"
    MYSQL_DB="$(echo "$hostpart" | sed -E 's|^[^/]+/||')"
    hostport="$(echo "$hostpart" | sed -E 's|/.*||')"

    if echo "$hostport" | grep -qE ':[0-9]+$'; then
        MYSQL_HOST="$(echo "$hostport" | sed -E 's/:[0-9]+$//')"
        MYSQL_PORT="$(echo "$hostport" | sed -E 's/.*://')"
    else
        MYSQL_HOST="$hostport"
        MYSQL_PORT="3306"
    fi

    if [ -z "$MYSQL_HOST" ] || [ -z "$MYSQL_DB" ] || [ -z "$MYSQL_USER" ]; then
        error "Could not parse connection string. Expected mysql://user:pass@host:port/database."
        exit 1
    fi

    GOOSE_DSN="${MYSQL_USER}:${MYSQL_PASS}@tcp(${MYSQL_HOST}:${MYSQL_PORT})/${MYSQL_DB}?parseTime=true"

    # PlanetScale terminates TLS on the edge and rejects plaintext connections; go-sql-driver defaults
    # to no TLS, so it has to be asked for explicitly.
    if is_planetscale && ! echo "$QUERY" | grep -q 'tls='; then
        GOOSE_DSN="${GOOSE_DSN}&tls=true"
    fi
}

is_planetscale() {
    echo "$MYSQL_HOST" | grep -qiE 'psdb\.cloud|planetscale|pscale'
}

resolve_target() {
    case "$TARGET" in
        local)
            local url="${DB_URL:-}"
            if [ -z "$url" ]; then
                error "DB_URL is not set. Export it or add it to .env, or run 'make local-db'."
                exit 1
            fi
            parse_url "$url"
            if is_planetscale; then
                error "DB_URL points at PlanetScale. The local target must be the Docker MySQL."
                exit 1
            fi
            if ! echo "$MYSQL_HOST" | grep -qiE '^(localhost|127\.0\.0\.1|::1|mysql|seed-db)$'; then
                error "Refusing to run: the local target must connect to localhost or a Docker container (got $MYSQL_HOST)."
                exit 1
            fi
            TARGET_LABEL="local MySQL ($MYSQL_HOST:$MYSQL_PORT/$MYSQL_DB)"
            ;;
        branch)
            local url="${PS_BRANCH_DB_URL:-}"
            if [ -z "$url" ]; then
                error "PS_BRANCH_DB_URL is not set. Create a PlanetScale dev branch and export its connection string."
                exit 1
            fi
            parse_url "$url"
            TARGET_LABEL="PlanetScale dev branch ($MYSQL_HOST/$MYSQL_DB)"
            ;;
        prod)
            local url="${PS_PROD_DB_URL:-}"
            if [ -z "$url" ]; then
                error "PS_PROD_DB_URL is not set."
                exit 1
            fi
            parse_url "$url"
            TARGET_LABEL="PlanetScale PROD ($MYSQL_HOST/$MYSQL_DB)"
            ;;
    esac
}

mysql_query() {
    MYSQL_PWD="$MYSQL_PASS" mysql \
        -u"$MYSQL_USER" -h"$MYSQL_HOST" -P"$MYSQL_PORT" --protocol=tcp \
        -D "$MYSQL_DB" -s --skip-column-names -e "$1"
}

run_goose() {
    local dir="$1"
    shift
    local table_args=()
    if [ "$dir" = "$DML_DIR" ]; then
        table_args=(-table "$DML_TABLE")
    fi

    GOOSE_DRIVER=mysql GOOSE_DBSTRING="$GOOSE_DSN" \
        goose ${table_args[@]+"${table_args[@]}"} -dir "$dir" "$@"
}

require_goose() {
    if ! command -v goose >/dev/null 2>&1; then
        error "goose is not installed. Run 'make install-tools' first."
        exit 1
    fi
}

require_mysql_client() {
    if ! command -v mysql >/dev/null 2>&1; then
        error "mysql client is not installed. Run: brew install mysql-client@8.4"
        exit 1
    fi
}

# Guards `up` against the one destructive case: a database that already carries the schema but has no
# goose bookkeeping. goose would treat the baseline as pending and run its DROP TABLE preamble. A
# PlanetScale branch freshly cut from prod is exactly this shape, which is what `baseline` is for.
require_safe_to_apply() {
    local tables goose_table

    tables="$(mysql_query "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = '$MYSQL_DB' AND TABLE_TYPE = 'BASE TABLE' AND TABLE_NAME NOT IN ('goose_db_version', '$DML_TABLE');")"
    goose_table="$(mysql_query "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = '$MYSQL_DB' AND TABLE_NAME = 'goose_db_version';")"

    if [ "$tables" -gt 0 ] && [ "$goose_table" -eq 0 ]; then
        error "$TARGET_LABEL already has $tables tables but no goose_db_version table."
        error ""
        error "Applying migrations now would run the baseline (00001_initial.sql), which begins by"
        error "dropping every table. Record the baseline as already applied instead:"
        error ""
        error "    ./scripts/migrate.sh baseline --target $TARGET"
        error ""
        exit 1
    fi
}

confirm_prod() {
    if [ "$ASSUME_YES" -eq 1 ]; then
        return
    fi
    warn "This applies data migrations to PRODUCTION ($MYSQL_HOST/$MYSQL_DB)."
    read -r -p "Type 'yes' to continue: " reply
    if [ "$reply" != "yes" ]; then
        error "Aborted."
        exit 1
    fi
}

# --- Commands ---

# The file is written here rather than by `goose create` so the template can carry the annotations this
# database needs. Schema migrations get NO TRANSACTION because Vitess rejects DDL inside an explicit
# transaction; MySQL autocommits DDL anyway, so the wrapper only ever costs. Data migrations keep it.
scaffold() {
    local dir="$1" name="$2" no_transaction="$3"
    local slug next path

    slug="$(echo "$name" | tr '[:upper:] ' '[:lower:]_' | sed -E 's/[^a-z0-9_]//g')"
    if [ -z "$slug" ]; then
        error "Migration name must contain at least one letter or digit."
        exit 1
    fi

    mkdir -p "$dir"
    next="$(next_sequence "$dir")"
    path="$dir/${next}_${slug}.sql"

    if [ -e "$path" ]; then
        error "$path already exists."
        exit 1
    fi

    {
        if [ "$no_transaction" = "yes" ]; then
            echo "-- +goose NO TRANSACTION"
        fi
        echo "-- +goose Up"
        echo ""
        echo ""
        echo "-- +goose Down"
        echo ""
    } > "$path"

    info "Created $path"
}

# Sequential numbering, zero-padded to five digits, matching the baseline and the agent-service
# migrations. Width matters beyond tidiness: setup-e2e-db.sh applies migrations via a shell glob, which
# sorts lexically, so a narrower number would sort ahead of the baseline and run first.
next_sequence() {
    local dir="$1" highest=0 base version

    for f in "$dir"/*.sql; do
        [ -f "$f" ] || continue
        base="$(basename "$f")"
        version="${base%%_*}"
        case "$version" in
            ''|*[!0-9]*) continue ;;
        esac
        version=$((10#$version))
        if [ "$version" -gt "$highest" ]; then
            highest="$version"
        fi
    done

    printf '%05d' $((highest + 1))
}

case "$COMMAND" in
    create)
        scaffold "$DDL_DIR" "$NAME" yes
        ;;

    create-data)
        scaffold "$DML_DIR" "$NAME" no
        ;;

    up)
        require_goose
        require_mysql_client
        resolve_target
        if [ "$TARGET" = "prod" ]; then
            error "Schema migrations never run against prod."
            error "Prod has safe migrations enabled; apply to a dev branch and open a PlanetScale deploy request:"
            error "    ./scripts/migrate.sh up --target branch"
            exit 1
        fi
        require_safe_to_apply
        info "Applying schema migrations to $TARGET_LABEL"
        run_goose "$DDL_DIR" up
        ;;

    down)
        require_goose
        resolve_target
        if [ "$TARGET" = "prod" ]; then
            error "Schema migrations never run against prod. Revert the PlanetScale deploy request instead."
            exit 1
        fi
        info "Rolling back the most recent schema migration on $TARGET_LABEL"
        run_goose "$DDL_DIR" down
        ;;

    status)
        require_goose
        resolve_target
        info "Schema migration status for $TARGET_LABEL"
        run_goose "$DDL_DIR" status
        ;;

    version)
        require_goose
        resolve_target
        run_goose "$DDL_DIR" version
        ;;

    baseline)
        require_goose
        require_mysql_client
        resolve_target

        applied="$(mysql_query "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = '$MYSQL_DB' AND TABLE_NAME = 'goose_db_version';")"
        if [ "$applied" -gt 0 ]; then
            recorded="$(mysql_query "SELECT COUNT(*) FROM goose_db_version WHERE version_id = $BASELINE_VERSION AND is_applied = 1;")"
            if [ "$recorded" -gt 0 ]; then
                info "Baseline is already recorded on $TARGET_LABEL. Nothing to do."
                exit 0
            fi
        else
            info "Creating goose_db_version on $TARGET_LABEL..."
            mysql_query "CREATE TABLE goose_db_version (id bigint NOT NULL AUTO_INCREMENT, version_id bigint NOT NULL, is_applied boolean NOT NULL, tstamp timestamp NULL DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY (id));" >/dev/null
            mysql_query "INSERT INTO goose_db_version (version_id, is_applied) VALUES (0, 1);" >/dev/null
        fi

        info "Recording baseline (version $BASELINE_VERSION) as applied without running it..."
        mysql_query "INSERT INTO goose_db_version (version_id, is_applied) VALUES ($BASELINE_VERSION, 1);" >/dev/null
        info "Done. 'up' will now start from the first migration after the baseline."
        ;;

    data-up)
        require_goose
        resolve_target
        if [ "$TARGET" = "prod" ]; then
            confirm_prod
        fi
        info "Applying data migrations to $TARGET_LABEL"
        run_goose "$DML_DIR" up
        ;;

    data-status)
        require_goose
        resolve_target
        info "Data migration status for $TARGET_LABEL"
        run_goose "$DML_DIR" status
        ;;

    *)
        error "Unknown command: $COMMAND"
        usage
        exit 1
        ;;
esac
