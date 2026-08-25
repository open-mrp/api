#!/usr/bin/env bash

# planetscale-release-branch.sh
# Prepares the schema half of a release: cuts a PlanetScale dev branch from prod, applies the pending
# goose migrations to it, and opens a deploy request for review.
#
# Runs on the release PR, before merge. The deploy request it leaves behind is deployed later by
# planetscale-deploy-release.sh, which the release pipeline runs after the PR merges.
#
# Nothing here touches prod's schema. Prod has safe migrations enabled, so it only ever accepts DDL
# through the deploy request this script creates.
#
# Required env:
#   RELEASE_VERSION              version being released, e.g. 1.2.0 or v1.2.0
#   PLANETSCALE_SERVICE_TOKEN_ID
#   PLANETSCALE_SERVICE_TOKEN
# Optional env:
#   PS_ORG            default augno-inc
#   PS_DATABASE       default augno_core
#   PS_PROD_BRANCH    default prod
#   BASE_REF          git ref the release is cut from, for change detection (default HEAD)
#
# Outputs (written to $GITHUB_OUTPUT when set):
#   has_migrations    true|false
#   branch            the PlanetScale branch name
#   deploy_request    deploy request number, when one was created
#   deploy_request_url

set -euo pipefail

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1" >&2; }

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

PS_ORG="${PS_ORG:-augno-inc}"
PS_DATABASE="${PS_DATABASE:-augno_core}"
PS_PROD_BRANCH="${PS_PROD_BRANCH:-prod}"
BASE_REF="${BASE_REF:-HEAD}"
MIGRATIONS_DIR="shared/db/migrations"

for var in RELEASE_VERSION PLANETSCALE_SERVICE_TOKEN_ID PLANETSCALE_SERVICE_TOKEN; do
    if [ -z "${!var:-}" ]; then
        error "$var is required."
        exit 1
    fi
done

emit() {
    if [ -n "${GITHUB_OUTPUT:-}" ]; then
        echo "$1=$2" >> "$GITHUB_OUTPUT"
    fi
}

# PlanetScale branch names allow lowercase alphanumerics and dashes; a version like 1.2.0 has to be
# flattened. Both this script and the deploy step derive the name the same way from the same version,
# which is what lets the deploy step find the deploy request without any state passed between them.
source "$SCRIPT_DIR/planetscale-branch-name.sh"
BRANCH="$(planetscale_release_branch "$RELEASE_VERSION")"
emit branch "$BRANCH"

info "Release $RELEASE_VERSION -> PlanetScale branch $BRANCH"

# --- Is there anything to deploy? ---

# The release PR itself only carries release-please's version and changelog commits; the migrations
# were merged to main in earlier PRs. So the comparison that matters is the previous release tag
# against the commit being released, not the PR's own diff.
git fetch --force --tags --quiet origin 2>/dev/null || true

PREVIOUS_TAG="$(git tag --list 'v*' --sort=-v:refname --merged "$BASE_REF" | head -1 || true)"

if [ -z "$PREVIOUS_TAG" ]; then
    warn "No previous release tag found; treating every migration as new."
    CHANGED="$(git ls-tree -r --name-only "$BASE_REF" -- "$MIGRATIONS_DIR" || true)"
else
    info "Comparing $MIGRATIONS_DIR between $PREVIOUS_TAG and $BASE_REF"
    CHANGED="$(git diff --name-only "$PREVIOUS_TAG" "$BASE_REF" -- "$MIGRATIONS_DIR" || true)"
fi

if [ -z "$CHANGED" ]; then
    info "No migration changes in this release. Nothing to deploy."
    emit has_migrations false
    exit 0
fi

info "Migrations in this release:"
echo "$CHANGED" | sed 's/^/  /'
emit has_migrations true

# --- Migrations already live in prod ---

