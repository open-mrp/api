#!/usr/bin/env bash

# planetscale-deploy-release.sh
# Deploys the schema for a release: finds the deploy request(s) the release PR left open for this
# version and deploys them to prod one at a time, blocking until PlanetScale finishes each migration.
#
# A release within the table limit has one deploy request, on <branch>. A larger one was split by
# planetscale-release-branch.sh into parts on <branch>-p1, -p2, ..., each deploy request's notes
# carrying a marker with the part number, the part count and the id of the prepare run that made it.
#
# Each deploy request is waited on rather than assumed: one may already be queued or running (an
# auto-apply deploy started by an earlier attempt of this job), and asking PlanetScale to deploy it
# again is refused with "This deploy request cannot be deployed" — which failed v2.18.4 while its
# schema was in fact still rolling out.
#
# The release pipeline runs this before rolling any service image, and gates the rollout on it. A
# failed schema deploy must stop the release — new code against an old schema is the failure this
# ordering exists to prevent.
#
# Only called when the release changes shared/db/migrations, so a missing deploy request is a
# failure: the rollout must not run against a schema the release PR never prepared.
#
# Required env:
#   RELEASE_VERSION              version being released, e.g. 1.2.0 or v1.2.0
#   PLANETSCALE_SERVICE_TOKEN_ID
#   PLANETSCALE_SERVICE_TOKEN
# Optional env:
#   PS_ORG            default augno-inc
#   PS_DATABASE       default augno_core
#   PS_PROD_BRANCH    default prod
#   DEPLOY_WAIT_MINUTES  how long to wait for each deploy request to finish (default 240)
#   DEPLOY_POLL_SECONDS  how often to check on it (default 30)

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
DEPLOY_WAIT_MINUTES="${DEPLOY_WAIT_MINUTES:-240}"
DEPLOY_POLL_SECONDS="${DEPLOY_POLL_SECONDS:-30}"

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

dr_url() {
    echo "https://app.planetscale.com/$PS_ORG/$PS_DATABASE/deploy-requests/${1:-}"
}

# The deploy request JSON for a branch name, or nothing when it has none. `show` takes a branch name as
# readily as a number and still resolves after --auto-delete-branch has removed the branch.
dr_json() {
    local out
    out="$(pscale_cmd deploy-request show "$PS_DATABASE" "$1" --format json 2>/dev/null)" || true
    # pscale reports "not found" as JSON on stdout too; only a deploy request has a number.
    if echo "$out" | jq -e '.number' >/dev/null 2>&1; then
        echo "$out"
    fi
}

dr_field() {
    echo "$1" | jq -r "$2 // empty" 2>/dev/null || true
}

deployment_state() {
    dr_field "$1" '.deployment_state // .deployment.state'
}

# Live means this deploy request belongs to the release as it stands: still open, or already deployed.
# A closed deploy request whose deployment never completed was superseded by a later prepare run.
is_live() {
    local json="$1"
    [ -n "$json" ] || return 1
    [ "$(dr_field "$json" '.state')" != "closed" ] && return 0
    case "$(deployment_state "$json")" in
        complete|complete_pending_revert) return 0 ;;
    esac
    return 1
}

# Deploys one deploy request and waits for it to finish. Safe to call on one that is already deployed,
# already queued or already running: it only asks PlanetScale to deploy one that is ready and idle.
deploy_and_wait() {
    local branch="$1" json number state deploy_state deadline attempts=0 applied=false

    json="$(dr_json "$branch")"
    number="$(dr_field "$json" '.number')"
    deadline=$(( $(date +%s) + DEPLOY_WAIT_MINUTES * 60 ))

    while :; do
        state="$(dr_field "$json" '.state')"
        deploy_state="$(deployment_state "$json")"
        info "#${number:-?} on $branch: state ${state:-unknown}, deployment ${deploy_state:-none}"

        case "$deploy_state" in
            complete|complete_pending_revert)
                info "#${number:-?} is live. $(dr_url "$number")"
                return 0
                ;;
            no_changes)
                warn "#${number:-?} has no schema changes to deploy."
                return 0
                ;;
            # The schema was deployed and then taken back out, so prod is NOT running this release's
            # schema. Rolling the service images on top of that is exactly what this gate exists to prevent.
            complete_revert|complete_reverted|complete_revert_started|in_progress_revert|in_progress_revert_vschema)
                error "#${number:-?} was reverted (deployment state '$deploy_state'); its schema is not live."
                error "Re-cut the deploy request before releasing. See $(dr_url "$number")"
                return 1
                ;;
            # PlanetScale's terminal failures; they also leave the deploy queue blocked, which the next
            # release would hit as an unrelated-looking failure.
            complete_error|complete_revert_error|error|failed|complete_cancel|cancelled|canceled|in_progress_cancel)
                error "#${number:-?} ended in deployment state '$deploy_state'. See $(dr_url "$number")"
                error "The deploy queue may be blocked: pscale deploy-request unblock $PS_DATABASE ${number:-<number>} --org $PS_ORG"
                return 1
                ;;
            ready)
                # A refusal can be a race with a deploy that just started, so the next look decides; one
                # still sitting ready after that is asked again, a few times at most.
                if [ "$attempts" -ge 3 ]; then
                    error "PlanetScale would not start #${number:-?} after $attempts attempts. See $(dr_url "$number")"
                    return 1
                fi
                info "Deploying #${number:-?} into $PS_PROD_BRANCH..."
                pscale_cmd deploy-request deploy "$PS_DATABASE" "$number" \
                    || warn "PlanetScale refused the deploy; checking whether it is already under way."
                attempts=$((attempts + 1))
                ;;
            pending_cutover)
                # Auto-apply cuts over by itself; this only nudges one opened without it.
                if [ "$applied" = false ]; then
                    pscale_cmd deploy-request apply "$PS_DATABASE" "$number" >/dev/null 2>&1 || true
                    applied=true
                fi
                ;;
            # pending (diff still being computed), queued, submitting, in_progress*, and anything newer
            # than this script: keep waiting.
        esac

        if [ "$(date +%s)" -ge "$deadline" ]; then
            error "#${number:-?} did not finish within $DEPLOY_WAIT_MINUTES minutes (deployment state '${deploy_state:-none}')."
            error "It may still complete; check $(dr_url "$number") and re-run this job once it has."
            return 1
        fi

        sleep "$DEPLOY_POLL_SECONDS"
        json="$(dr_json "$branch")"
        if [ -z "$json" ]; then
            warn "Could not read #${number:-?}; retrying."
        fi
    done
}

