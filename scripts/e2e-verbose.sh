#!/usr/bin/env bash
#
# Verbose E2E runner — the same steps as `make e2e`, but nothing is hidden.
#
# Why this exists: `make e2e` funnels every step through scripts/run-quiet.sh,
# which buffers output to a temp file and only prints it *on failure*. When a
# step HANGS instead of failing (e.g. `build --parallel` thrashing the VM, or
# `up -d --wait` blocking on a service that never goes healthy), you see nothing
# and have no way to tell why. This script streams all output live, builds
# serially, and on a health-wait timeout dumps the logs of whatever is stuck.
#
# Usage:
#   scripts/e2e-verbose.sh [test-run-regex]
#
# Env:
#   SKIP_BUILD=1     reuse existing images (fast reruns; skip the serial build)
#   TIMEOUT=600s     go test timeout (default 600s)
#   HEALTH_WAIT=420  seconds to wait for services to become healthy (default 420)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

RUN_REGEX="${1:-}"
TIMEOUT="${TIMEOUT:-600s}"
HEALTH_WAIT="${HEALTH_WAIT:-420}"
COMPOSE_FILE="docker-compose.e2e.yml"

# Force compose to do one thing at a time — parallel builds thrash the VM and
# are the usual cause of a silent wedge. Disable bake so COMPOSE_PARALLEL_LIMIT
# actually governs build concurrency.
export COMPOSE_PARALLEL_LIMIT=1
export COMPOSE_BAKE=false

compose() { docker compose -f "$COMPOSE_FILE" "$@"; }

hr()   { printf '\n\033[1;34m==== %s ====\033[0m\n' "$1"; }
info() { printf '\033[0;32m[INFO]\033[0m %s\n' "$1"; }
warn() { printf '\033[0;33m[WARN]\033[0m %s\n' "$1"; }
err()  { printf '\033[0;31m[ERROR]\033[0m %s\n' "$1" >&2; }

# ---------------------------------------------------------------------------
hr "1/5  Regenerate OpenAPI spec (gateway reads it at build time)"
# openapi-quiet suppresses its own output; run it but keep going on the spec.
make openapi-quiet

# ---------------------------------------------------------------------------
if [[ "${SKIP_BUILD:-0}" == "1" ]]; then
	hr "2/5  Build images  (SKIPPED — SKIP_BUILD=1)"
else
	hr "2/5  Build images  (serial via COMPOSE_PARALLEL_LIMIT=1, live output)"
	# --progress=plain gives full, streaming build logs (no collapsing TUI), so a
	# stall is visible on the exact step it's stuck on.
	compose build --progress=plain
fi

# ---------------------------------------------------------------------------
hr "3/5  Start databases + message broker (live, --wait)"
compose up -d --wait mysql-e2e postgres-e2e rabbitmq

hr "3b   Migrate + seed E2E databases"
./scripts/setup-e2e-db.sh

# ---------------------------------------------------------------------------
hr "4/5  Start services (detached) and watch health live"
compose up -d

# Poll health ourselves so a stuck service is VISIBLE (instead of a silent
# `--wait` block). On timeout, dump the logs of whatever never went healthy.
deadline=$(( SECONDS + HEALTH_WAIT ))
while :; do
	status="$(compose ps --format '{{.Service}}\t{{.State}}\t{{.Health}}')"
	printf '\033[2J\033[H'  # clear screen for a stable live view
	hr "service health  (t=$(( SECONDS ))s / ${HEALTH_WAIT}s)"
	echo -e "SERVICE\tSTATE\tHEALTH"
	echo -e "$status"

	# Anything still starting/unhealthy keeps us waiting.
	if ! echo "$status" | grep -qiE 'starting|unhealthy'; then
		# All containers either healthy or have no healthcheck (running).
		if ! echo "$status" | grep -qiE 'exited|dead|restarting'; then
			info "all services healthy"
			break
		fi
	fi

	if (( SECONDS > deadline )); then
		err "Timed out after ${HEALTH_WAIT}s waiting for services to become healthy."
		while IFS=$'\t' read -r svc state health; do
			[[ "$svc" == "SERVICE" ]] && continue
			if echo "$health $state" | grep -qiE 'starting|unhealthy|exited|dead|restarting'; then
				hr "LAST 80 LOG LINES: $svc  ($state / $health)"
				compose logs --tail=80 "$svc" || true
			fi
		done <<< "$status"
		err "Stack is unhealthy — see logs above. Leaving it up for inspection (compose ps / logs)."
		exit 1
	fi
	sleep 3
done

# ---------------------------------------------------------------------------
hr "5/5  Run E2E tests (timeout=${TIMEOUT}${RUN_REGEX:+, run=$RUN_REGEX})"
set +e
./scripts/run-e2e-tests.sh "$TIMEOUT" "$RUN_REGEX"
rc=$?
set -e

if (( rc != 0 )); then
	err "E2E tests failed (exit $rc). Stack left up for inspection:"
	err "  docker compose -f $COMPOSE_FILE logs <service>"
	err "  docker compose -f $COMPOSE_FILE ps"
	err "Tear down with: make e2e-down"
fi

exit "$rc"
