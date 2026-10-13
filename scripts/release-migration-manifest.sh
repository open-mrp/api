#!/usr/bin/env bash

# release-migration-manifest.sh
# Lists everything a release will apply to the databases: schema and data migrations, across both the
# core MySQL database and the agent-service Postgres one.
#
# The MySQL schema half gets a reviewable PlanetScale deploy request. The other three do not — Postgres
# applies DDL directly and backfills are DML — so this manifest is the review surface for them, printed
# onto the release PR before anyone merges it.
#
# Usage: ./scripts/release-migration-manifest.sh [--format markdown|plain]
#
# Optional env:
#   BASE_REF   commit being released (default HEAD)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$(cd "$SCRIPT_DIR/.." && pwd)"

BASE_REF="${BASE_REF:-HEAD}"
FORMAT="markdown"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --format) FORMAT="${2:-markdown}"; shift 2 ;;
        *) echo "Unknown argument: $1" >&2; exit 1 ;;
    esac
done

git fetch --force --tags --quiet origin 2>/dev/null || true
source "$SCRIPT_DIR/release-migration-changes.sh"
PREVIOUS_TAG="$(release_previous_tag "$BASE_REF")"

CORE_SCHEMA="$(release_new_migrations "$PREVIOUS_TAG" "$BASE_REF" shared/db/migrations)"
CORE_DATA="$(release_new_migrations "$PREVIOUS_TAG" "$BASE_REF" shared/db/data-migrations)"
AGENT_SCHEMA="$(release_new_migrations "$PREVIOUS_TAG" "$BASE_REF" services/agent-service/db/migrations)"
AGENT_DATA="$(release_new_migrations "$PREVIOUS_TAG" "$BASE_REF" services/agent-service/db/data-migrations)"

if [ -z "$CORE_SCHEMA$CORE_DATA$AGENT_SCHEMA$AGENT_DATA" ]; then
    if [ "$FORMAT" = "markdown" ]; then
        echo "No database migrations in this release."
    else
        echo "none"
    fi
    exit 0
fi

section() {
    local title="$1" how="$2" files="$3"
    [ -z "$files" ] && return 0

    if [ "$FORMAT" = "markdown" ]; then
        echo "**$title** — $how"
        echo
        echo "$files" | sed 's|.*/||; s/^/- `/; s/$/`/'
        echo
    else
        echo "$title ($how):"
        echo "$files" | sed 's|.*/||; s/^/  /'
    fi
}

section "Core schema (MySQL)"  "deploy request, reviewable below"        "$CORE_SCHEMA"
section "Agent schema (Postgres)" "applied directly on merge"            "$AGENT_SCHEMA"
section "Core backfills (MySQL)"  "applied on merge, after the schema"   "$CORE_DATA"
section "Agent backfills (Postgres)" "applied on merge, after the schema" "$AGENT_DATA"
