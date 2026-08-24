#!/usr/bin/env bash

# planetscale-deploy-release.sh
# Deploys the schema for a release: finds the deploy request the release PR left open for this
# version and deploys it to prod, blocking until PlanetScale finishes the migration.
#
# The release pipeline runs this before rolling any service image, and gates the rollout on it. A
# failed schema deploy must stop the release — new code against an old schema is the failure this
# ordering exists to prevent.
#
# Exits 0 with nothing to do when the release carries no migrations, so the pipeline is not gated on
# a deploy request that was never created.
#
# Required env:
#   RELEASE_VERSION              version being released, e.g. 1.2.0 or v1.2.0
#   PLANETSCALE_SERVICE_TOKEN_ID
#   PLANETSCALE_SERVICE_TOKEN
# Optional env:
#   PS_ORG            default augno-inc
#   PS_DATABASE       default augno_core
#   PS_PROD_BRANCH    default prod

set -euo pipefail

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1" >&2; }

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$(cd "$SCRIPT_DIR/.." && pwd)"

PS_ORG="${PS_ORG:-augno-inc}"
PS_DATABASE="${PS_DATABASE:-augno_core}"
PS_PROD_BRANCH="${PS_PROD_BRANCH:-prod}"

for var in RELEASE_VERSION PLANETSCALE_SERVICE_TOKEN_ID PLANETSCALE_SERVICE_TOKEN; do
    if [ -z "${!var:-}" ]; then
        error "$var is required."
        exit 1
    fi
done

source "$SCRIPT_DIR/planetscale-branch-name.sh"
BRANCH="$(planetscale_release_branch "$RELEASE_VERSION")"

pscale_cmd() {
    pscale "$@" --org "$PS_ORG"
}

info "Looking for the deploy request on branch $BRANCH..."

# `show` takes a branch name as readily as a number, so the branch name derived from the version is
# the only thing this needs from the run that prepared it. A release with no migrations never had a
# branch cut, and a re-run of an already-deployed release has had its branch auto-deleted — both
# surface here as "not found", and both mean there is nothing to apply.
if ! DR_JSON="$(pscale_cmd deploy-request show "$PS_DATABASE" "$BRANCH" --format json 2>/dev/null)"; then
    info "No deploy request for $BRANCH — this release carries no schema changes."
    exit 0
fi

DR_NUMBER="$(echo "$DR_JSON" | jq -r '.number // empty')"
DR_STATE="$(echo "$DR_JSON" | jq -r '.state // empty')"
DR_URL="https://app.planetscale.com/$PS_ORG/$PS_DATABASE/deploy-requests/${DR_NUMBER:-}"

info "Deploy request #${DR_NUMBER:-?} (state: ${DR_STATE:-unknown}) — $DR_URL"

# Re-running a release must not fail on schema that is already live: a closed deploy request has
# either been deployed or deliberately abandoned, and either way there is nothing here to apply.
if [ "$DR_STATE" = "closed" ]; then
    info "Deploy request is already closed. Nothing to deploy."
    exit 0
fi

info "Deploying into $PS_PROD_BRANCH and waiting for it to finish..."

if ! pscale_cmd deploy-request deploy "$PS_DATABASE" "$BRANCH" --wait; then
    error "Schema deploy for release $RELEASE_VERSION failed. See $DR_URL"
    error "The service rollout is gated on this and will not run."
    exit 1
fi

# `--wait` returning zero is the real signal. This second look only catches a deploy that reported
# success while the request itself ended somewhere bad; an unrecognised state is not treated as a
# failure, because failing the release on a field this script could not parse would be worse than
# trusting the exit code that already succeeded.
FINAL_STATE="$(pscale_cmd deploy-request show "$PS_DATABASE" "$BRANCH" --format json 2>/dev/null \
    | jq -r '.deployment_state // .state // empty' || true)"

case "$FINAL_STATE" in
    # complete_error / complete_revert_error are PlanetScale's own terminal failure states; they also
    # leave the deploy queue blocked, which the next release would hit as an unrelated-looking failure.
    complete_error|complete_revert_error|error|failed|cancelled|canceled)
        error "Deploy request ended in state '$FINAL_STATE'. See $DR_URL"
        error "The deploy queue may be blocked: pscale deploy-request unblock $PS_DATABASE ${DR_NUMBER:-<number>} --org $PS_ORG"
        exit 1
        ;;
    "")
        warn "Could not read the final deploy request state; trusting the successful deploy."
        ;;
    *)
        info "Deploy request finished in state '$FINAL_STATE'."
        ;;
esac

info "Schema for release $RELEASE_VERSION is live."
