#!/bin/bash

# dispatch-infra-deploy.sh
# Asks the private open-mrp/infra repo to roll this release onto EKS, then blocks until that run
# finishes and exits with its result.
#
# This repo can build and push images; it deliberately cannot deploy them. The rollout runs in
# open-mrp/infra behind that repo's `production` environment, so the credential that can mutate the
# cluster never exists here. Blocking rather than firing-and-forgetting preserves the release
# ordering the pipeline has always had: nothing publishes an OpenAPI spec or an SDK for code that
# is not live yet.
#
# Required env:
#   GH_TOKEN            token with `actions: write` on open-mrp/infra
#   IMAGE_TAG           release tag whose images are in ECR
# Optional env:
#   DEPLOY_SERVICES     comma-separated services to roll (empty applies cluster config only)
#   MCP_IMAGE_TAG       if set, mcp-server is rolled to this tag too
#   INFRA_REPO          default open-mrp/infra
#   INFRA_REF           default main
#   DEPLOY_TIMEOUT_SECS default 2400

set -euo pipefail

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

print_status() { echo -e "${GREEN}[INFO]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1" >&2; }

INFRA_REPO=${INFRA_REPO:-open-mrp/infra}
INFRA_REF=${INFRA_REF:-main}
WORKFLOW=${WORKFLOW:-deploy.yml}
DEPLOY_TIMEOUT_SECS=${DEPLOY_TIMEOUT_SECS:-2400}
DEPLOY_SERVICES=${DEPLOY_SERVICES:-}
MCP_IMAGE_TAG=${MCP_IMAGE_TAG:-}

if [ -z "${IMAGE_TAG:-}" ]; then
    print_error "IMAGE_TAG is required."
    exit 1
fi

# The dispatch API returns nothing identifying, so tag the run and find it by name. GITHUB_RUN_ID is
# unique per attempt, which keeps a re-run from latching onto the previous attempt's deploy.
CORRELATION_ID="${GITHUB_RUN_ID:-local}-${GITHUB_RUN_ATTEMPT:-1}"

print_status "Dispatching ${WORKFLOW} in ${INFRA_REPO} for ${IMAGE_TAG} [${CORRELATION_ID}]"
print_status "  services: ${DEPLOY_SERVICES:-(cluster config only)}"
[ -n "$MCP_IMAGE_TAG" ] && print_status "  mcp-server: ${MCP_IMAGE_TAG}"

gh workflow run "$WORKFLOW" \
    --repo "$INFRA_REPO" \
    --ref "$INFRA_REF" \
    -f image_tag="$IMAGE_TAG" \
    -f deploy_services="$DEPLOY_SERVICES" \
    -f build_services="$DEPLOY_SERVICES" \
    -f mcp_image_tag="$MCP_IMAGE_TAG" \
    -f correlation_id="$CORRELATION_ID"

# The run does not exist the instant the dispatch returns; poll for it by correlation id.
print_status "Waiting for the deploy run to appear..."
run_id=""
for _ in $(seq 1 30); do
    sleep 5
    run_id=$(gh run list \
        --repo "$INFRA_REPO" \
        --workflow "$WORKFLOW" \
        --limit 30 \
        --json databaseId,displayTitle \
        --jq "[.[] | select(.displayTitle | contains(\"[${CORRELATION_ID}]\"))] | first | .databaseId // empty")
    [ -n "$run_id" ] && break
done

if [ -z "$run_id" ]; then
    print_error "No ${INFRA_REPO} deploy run appeared for ${CORRELATION_ID} within 150s."
    print_error "Check https://github.com/${INFRA_REPO}/actions/workflows/${WORKFLOW}"
    exit 1
fi

run_url="https://github.com/${INFRA_REPO}/actions/runs/${run_id}"
print_status "Deploy run: ${run_url}"

deadline=$((SECONDS + DEPLOY_TIMEOUT_SECS))
while [ "$SECONDS" -lt "$deadline" ]; do
    read -r status conclusion <<<"$(gh run view "$run_id" --repo "$INFRA_REPO" \
        --json status,conclusion --jq '"\(.status) \(.conclusion // "")"')"

    if [ "$status" = "completed" ]; then
        if [ "$conclusion" = "success" ]; then
            print_status "Deploy succeeded."
            exit 0
        fi
        # A deploy awaiting environment approval that times out reports as cancelled/failure here;
        # the run page says which, so point at it rather than guessing.
        print_error "Deploy finished with conclusion '${conclusion}'. See ${run_url}"
        exit 1
    fi

    sleep 15
done

print_error "Deploy did not finish within ${DEPLOY_TIMEOUT_SECS}s. See ${run_url}"
print_error "It may still be waiting for approval on the ${INFRA_REPO} 'production' environment."
exit 1
