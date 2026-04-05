#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

MAX_WORKERS="${MAX_WORKERS:-5}"
WORK_DIR=".auth-pattern-verify"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

usage() {
    cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Runs parallel Claude Code sessions to audit service files for correct
authorization check patterns (cross-account access, permission domain
routing, external target checks).

Options:
  -w, --workers N      Number of parallel workers (default: 5)
  -s, --service NAME   Only audit a specific microservice (e.g. "core-service")
  -n, --dry-run        Show which files would be audited without running
  -r, --resume         Skip files that already have a result from a previous run
  -h, --help           Show this help message

Environment variables:
  MAX_WORKERS          Same as --workers
  CLAUDE_MODEL         Model to use (default: sonnet)
EOF
    exit 0
}

SERVICE_FILTER=""
DRY_RUN=false
RESUME=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        -w|--workers) MAX_WORKERS="$2"; shift 2 ;;
        -s|--service) SERVICE_FILTER="$2"; shift 2 ;;
        -n|--dry-run) DRY_RUN=true; shift ;;
        -r|--resume) RESUME=true; shift ;;
        -h|--help) usage ;;
        *) echo "Unknown option: $1"; usage ;;
    esac
done

if ! command -v claude &>/dev/null; then
    echo -e "${RED}Error:${NC} claude CLI not found in PATH."
    exit 1
fi

mkdir -p "$WORK_DIR/logs" "$WORK_DIR/results" "$WORK_DIR/queue"

rm -f "$WORK_DIR/queue/"*.pending "$WORK_DIR/queue/"*.running "$WORK_DIR/queue/"*.done "$WORK_DIR/queue/"*.failed

safe_name() {
    echo "$1" | tr '/ ' '--' | tr -cd 'a-zA-Z0-9_-.'
}

has_result() {
    local filepath="$1"
    local safe
    safe=$(safe_name "$filepath")
    [ -f "$WORK_DIR/results/${safe}.md" ]
}

discover_files() {
    local search_path="services"
    if [ -n "$SERVICE_FILTER" ]; then
        search_path="services/$SERVICE_FILTER"
        if [ ! -d "$search_path" ]; then
            echo -e "${RED}Error:${NC} Service directory not found: $search_path" >&2
            exit 1
        fi
    fi

    find "$search_path" -path '*/internal/service/*.go' \
        ! -name '*_test.go' \
        ! -name '*mock*' \
        ! -path '*/mock/*' \
        ! -name 'transaction_manager.go' \
        | sort
}

build_queue() {
    local idx=0
    local skipped=0

    while IFS= read -r filepath; do
        if [ "$RESUME" = true ] && has_result "$filepath"; then
            skipped=$((skipped + 1))
            continue
        fi
        printf '%s\n' "$filepath" > "$WORK_DIR/queue/${idx}.pending"
        idx=$((idx + 1))
    done < <(discover_files)

    if [ "$skipped" -gt 0 ]; then
        echo -e "${YELLOW}Skipped $skipped files with existing results (--resume)${NC}" >&2
    fi

    echo "$idx"
}

TOTAL=$(build_queue)

if [ "$TOTAL" -eq 0 ]; then
    echo -e "${GREEN}No service files to audit.${NC}"
    exit 0
fi

echo -e "${CYAN}=== Authorization Pattern Audit ===${NC}"
echo -e "Files to audit:    ${YELLOW}${TOTAL}${NC}"
echo -e "Parallel workers:  ${YELLOW}${MAX_WORKERS}${NC}"
echo ""

if [ "$DRY_RUN" = true ]; then
    echo -e "${BLUE}Dry run — files that would be audited:${NC}"
    for f in "$WORK_DIR/queue/"*.pending; do
        [ -f "$f" ] || continue
        echo "  $(cat "$f")"
    done
    exit 0
fi

