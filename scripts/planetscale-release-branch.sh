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
#   PS_MAX_TABLES_PER_DEPLOY_REQUEST
#                     most tables one deploy request may change (default 10, PlanetScale's limit)
#
# A release that changes more tables than one deploy request allows is split: the migrations are still
# applied once to the release branch, then `schemasplit` diffs that branch against prod and packs the
# per-table DDL into parts of at most PS_MAX_TABLES_PER_DEPLOY_REQUEST tables. Each part gets its own
# branch cut from prod (<branch>-p1, -p2, ...) and its own deploy request, and the deploy step deploys
# them one after another. A release within the limit opens one deploy request from <branch>, as always.
#
# Outputs (written to $GITHUB_OUTPUT when set):
#   has_migrations    true|false
#   branch            the PlanetScale branch name
#   deploy_request    deploy request number (the first part's, when split)
#   deploy_request_url
#   deploy_request_count
#   deploy_requests_md  markdown list of every deploy request, one per line

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
PS_MAX_TABLES_PER_DEPLOY_REQUEST="${PS_MAX_TABLES_PER_DEPLOY_REQUEST:-10}"
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

emit_multiline() {
    if [ -n "${GITHUB_OUTPUT:-}" ]; then
        local delimiter="EOF_${RANDOM}${RANDOM}"
        { echo "$1<<$delimiter"; echo "$2"; echo "$delimiter"; } >> "$GITHUB_OUTPUT"
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

# --- PlanetScale ---

pscale_cmd() {
    pscale "$@" --org "$PS_ORG"
}

# The deploy request JSON for a branch name, or nothing when it has none. `show` resolves a branch name
# to its most recent deploy request, even after --auto-delete-branch has removed the branch.
dr_json() {
    pscale_cmd deploy-request show "$PS_DATABASE" "$1" --format json 2>/dev/null || true
}

FIRST_PART_BRANCH="$(planetscale_release_part_branch "$BRANCH" 1)"

# --- Already deployed? ---

# Once this release's deploy request has been applied, prod carries the release's migrations. Re-cutting
# the branch from prod and replaying `up` would then collide on the first non-idempotent DDL — mark-shipped
# only records migrations from the *previous* release, so a current-release migration already live in prod
# gets replayed (deploy request #209: 00018 re-added an index prod already had, errno 1061). The deploy
# step (planetscale-deploy-release.sh) already treats a completed deploy request as "nothing to do"; the
# prepare step has to be just as idempotent. A split release is checked by its first part.
for head in "$BRANCH" "$FIRST_PART_BRANCH"; do
    DEPLOYED_STATE="$(dr_json "$head" | jq -r '.deployment_state // .deployment.state // empty' 2>/dev/null || true)"

    case "$DEPLOYED_STATE" in
        # complete / complete_pending_revert both mean the schema is live in prod (the latter is just inside
        # the 30-minute revert window). A reverted deploy is deliberately not matched: prod no longer carries
        # the schema, so a fresh branch and deploy request still need to be cut.
        complete|complete_pending_revert)
            info "Release $RELEASE_VERSION schema is already live in prod ($head, deploy request state '$DEPLOYED_STATE'). Nothing to prepare."
            exit 0
            ;;
    esac
done

# --- Clear out earlier runs ---

# Branches are recreated rather than reused. A release PR is rebuilt every time a commit lands on main,
# and a branch left over from an earlier run may have had a since-edited migration applied to it. Cutting
# fresh from prod means each deploy request diff describes exactly the migrations in this release. An
# earlier run may also have split differently, so every part branch it left goes too; closing their
# deploy requests is what tells the deploy step they were superseded.
retire_branch() {
    local name="$1" open

    open="$(dr_json "$name" | jq -r 'select(.state == "open" or .state == "pending") | .number // empty' 2>/dev/null || true)"
    if [ -n "$open" ]; then
        info "Closing superseded deploy request #$open ($name)"
        pscale_cmd deploy-request close "$PS_DATABASE" "$open" >/dev/null || true
    fi

    if pscale_cmd branch show "$PS_DATABASE" "$name" >/dev/null 2>&1; then
        info "Deleting branch $name left by an earlier run"
        pscale_cmd branch delete "$PS_DATABASE" "$name" --force
    fi
}

retire_branch "$BRANCH"

part=1
while [ "$part" -le 50 ]; do
    name="$(planetscale_release_part_branch "$BRANCH" "$part")"
    if [ -z "$(dr_json "$name")" ] && ! pscale_cmd branch show "$PS_DATABASE" "$name" >/dev/null 2>&1; then
        break
    fi
    retire_branch "$name"
    part=$((part + 1))
done

# --- Apply migrations ---

# schemasplit plans the split. CI builds it before this runs; locally it is built on demand. Without it
# the release still gets its single deploy request, which is all a release within the limit needs.
ensure_schemasplit() {
    if command -v schemasplit >/dev/null 2>&1; then
        return 0
    fi
    if ! command -v go >/dev/null 2>&1; then
        return 1
    fi
    local bin_dir
    bin_dir="$(mktemp -d)"
    (cd "$REPO_ROOT/tools" && go build -o "$bin_dir/schemasplit" ./schemasplit) || return 1
    export PATH="$bin_dir:$PATH"
}

