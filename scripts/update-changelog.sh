#!/usr/bin/env bash
#
# update-changelog.sh - Prepend a changelog entry to CHANGELOG.md
#
# Usage:
#   ./scripts/generate-changelog.sh | ./scripts/update-changelog.sh
#   ./scripts/update-changelog.sh < changelog-entry.md
#
# Reads changelog content from stdin and inserts it into CHANGELOG.md
# after the first --- separator (between the header and release entries).
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CHANGELOG="$REPO_ROOT/CHANGELOG.md"

# Read new entry from stdin
NEW_ENTRY=$(cat)

if [[ -z "$NEW_ENTRY" ]]; then
    echo "Error: No changelog entry provided on stdin" >&2
    exit 1
fi

# Find the first --- separator line
INSERT_LINE=$(grep -n '^---$' "$CHANGELOG" | head -1 | cut -d: -f1)

if [[ -z "$INSERT_LINE" ]]; then
    echo "Error: Could not find --- separator in CHANGELOG.md" >&2
    exit 1
fi

# Build new file: header (up to ---) + blank line + new entry + rest of file
{
    head -n "$INSERT_LINE" "$CHANGELOG"
    echo ""
    echo "$NEW_ENTRY"
    tail -n "+$((INSERT_LINE + 1))" "$CHANGELOG"
} > "$CHANGELOG.tmp"

mv "$CHANGELOG.tmp" "$CHANGELOG"