build_prompt() {
    local filepath="$1"
    local result_file="$2"

    cat <<'PROMPT_HEADER'
You are auditing and fixing a Go service file for correct authorization check patterns.

## Reference Files

Read these files first to understand the expected patterns:

1. `docs/patterns/authorization-check-patterns.md` — the patterns you are checking against
2. `services/auth-service/pkg/types/identity_model.go` — identity boolean helpers
3. `services/auth-service/pkg/types/identity_utils.go` — error-returning check methods
4. `services/core-service/internal/service/address_service.go` — reference implementation

PROMPT_HEADER

    cat <<PROMPT_FILE

## File to Audit

\`$filepath\`

Read this file and audit every public method (exported, capitalized) that extracts identity from context.
Skip constructors, helpers, and methods that don't handle requests.

**Fix any issues you find** by editing the service file to match the patterns in the reference implementation.

PROMPT_FILE

    cat <<'PROMPT_CHECKS'
## Check Items

For each public service method that extracts identity, verify:

### 1. Target Account Check
- PASS: Uses `!identity.IsTargetAccountSet()` to verify the account header
- FAIL: Uses `identity.TargetAccountID == nil` directly
- SKIP: Method does not check target account (may be valid for some endpoints)

### 2. Permission Check Uses IsInternalActor (not IsInternalUser)
- PASS: Permission branching uses `identity.IsInternalActor()` (works cross-account)
- FAIL: Uses `identity.IsInternalUser()` to gate permission checks (breaks when merchant targets customer)
- SKIP: Method uses `identity.CheckIsInternalActor()` as a gate — this is valid for internal-only endpoints that should NOT support cross-account access

### 3. Permission Domain Routes by Target Relation Type
- PASS: When checking permissions, branches on `IsTargetCustomerAccount()` → customers domain, `IsTargetSupplierAccount()` → suppliers domain, fallback → resource domain
- FAIL: Only checks the resource's own permission domain without considering customer/supplier targets
- SKIP: Method is gated by `CheckIsInternalActor()` (same-account only) or does not check permissions

### 4. External Target Access Check
- PASS: Has `identity.IsExternalTarget()` check followed by `CheckReadAccess` (reads) or `CheckEditAccess` (writes)
- FAIL: Missing external target handling entirely — would allow any related account to access data without access verification
- SKIP: Method is gated by `CheckIsInternalActor()` (same-account only, external targets already rejected)

### 5. Idempotency (for write operations only)
- PASS: POST/PATCH-style methods use `UpsertIdempotencyKey` with recovery point switch
- FAIL: Write method is missing idempotency
- SKIP: Read-only method, or DELETE/PUT (idempotent by design)

PROMPT_CHECKS

    cat <<PROMPT_OUTPUT
## Output

Write your findings to \`$result_file\` in this format:

\`\`\`markdown
# Auth Pattern Audit: $(basename "$filepath")

## Summary
- **File**: \`$filepath\`
- **Methods audited**: N
- **Passed**: N
- **Fixed**: N
- **Skipped**: N

## Details

### MethodName
- **Status**: PASS | FAIL | FIXED | SKIP
- **Issues**: (list specific issues with line numbers, or "None")
- **Fix applied**: (describe what was changed, or "N/A")

(repeat for each method)
\`\`\`

Be thorough. Check every exported method. Fix every issue you find.
PROMPT_OUTPUT
}

claim_next() {
    for f in "$WORK_DIR/queue/"*.pending; do
        [ -f "$f" ] || continue
        local running="${f%.pending}.running"
        if mv "$f" "$running" 2>/dev/null; then
            echo "$running"
            return 0
        fi
    done
    return 1
}

worker() {
    local wid=$1

    while true; do
        local claimed
        if ! claimed=$(claim_next); then
            break
        fi

        local filepath
        filepath=$(cat "$claimed")
        local safe
        safe=$(safe_name "$filepath")
        local result_file="$WORK_DIR/results/${safe}.md"
        local log_file="$WORK_DIR/logs/${safe}.log"

        echo -e "${BLUE}[Worker $wid]${NC} > Starting: ${YELLOW}${filepath}${NC}"

        local prompt
        prompt=$(build_prompt "$filepath" "$result_file")

        local model="${CLAUDE_MODEL:-sonnet}"
        local claude_args=(
            -p "$prompt"
            --dangerously-skip-permissions
            --model "$model"
            -n "auth-verify: $safe"
        )

        if claude "${claude_args[@]}" > "$log_file" 2>&1; then
            mv "$claimed" "${claimed%.running}.done"
            echo -e "${GREEN}[Worker $wid]${NC} OK: ${GREEN}${filepath}${NC}"
        else
            mv "$claimed" "${claimed%.running}.failed"
            echo -e "${RED}[Worker $wid]${NC} FAIL: ${RED}${filepath}${NC}"
        fi
    done

    echo -e "${CYAN}[Worker $wid]${NC} No more files. Exiting."
}

shutdown() {
    echo ""
    echo -e "${YELLOW}Shutting down workers...${NC}"
    kill $(jobs -p) 2>/dev/null || true
    wait 2>/dev/null || true
    show_summary
    exit 130
}

trap shutdown SIGINT SIGTERM

show_summary() {
    local done_count=0 fail_count=0
    for f in "$WORK_DIR/queue/"*.done; do [ -f "$f" ] && done_count=$((done_count + 1)); done
    for f in "$WORK_DIR/queue/"*.failed; do [ -f "$f" ] && fail_count=$((fail_count + 1)); done
    local remaining=$((TOTAL - done_count - fail_count))

    echo ""
    echo -e "${CYAN}=== Audit Summary ===${NC}"
    echo -e "  ${GREEN}Completed: $done_count${NC}"
    echo -e "  ${RED}Failed:    $fail_count${NC}"
    echo -e "  Remaining: $remaining"
    echo ""
    echo -e "  Logs:    $WORK_DIR/logs/"
    echo -e "  Results: $WORK_DIR/results/"

    if [ "$fail_count" -gt 0 ]; then
        echo ""
        echo -e "${YELLOW}Failed files:${NC}"
        for f in "$WORK_DIR/queue/"*.failed; do
            [ -f "$f" ] || continue
            echo -e "  ${RED}$(cat "$f")${NC}"
        done
    fi
}

pids=()
for i in $(seq 1 "$MAX_WORKERS"); do
    worker "$i" &
    pids+=($!)
done

for pid in "${pids[@]}"; do
    wait "$pid" || true
done

show_summary
