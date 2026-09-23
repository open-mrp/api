#!/usr/bin/env bash

# rollback.sh
# Rolls production back to a previously-released image tag, without rebuilding anything.
#
# A release only builds the services it changed, so most release tags have no image for most
# services: augno/<service>:vX.Y.Z exists only where that service was part of release X.Y.Z. The
# target tag is therefore not directly deployable. This script detects which services changed
# between the target tag and the live release, resolves each one to the newest tag at or before the
# target that ECR actually holds an image for, and hands each (tag, services) group to
# dispatch-infra-deploy.sh, which asks open-mrp/infra to re-roll them. Falling back to an older tag
# does not roll a service back further than asked: no image means the service did not change, so the
# older image is the same code.
#
# Image rollback only. It does NOT revert database schema or data migrations: PlanetScale's MySQL
# revert window is 30 minutes and Postgres DDL applies directly with no revert, so a schema change is
# effectively forward-only. When the window between the tags contains migrations this warns and keeps
# going — reverting the schema, if needed, is a separate deliberate decision. Roll back to a tag whose
# schema the target images can still speak.
#
# Required env:
#   TO_TAG            release tag to roll back to, e.g. v2.6.1
# Optional env:
#   SERVICES          comma-separated services to roll back; empty auto-detects the changed set
#   CURRENT_TAG       the live release rolled back from; default is the highest release tag
#   GH_TOKEN          token with actions:write on open-mrp/infra (required to actually dispatch)

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

TO_TAG="${TO_TAG:-}"
SERVICES="${SERVICES:-}"
CURRENT_TAG="${CURRENT_TAG:-}"

if [ -z "$TO_TAG" ]; then
    error "TO_TAG is required (the release tag to roll back to, e.g. v2.6.1)."
    exit 1
fi

git fetch --tags --force origin >/dev/null 2>&1 || true

if [ -z "$CURRENT_TAG" ]; then
    CURRENT_TAG="$(git tag -l 'v*' --sort=-version:refname | head -1)"
fi
if [ -z "$CURRENT_TAG" ]; then
    error "Could not determine the current release tag; pass CURRENT_TAG."
    exit 1
fi

semver_re='^v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z][0-9A-Za-z.-]*)?$'
if [[ ! "$TO_TAG" =~ $semver_re ]]; then
    error "TO_TAG '$TO_TAG' is not a release tag such as v2.6.1."
    exit 1
fi

if ! git rev-parse -q --verify "${TO_TAG}^{commit}" >/dev/null; then
    error "TO_TAG '$TO_TAG' is not an existing tag."
    exit 1
fi

if [ "$TO_TAG" = "$CURRENT_TAG" ]; then
    error "TO_TAG '$TO_TAG' is already the current release; nothing to roll back."
    exit 1
fi

if ! git merge-base --is-ancestor "${TO_TAG}^{commit}" "${CURRENT_TAG}^{commit}"; then
    error "TO_TAG '$TO_TAG' is not an ancestor of the current release '$CURRENT_TAG' — refusing to 'roll back' forward."
    exit 1
fi

info "Rolling back from ${CURRENT_TAG} to ${TO_TAG}."

if [ -z "$SERVICES" ]; then
    info "Detecting services that changed between ${TO_TAG} and ${CURRENT_TAG}..."
    detect_output="$(go run ./cmd/detect-release-changes --current-tag "$CURRENT_TAG" --base-tag "$TO_TAG")"
    echo "$detect_output"
    SERVICES="$(printf '%s\n' "$detect_output" | sed -n 's/^services_csv=//p')"
fi

if [ -z "$SERVICES" ]; then
    info "No services changed between ${TO_TAG} and ${CURRENT_TAG}; nothing to roll back."
    exit 0
fi

info "Services to roll back: ${SERVICES}"

migration_dirs=(
    shared/db/migrations
    shared/db/data-migrations
    services/agent-service/db/migrations
    services/agent-service/db/data-migrations
)
migration_changes="$(git diff --name-only "$TO_TAG" "$CURRENT_TAG" -- "${migration_dirs[@]}" 2>/dev/null | grep '\.sql$' || true)"
if [ -n "$migration_changes" ]; then
    warn "This window contains database migrations. Image rollback does NOT revert them:"
    printf '%s\n' "$migration_changes" | sed 's/^/         /' >&2
    warn "The rolled-back images will run against the current (migrated) schema. Confirm they can before proceeding."
fi

info "Resolving each service to a tag ECR has an image for..."
resolve_output="$(go run ./cmd/resolve-rollback-images --to-tag "$TO_TAG" --services "$SERVICES")"
echo "$resolve_output"
groups="$(printf '%s\n' "$resolve_output" | sed -n 's/^group=//p')"

if [ -z "$groups" ]; then
    error "No image tags resolved for: ${SERVICES}"
    exit 1
fi

# One dispatch per distinct tag: infra's deploy takes a single image_tag for the services it is given.
while read -r group_tag group_services; do
    [ -z "$group_tag" ] && continue
    info "Dispatching ${group_services} at ${group_tag} to open-mrp/infra..."
    IMAGE_TAG="$group_tag" DEPLOY_SERVICES="$group_services" "$SCRIPT_DIR/dispatch-infra-deploy.sh"
done <<<"$groups"
