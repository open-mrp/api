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
# Reruns are cheap when nothing changed. A release PR is rebuilt on every push to main, but the
# deploy request is only recut when the release's new migrations (or the tag they are measured from)
# differ from what the open deploy request was cut from; its notes carry that fingerprint. Deploy
# requests and branches an earlier run left for a version this release PR no longer carries (a feat
# landing bumps 4.6.1 to 4.7.0) are closed and deleted.
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
#   superseded_md     markdown links to this release's deploy requests closed here, comma-separated

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

source "$SCRIPT_DIR/release-migration-changes.sh"
PREVIOUS_TAG="$(release_previous_tag "$BASE_REF")"

if [ -z "$PREVIOUS_TAG" ]; then
    warn "No previous release tag found; treating every migration as new."
else
    info "Comparing $MIGRATIONS_DIR between $PREVIOUS_TAG and $BASE_REF"
fi

NEW_MIGRATIONS="$(release_new_migrations "$PREVIOUS_TAG" "$BASE_REF" "$MIGRATIONS_DIR")"

if [ -n "$PREVIOUS_TAG" ]; then
    EDITED="$(git diff --name-only "$PREVIOUS_TAG" "$BASE_REF" -- "$MIGRATIONS_DIR" | grep -vxF "${NEW_MIGRATIONS:-//none//}" || true)"
    if [ -n "$EDITED" ]; then
        warn "Changed since $PREVIOUS_TAG but already shipped; goose will not re-run them, so they are ignored:"
        echo "$EDITED" | sed 's/^/  /'
    fi
fi
# shellcheck disable=SC2086 # one path per word; migration file names have no spaces
FINGERPRINT="$(release_migrations_fingerprint "$PREVIOUS_TAG" "$BASE_REF" $NEW_MIGRATIONS)"
FINGERPRINT_MARKER="[release-migrations sha=$FINGERPRINT]"

if [ -n "$NEW_MIGRATIONS" ]; then
    info "Migrations in this release (fingerprint $FINGERPRINT):"
    echo "$NEW_MIGRATIONS" | sed 's/^/  /'
fi

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

dr_field() {
    echo "$1" | jq -r "$2 // empty" 2>/dev/null || true
}

# The deploy request JSON for a branch name, or nothing when it has none. `show` resolves a branch name
# to its most recent deploy request, even after --auto-delete-branch has removed the branch.
dr_json() {
    local out
    out="$(pscale_cmd deploy-request show "$PS_DATABASE" "$1" --format json 2>/dev/null)" || true
    # pscale reports "not found" as JSON on stdout too; only a deploy request has a number.
    if echo "$out" | jq -e '.number' >/dev/null 2>&1; then
        echo "$out"
    fi
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

# Closing a deploy request is what tells the deploy step it was superseded. The ones closed for this
# release are reported, so the release PR can say which deploy request replaced them.
SUPERSEDED=()

retire_branch() {
    local name="$1" open

    open="$(dr_json "$name" | jq -r 'select(.state == "open" or .state == "pending") | .number // empty' 2>/dev/null || true)"
    if [ -n "$open" ]; then
        info "Closing superseded deploy request #$open ($name)"
        pscale_cmd deploy-request close "$PS_DATABASE" "$open" >/dev/null || true
        if [ "$(release_base_of "$name")" = "$BRANCH" ]; then
            SUPERSEDED+=("$open")
        fi
    fi

    if pscale_cmd branch show "$PS_DATABASE" "$name" >/dev/null 2>&1; then
        info "Deleting branch $name left by an earlier run"
        pscale_cmd branch delete "$PS_DATABASE" "$name" --force
    fi
}

emit_superseded() {
    local number links=""
    for number in "${SUPERSEDED[@]}"; do
        links="${links:+$links, }[#${number}](https://app.planetscale.com/$PS_ORG/$PS_DATABASE/deploy-requests/$number)"
    done
    emit superseded_md "$links"
}

retire_release() {
    local base="$1" part name
    retire_branch "$base"
    part=1
    while [ "$part" -le 50 ]; do
        name="$(planetscale_release_part_branch "$base" "$part")"
        if [ -z "$(dr_json "$name")" ] && ! pscale_cmd branch show "$PS_DATABASE" "$name" >/dev/null 2>&1; then
            break
        fi
        retire_branch "$name"
        part=$((part + 1))
    done
}

# The release PR's version moves as commits land (a fix after a feat keeps it, a feat after a fix bumps
# it), and each version gets its own branch name, so the previous version's deploy request would stay
# open forever. Anything on a release branch for a version that is neither this one nor already tagged
# belongs to a release PR that no longer exists. Tagged versions are left alone: their deploy step owns
# them, including a manual resume of the latest release.
RELEASED_BRANCHES="$(git tag --list 'v*' | while IFS= read -r tag; do planetscale_release_branch "$tag" 2>/dev/null || true; done)"

release_base_of() {
    echo "$1" | sed -E 's/-p[0-9]+$//'
}

is_stale_release_branch() {
    local base
    case "$1" in release-[0-9]*) ;; *) return 1 ;; esac
    base="$(release_base_of "$1")"
    [ "$base" != "$BRANCH" ] || return 1
    ! echo "$RELEASED_BRANCHES" | grep -qxF "$base"
}