WORK_DIR="$(mktemp -d)"
CAN_SPLIT=false
if ensure_schemasplit; then
    CAN_SPLIT=true
    export SCHEMA_BEFORE="$WORK_DIR/before.sql"
    export SCHEMA_AFTER="$WORK_DIR/after.sql"
else
    warn "schemasplit is unavailable; a release over $PS_MAX_TABLES_PER_DEPLOY_REQUEST tables will not be split."
fi

info "Creating branch $BRANCH from $PS_PROD_BRANCH..."
pscale_cmd branch create "$PS_DATABASE" "$BRANCH" --from "$PS_PROD_BRANCH" --wait

# `pscale connect` opens a local proxy and runs the command with DATABASE_URL pointed at it, so no
# branch password is ever created, stored, or left behind for cleanup.
#
# pscale owns the exit status of --execute, and a swallowed non-zero there would look like a clean run
# with an empty schema diff. The sentinel is the independent proof that the command actually finished.
run_on_branch() {
    local branch="$1" script="$2" sentinel

    sentinel="$(mktemp)"
    rm -f "$sentinel"
    export MIGRATE_SENTINEL="$sentinel"

    pscale_cmd connect "$PS_DATABASE" "$branch" \
        --execute-protocol mysql \
        --execute "$script"

    if [ ! -f "$sentinel" ]; then
        error "$(basename "$script") did not complete successfully on $branch."
        exit 1
    fi
    rm -f "$sentinel"
}

info "Applying migrations to $BRANCH..."
run_on_branch "$BRANCH" "$SCRIPT_DIR/planetscale-apply-migrations.sh"

# --- Split over the table limit ---

PART_COUNT=1
if [ "$CAN_SPLIT" = true ]; then
    info "Planning deploy requests of at most $PS_MAX_TABLES_PER_DEPLOY_REQUEST tables..."
    if schemasplit plan -from "$SCHEMA_BEFORE" -to "$SCHEMA_AFTER" \
        -max-tables "$PS_MAX_TABLES_PER_DEPLOY_REQUEST" -out "$WORK_DIR/plan"; then
        PART_COUNT="$(jq '.parts | length' "$WORK_DIR/plan/plan.json")"
    else
        warn "Could not plan a split; opening a single deploy request. PlanetScale will refuse to deploy it if it is over the limit."
    fi
fi

# --- Open the deploy request(s) ---

DR_NUMBERS=()
DR_LIST=""

open_deploy_request() {
    local branch="$1" notes="$2" label="$3" json number url

    json="$(pscale_cmd deploy-request create "$PS_DATABASE" "$branch" \
        --into "$PS_PROD_BRANCH" \
        --enable-auto-apply \
        --auto-delete-branch \
        --notes "$notes" \
        --format json)"

    number="$(echo "$json" | jq -r '.number')"
    if [ -z "$number" ] || [ "$number" = "null" ]; then
        error "Could not read the deploy request number from pscale:"
        echo "$json" >&2
        exit 1
    fi

    url="https://app.planetscale.com/$PS_ORG/$PS_DATABASE/deploy-requests/$number"
    DR_NUMBERS+=("$number")
    DR_LIST="${DR_LIST}- [#${number}](${url}) on \`${branch}\`${label}"$'\n'
    info "Deploy request #$number is ready for review: $url"
}

if [ "$PART_COUNT" -le 1 ]; then
    info "Creating deploy request into $PS_PROD_BRANCH..."
    open_deploy_request "$BRANCH" \
        "Automated: schema for release $RELEASE_VERSION. Deployed when the release PR merges." ""
else
    info "Release changes more than $PS_MAX_TABLES_PER_DEPLOY_REQUEST tables; opening $PART_COUNT deploy requests."

    PLAN_ID="${GITHUB_RUN_ID:-local}-${GITHUB_RUN_ATTEMPT:-1}-$(date +%s)"
    export SCHEMA_PART_FILE

    for part in $(seq 1 "$PART_COUNT"); do
        part_branch="$(planetscale_release_part_branch "$BRANCH" "$part")"
        tables="$(jq -r ".parts[$((part - 1))].tables | join(\", \")" "$WORK_DIR/plan/plan.json")"

        info "Part $part/$PART_COUNT on $part_branch: $tables"
        pscale_cmd branch create "$PS_DATABASE" "$part_branch" --from "$PS_PROD_BRANCH" --wait

        SCHEMA_PART_FILE="$WORK_DIR/plan/part-$part.sql"
        run_on_branch "$part_branch" "$SCRIPT_DIR/planetscale-apply-schema-part.sh"

        open_deploy_request "$part_branch" \
            "Automated: schema for release $RELEASE_VERSION, part $part of $PART_COUNT ($tables). Deployed in order when the release PR merges. $(planetscale_plan_marker "$PLAN_ID" "$part" "$PART_COUNT")" \
            " — part $part of $PART_COUNT: $tables"
    done

    # The parts carry everything; the full branch only existed to compute them.
    pscale_cmd branch delete "$PS_DATABASE" "$BRANCH" --force
fi

FIRST_NUMBER="${DR_NUMBERS[0]}"
emit deploy_request "$FIRST_NUMBER"
emit deploy_request_url "https://app.planetscale.com/$PS_ORG/$PS_DATABASE/deploy-requests/$FIRST_NUMBER"
emit deploy_request_count "${#DR_NUMBERS[@]}"
emit_multiline deploy_requests_md "$DR_LIST"
