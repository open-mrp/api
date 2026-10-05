#!/usr/bin/env bash

# planetscale-apply-schema-part.sh
# Runs inside `pscale connect --execute`: applies one part of a split release (written by
# `schemasplit plan`) to a branch freshly cut from prod. Not meant to be called directly —
# planetscale-release-branch.sh invokes it once per part.
#
# Required env:
#   DATABASE_URL       set by pscale connect
#   SCHEMA_PART_FILE   the part-N.sql file to apply
#   MIGRATE_SENTINEL   file to touch on success, proving to the caller that this actually finished

set -euo pipefail

if [ -z "${DATABASE_URL:-}" ]; then
    echo "DATABASE_URL is not set. This script runs under 'pscale connect --execute'." >&2
    exit 1
fi

schemasplit apply -file "$SCHEMA_PART_FILE"

if [ -n "${MIGRATE_SENTINEL:-}" ]; then
    touch "$MIGRATE_SENTINEL"
fi
