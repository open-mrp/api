#!/usr/bin/env bash

# check-migration-versions.sh
# Fails when a goose migration directory holds two files with the same version, or when a migration
# added since BASE_REF is numbered at or below the highest version already on BASE_REF.
#
# goose refuses to load a directory with a duplicate version, so one reaching main breaks every
# migrate run — including the release step that opens the PlanetScale deploy request. 00018 was
# reused that way and v2.6.4 shipped with its schema never reaching prod. A new migration numbered
# below one already on main is the same mistake one rebase later: the release treats every version
# present at the previous tag as live, so an out-of-order file can be skipped without running.
#
# Optional env:
#   BASE_REF   git ref new migrations are compared against (default origin/main; skipped if missing)

set -uo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$(cd "$SCRIPT_DIR/.." && pwd)" || exit 1

BASE_REF="${BASE_REF:-origin/main}"

DIRS=(
    shared/db/migrations
    shared/db/data-migrations
    services/agent-service/db/migrations
    services/agent-service/db/data-migrations
    services/agent-service/db/seeds
)

failed=0

# Prints "<version> <file>" for each goose migration file, version as a plain integer.
versions_in() {
    local file version
    while IFS= read -r file; do
        [ -n "$file" ] || continue
        version="$(basename "$file")"
        version="${version%%_*}"
        case "$version" in ''|*[!0-9]*) continue ;; esac
        echo "$((10#$version)) $file"
    done
}

has_base=0
if git rev-parse --verify --quiet "$BASE_REF^{commit}" >/dev/null; then
    has_base=1
fi

for dir in "${DIRS[@]}"; do
    [ -d "$dir" ] || continue

    current="$(find "$dir" -maxdepth 1 -type f \( -name '*.sql' -o -name '*.go' \) ! -name '*_test.go' | sort | versions_in)"

    dupes="$(echo "$current" | awk 'NF { print $1 }' | sort -n | uniq -d)"
    for v in $dupes; do
        echo -e "${RED}[ERROR]${NC} $dir has more than one migration with version $v:"
        echo "$current" | awk -v v="$v" '$1 == v { print "    " $2 }'
        failed=1
    done

    [ "$has_base" -eq 1 ] || continue

    base_max="$(git ls-tree -r --name-only "$BASE_REF" -- "$dir" | versions_in | awk 'NF { print $1 }' | sort -n | tail -1)"
    [ -n "$base_max" ] || continue

    added="$(git diff --name-only --diff-filter=AR "$BASE_REF" -- "$dir" | versions_in)"
    while read -r v file; do
        [ -n "${v:-}" ] || continue
        if [ "$v" -le "$base_max" ]; then
            echo -e "${RED}[ERROR]${NC} $file is new but its version ($v) is not above the highest on $BASE_REF ($base_max). Renumber it."
            failed=1
        fi
    done <<EOF
$added
EOF
done

if [ "$failed" -ne 0 ]; then
    exit 1
fi

if [ "$has_base" -eq 0 ]; then
    echo -e "${GREEN}[OK]${NC} No duplicate migration versions ($BASE_REF not found; ordering check skipped)."
else
    echo -e "${GREEN}[OK]${NC} Migration versions are unique and new ones sort after $BASE_REF."
fi