STALE="$(
    {
        pscale_cmd deploy-request list "$PS_DATABASE" --format json 2>/dev/null |
            jq -r '.[]? | select(.state == "open") | .branch // empty' 2>/dev/null || true
        pscale_cmd branch list "$PS_DATABASE" --format json 2>/dev/null |
            jq -r '.[]? | .name // empty' 2>/dev/null || true
    } | while IFS= read -r name; do
        if [ -n "$name" ] && is_stale_release_branch "$name"; then release_base_of "$name"; fi
    done | sort -u
)"
for base in $STALE; do
    info "Retiring $base: its version is no longer the one this release PR carries."
    retire_release "$base"
done

if [ -z "$NEW_MIGRATIONS" ]; then
    # A migration an earlier run prepared may since have been reverted on main.
    retire_release "$BRANCH"
    info "No new migrations in this release. Nothing to deploy."
    emit has_migrations false
    emit_superseded
    exit 0
fi

emit has_migrations true

# --- Already prepared? ---

# Most pushes to main while a release PR is open change no migrations. If every deploy request this
# release has is still open and was cut from the same fingerprint, a recut would produce the same diff.
# no_changes is not reusable: these migrations do change the schema, so that diff is wrong (#256).
open_from_fingerprint() {
    [ "$(dr_field "$1" '.state')" = "open" ] || return 1
    case "$(dr_field "$1" '.deployment_state // .deployment.state')" in
        pending|ready) ;;
        *) return 1 ;;
    esac
    dr_field "$1" '.notes' | grep -qF "$FINGERPRINT_MARKER"
}

