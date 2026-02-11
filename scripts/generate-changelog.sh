#!/usr/bin/env bash
#
# generate-changelog.sh - Generate changelog from conventional commits
#
# Usage:
#   ./scripts/generate-changelog.sh [from-tag] [version]
#
# Arguments:
#   from-tag    Starting tag (optional, defaults to previous tag)
#   version     Version string (optional, defaults to go run ./cmd/print-version)
#
# Output:
#   Markdown formatted changelog to stdout
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# shellcheck source=./version.sh
source "$SCRIPT_DIR/version.sh"

cd "$REPO_ROOT"

# Get the starting point
if [[ $# -gt 0 ]]; then
    FROM_TAG="$1"
else
    # Find previous tag
    FROM_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
fi

# Get current version (second argument overrides)
if [[ $# -gt 1 && -n "$2" ]]; then
    CURRENT_VERSION="$2"
else
    CURRENT_VERSION=$(get_current_version)
fi
CURRENT_TAG=$(version_to_tag "$CURRENT_VERSION")

# Determine commit range
if [[ -n "$FROM_TAG" ]]; then
    RANGE="${FROM_TAG}..HEAD"
    COMPARE_URL="https://github.com/Augno/api/compare/${FROM_TAG}...${CURRENT_TAG}"
else
    RANGE="HEAD"
    COMPARE_URL=""
fi

# Initialize arrays for different commit types
declare -a FEATURES=()
declare -a FIXES=()
declare -a BREAKING=()
declare -a OTHER=()

# Parse commits
while IFS= read -r line; do
    [[ -z "$line" ]] && continue

    # Extract commit hash and message
    hash="${line%% *}"
    message="${line#* }"
    short_hash="${hash:0:7}"

    # Check for breaking changes (! or BREAKING CHANGE in body)
    if [[ "$message" =~ ^[a-z]+!: ]] || [[ "$message" =~ BREAKING\ CHANGE ]]; then
        # Remove the type prefix for display
        display_msg="${message#*: }"
        BREAKING+=("* $display_msg ([${short_hash}](https://github.com/Augno/api/commit/${hash}))")
        continue
    fi

    # Categorize by conventional commit type
    if [[ "$message" =~ ^feat(\(.+\))?:\ (.+)$ ]]; then
        display_msg="${BASH_REMATCH[2]}"
        FEATURES+=("* $display_msg ([${short_hash}](https://github.com/Augno/api/commit/${hash}))")
    elif [[ "$message" =~ ^fix(\(.+\))?:\ (.+)$ ]]; then
        display_msg="${BASH_REMATCH[2]}"
        FIXES+=("* $display_msg ([${short_hash}](https://github.com/Augno/api/commit/${hash}))")
    elif [[ "$message" =~ ^(docs|style|refactor|perf|test|build|ci|chore)(\(.+\))?:\ (.+)$ ]]; then
        # Skip chore, ci, and build commits from changelog
        type="${BASH_REMATCH[1]}"
        if [[ "$type" != "chore" && "$type" != "ci" && "$type" != "build" ]]; then
            display_msg="${BASH_REMATCH[3]}"
            OTHER+=("* **${type}:** $display_msg ([${short_hash}](https://github.com/Augno/api/commit/${hash}))")
        fi
    fi
done < <(git log --format="%H %s" "$RANGE" 2>/dev/null)

# Generate markdown output
echo "## [$CURRENT_VERSION]($COMPARE_URL) ($(date +%Y-%m-%d))"
echo ""

if [[ ${#BREAKING[@]} -gt 0 ]]; then
    echo "### ⚠ BREAKING CHANGES"
    echo ""
    printf '%s\n' "${BREAKING[@]}"
    echo ""
fi

if [[ ${#FEATURES[@]} -gt 0 ]]; then
    echo "### Features"
    echo ""
    printf '%s\n' "${FEATURES[@]}"
    echo ""
fi

if [[ ${#FIXES[@]} -gt 0 ]]; then
    echo "### Bug Fixes"
    echo ""
    printf '%s\n' "${FIXES[@]}"
    echo ""
fi

if [[ ${#OTHER[@]} -gt 0 ]]; then
    echo "### Other Changes"
    echo ""
    printf '%s\n' "${OTHER[@]}"
    echo ""
fi

# If nothing to show
if [[ ${#BREAKING[@]} -eq 0 && ${#FEATURES[@]} -eq 0 && ${#FIXES[@]} -eq 0 && ${#OTHER[@]} -eq 0 ]]; then
    echo "No notable changes."
    echo ""
fi