# The branch is cut from prod, so it inherits the schema of every migration shipped in prior releases.
# baseline only records 00001, so goose would otherwise replay 00002+ against a branch that already
# carries them and collide on the first non-idempotent DDL. Record the versions present at the previous
# release tag (what prod reflects) as applied, and let `up` run only what this release adds.
SHIPPED_MIGRATION_VERSIONS=""
if [ -n "$PREVIOUS_TAG" ]; then
    while IFS= read -r shipped_file; do
        [ -n "$shipped_file" ] || continue
        version="$(basename "$shipped_file")"
        version="${version%%_*}"
        case "$version" in ''|*[!0-9]*) continue ;; esac
        version=$((10#$version))
        # 00001 is the baseline; `migrate.sh baseline` records it.
        if [ "$version" -le 1 ]; then continue; fi
        SHIPPED_MIGRATION_VERSIONS="$SHIPPED_MIGRATION_VERSIONS $version"
    done <<EOF
$(git ls-tree -r --name-only "$PREVIOUS_TAG" -- "$MIGRATIONS_DIR" || true)
EOF
fi
export SHIPPED_MIGRATION_VERSIONS

if [ -n "$SHIPPED_MIGRATION_VERSIONS" ]; then
    info "Already deployed in $PREVIOUS_TAG, recorded as applied on the branch:$SHIPPED_MIGRATION_VERSIONS"
fi

# --- Recreate the branch ---

pscale_cmd() {
    pscale "$@" --org "$PS_ORG"
}

# The branch is recreated rather than reused. A release PR is rebuilt every time a commit lands on
# main, and a branch left over from an earlier run may have had a since-edited migration applied to
# it. Cutting fresh from prod means the deploy request diff always describes exactly the migrations
# in this release.
if pscale_cmd branch show "$PS_DATABASE" "$BRANCH" >/dev/null 2>&1; then
    info "Branch $BRANCH already exists; closing any open deploy request and recreating it."

    EXISTING="$(pscale_cmd deploy-request show "$PS_DATABASE" "$BRANCH" --format json 2>/dev/null \
        | jq -r 'select(.state == "open" or .state == "pending") | .number // empty' || true)"

    if [ -n "$EXISTING" ]; then
        info "Closing superseded deploy request #$EXISTING"
        pscale_cmd deploy-request close "$PS_DATABASE" "$EXISTING" >/dev/null || true
    fi

    pscale_cmd branch delete "$PS_DATABASE" "$BRANCH" --force
fi

info "Creating branch $BRANCH from $PS_PROD_BRANCH..."
pscale_cmd branch create "$PS_DATABASE" "$BRANCH" --from "$PS_PROD_BRANCH" --wait

# --- Apply migrations ---

# `pscale connect` opens a local proxy and runs the command with DATABASE_URL pointed at it, so no
# branch password is ever created, stored, or left behind for cleanup.
info "Applying migrations to $BRANCH..."

SENTINEL="$(mktemp)"
rm -f "$SENTINEL"
export MIGRATE_SENTINEL="$SENTINEL"

pscale_cmd connect "$PS_DATABASE" "$BRANCH" \
    --execute-protocol mysql \
    --execute "$SCRIPT_DIR/planetscale-apply-migrations.sh"

# pscale owns the exit status of --execute, and a swallowed non-zero there would look like a clean
# run with an empty schema diff. The sentinel is the independent proof that goose actually finished.
if [ ! -f "$SENTINEL" ]; then
    error "Migrations did not complete successfully on $BRANCH."
    exit 1
fi
rm -f "$SENTINEL"

# --- Open the deploy request ---

info "Creating deploy request into $PS_PROD_BRANCH..."

DR_JSON="$(pscale_cmd deploy-request create "$PS_DATABASE" "$BRANCH" \
    --into "$PS_PROD_BRANCH" \
    --enable-auto-apply \
    --auto-delete-branch \
    --notes "Automated: schema for release $RELEASE_VERSION. Deployed when the release PR merges." \
    --format json)"

DR_NUMBER="$(echo "$DR_JSON" | jq -r '.number')"

if [ -z "$DR_NUMBER" ] || [ "$DR_NUMBER" = "null" ]; then
    error "Could not read the deploy request number from pscale:"
    echo "$DR_JSON" >&2
    exit 1
fi

DR_URL="https://app.planetscale.com/$PS_ORG/$PS_DATABASE/deploy-requests/$DR_NUMBER"

emit deploy_request "$DR_NUMBER"
emit deploy_request_url "$DR_URL"

info "Deploy request #$DR_NUMBER is ready for review: $DR_URL"