fail_rollout() {
    error "Schema deploy for release $RELEASE_VERSION failed. The service rollout is gated on this and will not run."
    exit 1
}

# --- Which deploy requests? ---

# The workflow only calls this when the release changes shared/db/migrations, so a missing deploy
# request means the prepare step never produced one — not that there is nothing to apply. v2.6.4
# passed here while goose was panicking in prepare, and shipped without its schema.

FIRST_PART_BRANCH="$(planetscale_release_part_branch "$BRANCH" 1)"
FIRST_PART_JSON="$(dr_json "$FIRST_PART_BRANCH")"

if is_live "$FIRST_PART_JSON"; then
    read -r PLAN_ID _ PART_COUNT <<<"$(planetscale_parse_plan_marker "$(dr_field "$FIRST_PART_JSON" '.notes')") "
    if [ -z "${PLAN_ID:-}" ] || [ -z "${PART_COUNT:-}" ]; then
        error "Deploy request on $FIRST_PART_BRANCH has no release-plan marker in its notes; cannot tell how many parts this release has."
        error "Re-run the prepare step on the release PR. See $(dr_url "$(dr_field "$FIRST_PART_JSON" '.number')")"
        exit 1
    fi

    info "Release $RELEASE_VERSION schema is split into $PART_COUNT deploy requests (plan $PLAN_ID)."

    # Every part has to come from the same prepare run as the first: a part left by an earlier run
    # describes a different split, and deploying it would ship the wrong schema.
    PARTS=()
    for part in $(seq 1 "$PART_COUNT"); do
        part_branch="$(planetscale_release_part_branch "$BRANCH" "$part")"
        part_json="$(dr_json "$part_branch")"
        part_marker="$(planetscale_parse_plan_marker "$(dr_field "$part_json" '.notes')")"

        if [ "$part_marker" != "$PLAN_ID $part $PART_COUNT" ] || ! is_live "$part_json"; then
            error "Part $part of $PART_COUNT ($part_branch) is missing or from a different prepare run."
            error "The prepare step on the release PR must not have finished; re-run it before releasing."
            exit 1
        fi
        PARTS+=("$part_branch")
    done

    for part_branch in "${PARTS[@]}"; do
        deploy_and_wait "$part_branch" || fail_rollout
    done

    info "Schema for release $RELEASE_VERSION is live ($PART_COUNT deploy requests)."
    exit 0
fi

info "Looking for the deploy request on branch $BRANCH..."

DR_JSON="$(dr_json "$BRANCH")"
if [ -z "$DR_JSON" ]; then
    error "No deploy request found for $BRANCH, but this release changes schema migrations."
    error "The prepare step on the release PR must have failed; fix it and re-run before releasing."
    exit 1
fi

# Re-running a release must not fail on schema that is already live: a closed deploy request has
# either been deployed or deliberately abandoned, and either way there is nothing here to apply.
# (A deployed one is reported as live by deploy_and_wait below.)
if [ "$(dr_field "$DR_JSON" '.state')" = "closed" ] && ! is_live "$DR_JSON"; then
    # A later prepare run that split the release closes this one as superseded. If the first part only
    # failed to load a moment ago, this is that case, and passing here would roll out without the schema.
    sleep 5
    if is_live "$(dr_json "$FIRST_PART_BRANCH")"; then
        error "Deploy request on $BRANCH was superseded by a split release on $FIRST_PART_BRANCH, which could not be read. Re-run this job."
        exit 1
    fi
    info "Deploy request #$(dr_field "$DR_JSON" '.number') is already closed. Nothing to deploy."
    exit 0
fi

deploy_and_wait "$BRANCH" || fail_rollout

info "Schema for release $RELEASE_VERSION is live."
