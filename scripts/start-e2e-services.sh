#!/usr/bin/env bash

# Start the e2e application services and block until every health-checked container
# is ready. We avoid `docker compose up --wait` because parallel image rebuilds
# recreate many interdependent containers at once; Compose's built-in waiter can
# race and fail with "No such container" while dependency health checks run.

set -euo pipefail

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; }

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

COMPOSE_FILE="docker-compose.e2e.yml"
MAX_UP_RETRIES=3
WAIT_TIMEOUT_SEC=180
POLL_INTERVAL_SEC=2

containers=(
	openmrp-platform-service-e2e
	openmrp-core-service-e2e
	openmrp-notification-service-e2e
	openmrp-auth-service-e2e
	openmrp-billing-service-e2e
	openmrp-agent-service-e2e
	openmrp-api-gateway-e2e
)

compose_up() {
	docker compose -f "$COMPOSE_FILE" up -d "$@"
}

info "Starting e2e application services..."

attempt=1
while true; do
	if compose_up; then
		break
	fi

	if [ "$attempt" -ge "$MAX_UP_RETRIES" ]; then
		error "Failed to start e2e services after $MAX_UP_RETRIES attempts."
		exit 1
	fi

	info "Compose up failed (attempt $attempt/$MAX_UP_RETRIES); retrying in ${POLL_INTERVAL_SEC}s..."
	sleep "$POLL_INTERVAL_SEC"
	attempt=$((attempt + 1))
done

deadline=$((SECONDS + WAIT_TIMEOUT_SEC))

info "Waiting for e2e services to become healthy..."

for container in "${containers[@]}"; do
	while true; do
		status="$(docker inspect --format='{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$container" 2>/dev/null || echo "missing")"

		if [ "$status" = "healthy" ]; then
			break
		fi

		if [ "$SECONDS" -ge "$deadline" ]; then
			error "Timed out waiting for $container (status: $status)."
			docker logs --tail 50 "$container" 2>&1 || true
			exit 1
		fi

		sleep "$POLL_INTERVAL_SEC"
	done
done

info "All e2e services are healthy."
