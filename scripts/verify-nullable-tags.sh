#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

MAX_WORKERS="${MAX_WORKERS:-5}"
WORK_DIR=".nullable-tags-verify"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

usage() {
    cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Runs parallel Claude Code sessions to audit api-gateway update endpoint
request structs for missing \`nullable:"false"\` tags on pointer fields
that should not accept explicit JSON \`null\` values.

For each update endpoint file, an agent:
  1. Reads the request struct and traces the field through the gRPC
     handler, domain service, repository, and SQL.
  2. Decides which pointer fields are non-nullable (cannot be cleared)
     vs. legitimately nullable (can be cleared via explicit null).
  3. Adds \`nullable:"false"\` to fields that should never accept null.
  4. Leaves fields that are legitimately clearable alone.

Options:
  -w, --workers N      Number of parallel workers (default: 5)
  -s, --section NAME   Only audit a specific endpoint subdirectory
                       (e.g. "item-categories", "unit-groups")
  -n, --dry-run        Show which files would be audited without running
  -r, --resume         Skip files that already have a result from a previous run
  -h, --help           Show this help message

Environment variables:
  MAX_WORKERS          Same as --workers
  CLAUDE_MODEL         Model to use (default: sonnet)
EOF
    exit 0
}

SECTION_FILTER=""
DRY_RUN=false
RESUME=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        -w|--workers) MAX_WORKERS="$2"; shift 2 ;;
        -s|--section) SECTION_FILTER="$2"; shift 2 ;;
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
    local search_path="services/api-gateway/endpoints"
    if [ -n "$SECTION_FILTER" ]; then
        search_path="services/api-gateway/endpoints/$SECTION_FILTER"
        if [ ! -d "$search_path" ]; then
            echo -e "${RED}Error:${NC} Section directory not found: $search_path" >&2
            exit 1
        fi
    fi

    # Update endpoints only — exclude test files and analytics/bulk-ops areas
    # which are pending a separate refactor.
    find "$search_path" -type f -name 'endpoint_update_*.go' \
        ! -name '*_test.go' \
        ! -path '*/analytics/*' \
        ! -path '*/bulk-ops/*' \
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
    echo -e "${GREEN}No update endpoint files to audit.${NC}"
    exit 0
fi

echo -e "${CYAN}=== Nullable Tag Audit ===${NC}"
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
You are auditing and fixing a single api-gateway update endpoint request
struct for correct `nullable` struct tags on its pointer fields.

## Background

In the api-gateway, PATCH-style update requests use pointer fields so that
"omitted" (nil) means "do not change this field". The `nullable` struct tag
controls how the field is documented in the OpenAPI spec AND whether
runtime middleware will reject explicit JSON `null`.

- `nullable:"false"` — explicit JSON `null` is rejected at the request
  validator (`shared/validate/json_null.go` -> `RejectExplicitJSONNulls`)
  with a 400 `invalid_format` error. Use this when the underlying field
  cannot be cleared (the column is NOT NULL, or the SQL UPDATE uses
  `COALESCE(?, col)` which makes a NULL bind a no-op).
- `nullable:"true"` — explicit JSON `null` is accepted and converted to
  the empty-string sentinel by `ApplyExplicitNulls` so the service can
  distinguish "not provided" from "clear this field". Use this ONLY when
  there is service/repo code that actually handles clearing the field
  (look for `*params.X == ""` checks or SQL that writes NULL directly).
- No `nullable` tag on a pointer field — the OpenAPI generator defaults
  to `nullable: true` based on the pointer, but no runtime enforcement
  exists. This is almost always wrong on update endpoints.

## Reference Files

Read these to understand the convention before auditing:

1. `shared/validate/json_null.go` — runtime enforcement of `nullable:"false"`
   and the `ApplyExplicitNulls` sentinel pattern for `nullable:"true"`.
2. `tools/apidocs/generator.go` (around line 600-625) — how the tag flows
   into the OpenAPI schema.
3. `services/api-gateway/endpoints/account-groups/endpoint_update_account_group.go`
   — a clean reference showing `nullable:"false"` on Name and policy fields.
4. `services/api-gateway/endpoints/product-lines/endpoint_update_product_line.go`
   — another clean reference (Name, CommissionPolicy, FreightPolicy,
   UnitGroupID all `nullable:"false"`).

PROMPT_HEADER

    cat <<PROMPT_FILE

## File to Audit

\`$filepath\`

Read this file and audit every pointer field on the request struct(s)
declared in it (typically named \`Update*Request\`).

PROMPT_FILE

    cat <<'PROMPT_PROCESS'
## Process

For each pointer field on the update request struct:

1. **Check the current `nullable` tag.**
   - If `nullable:"false"` already → leave it.
   - If `nullable:"true"` → verify the service/repo actually handles
     clearing (empty-string sentinel or direct NULL write). If it does
     NOT, change to `nullable:"false"`. If it does, leave it.
   - If no `nullable` tag → determine the correct value (see step 2).

2. **Trace the field downstream** to determine nullability:
   - The api-gateway service file in the same package
     (`service.go`) — see how the field is mapped into the gRPC request.
   - The gRPC handler in
     `services/<service>/internal/infrastructure/grpc/grpc_*_handler.go`.
   - The domain service in
     `services/<service>/internal/service/<entity>_service.go`.
   - The repository in
     `services/<service>/internal/infrastructure/repository/<entity>_repository.go`.
   - The generated SQL in
     `services/<service>/internal/infrastructure/sqlc/<entity>.sql.go`.

3. **Apply the heuristics** in priority order:
   a. **SQL signal:** If the UPDATE statement uses `COALESCE(?, col)` for
      this column, NULL is a no-op and the field MUST be `nullable:"false"`.
   b. **DB column signal:** If the column is NOT NULL in the schema, the
      field MUST be `nullable:"false"`.
   c. **Required-on-create signal:** If the corresponding
      `endpoint_create_*.go` file in the same package marks this field
      with `validate:"required"`, it almost certainly should be
      `nullable:"false"` on update too — you cannot clear something that
      is required to exist.
   d. **Explicit clearing signal:** If the service or repository has a
      branch like `if *params.X == "" { ... write NULL ... }` or sets a
      `sql.NullString{Valid: false}` based on the input being empty/null,
      then `nullable:"true"` is intentional — leave it alone.

4. **Apply fixes** by editing the request struct. Add or change the
   `nullable` tag. Preserve all other tags (`json`, `validate`, `format`,
   `default`, etc.) exactly. Tag ordering convention in this codebase is
   `json` first, then `nullable`, then `validate`/format/default.

5. **Do not** change any code outside the request struct. Do not change
   the JSON field name, do not add or remove fields, do not touch
   validators.

6. **Skip** non-pointer fields (string, int, etc. by value) — they cannot
   be `null` in JSON regardless and are typically required path params.

7. **Skip** fields whose comments or surrounding context clearly indicate
   they are intentionally clearable (e.g. "set to null to remove the
   default address"). When in doubt, prefer `nullable:"false"` for fields
   like names, IDs, codes, and required references; prefer leaving
   alone for fields like `description`, `notes`, `image_url`, optional
   foreign keys to deletable resources.

PROMPT_PROCESS

    cat <<PROMPT_OUTPUT
## Output

After making any edits, write your findings to \`$result_file\` in this
format:

\`\`\`markdown
# Nullable Tag Audit: $(basename "$filepath")

## Summary
- **File**: \`$filepath\`
- **Struct(s) audited**: <names>
- **Pointer fields reviewed**: N
- **Fixed (added/changed nullable:"false")**: N
- **Already correct**: N
- **Left as nullable (intentional)**: N

## Details

### FieldName (\`json_name\`)
- **Status**: FIXED | ALREADY_CORRECT | LEFT_NULLABLE | SKIPPED
- **Evidence**: brief — e.g. "SQL uses COALESCE(?, name)" or
  "service.go:45 has empty-string sentinel handling for clearing"
- **Old tag**: \`...\` (only if changed)
- **New tag**: \`...\` (only if changed)

(repeat for each pointer field)

## Notes
Anything ambiguous or worth a human review.
\`\`\`

Be precise. Cite file paths and line numbers in your evidence. Do not
modify any other endpoint files.
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
            -n "nullable-verify: $safe"
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