reuse_open_deploy_requests() {
    local json count number url list="" part part_json plan_id

    json="$(dr_json "$FIRST_PART_BRANCH")"
    if [ -n "$json" ] && [ "$(dr_field "$json" '.state')" = "open" ]; then
        read -r plan_id _ count <<<"$(planetscale_parse_plan_marker "$(dr_field "$json" '.notes')") "
        [ -n "${plan_id:-}" ] && [ -n "${count:-}" ] || return 1
        for part in $(seq 1 "$count"); do
            part_json="$(dr_json "$(planetscale_release_part_branch "$BRANCH" "$part")")"
            open_from_fingerprint "$part_json" || return 1
            [ "$(planetscale_parse_plan_marker "$(dr_field "$part_json" '.notes')")" = "$plan_id $part $count" ] || return 1
            number="$(dr_field "$part_json" '.number')"
            [ "$part" -eq 1 ] && FIRST_NUMBER="$number"
            url="https://app.planetscale.com/$PS_ORG/$PS_DATABASE/deploy-requests/$number"
            list="${list}- [#${number}](${url}) on \`$(planetscale_release_part_branch "$BRANCH" "$part")\` — part $part of $count"$'\n'
        done
    else
        json="$(dr_json "$BRANCH")"
        open_from_fingerprint "$json" || return 1
        count=1
        FIRST_NUMBER="$(dr_field "$json" '.number')"
        url="https://app.planetscale.com/$PS_ORG/$PS_DATABASE/deploy-requests/$FIRST_NUMBER"
        list="- [#${FIRST_NUMBER}](${url}) on \`$BRANCH\`"$'\n'
    fi

    info "Deploy request #$FIRST_NUMBER already carries these migrations (fingerprint $FINGERPRINT). Leaving it as it is."
    emit deploy_request "$FIRST_NUMBER"
    emit deploy_request_url "https://app.planetscale.com/$PS_ORG/$PS_DATABASE/deploy-requests/$FIRST_NUMBER"
    emit deploy_request_count "$count"
    emit_multiline deploy_requests_md "$list"
}

if reuse_open_deploy_requests; then
    exit 0
fi

# Branches are recreated rather than reused: a branch left over from an earlier run may have had a
# since-edited migration applied to it. Cutting fresh from prod means each deploy request diff describes
# exactly the migrations in this release. An earlier run may also have split differently, so every part
# branch it left goes too.
retire_release "$BRANCH"

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
        "Automated: schema for release $RELEASE_VERSION. Deployed when the release PR merges. $FINGERPRINT_MARKER" ""
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
            "Automated: schema for release $RELEASE_VERSION, part $part of $PART_COUNT ($tables). Deployed in order when the release PR merges. $(planetscale_plan_marker "$PLAN_ID" "$part" "$PART_COUNT") $FINGERPRINT_MARKER" \
            " — part $part of $PART_COUNT: $tables"
    done

    # The parts carry everything; the full branch only existed to compute them.
    pscale_cmd branch delete "$PS_DATABASE" "$BRANCH" --force
fi

# The deploy step passes a no_changes deploy request as "nothing to deploy", so one that wrongly reads
# no_changes would roll the release out without its schema. #256 did, with 00079 applied to its branch.
# schemasplit's diff of the same branch says whether there really is nothing; wait for PlanetScale's.
if [ "$CAN_SPLIT" = true ] && [ -f "$WORK_DIR/plan/plan.json" ] && [ "$(jq '.tables' "$WORK_DIR/plan/plan.json")" -gt 0 ]; then
    for number in "${DR_NUMBERS[@]}"; do
        deadline=$(( $(date +%s) + 300 ))
        while :; do
            state="$(dr_field "$(dr_json "$number")" '.deployment_state // .deployment.state')"
            case "$state" in ""|pending) ;; *) break ;; esac
            if [ "$(date +%s)" -ge "$deadline" ]; then
                warn "PlanetScale is still computing the diff for #$number; not waiting longer."
                break
            fi
            sleep 5
        done
        if [ "$state" = "no_changes" ]; then
            error "Deploy request #$number reports no schema changes, but this release changes $(jq '.tables' "$WORK_DIR/plan/plan.json") table(s)."
            error "Deploying it would ship nothing. Re-run this job to cut it again."
            pscale_cmd deploy-request close "$PS_DATABASE" "$number" >/dev/null || true
            exit 1
        fi
    done
fi

FIRST_NUMBER="${DR_NUMBERS[0]}"
emit deploy_request "$FIRST_NUMBER"
emit deploy_request_url "https://app.planetscale.com/$PS_ORG/$PS_DATABASE/deploy-requests/$FIRST_NUMBER"
emit deploy_request_count "${#DR_NUMBERS[@]}"
emit_multiline deploy_requests_md "$DR_LIST"
emit_superseded
