#!/usr/bin/env bash
# Runs the Vitess smoke test (services/core-service/.../vitess_smoke_test.go) against a throwaway
# vtgate. Production is PlanetScale, so every statement is planned by vtgate first; the e2e stack
# and the unit tests run on plain MySQL, which accepts SQL vtgate rejects. This applies the schema
# migrations, seeds and data migrations through vtgate, then runs the queries through it.
set -euo pipefail

IMAGE="${VITESS_SMOKE_IMAGE:-vitess/vttestserver:mysql84}"
CONTAINER="openmrp-vitess-smoke"
PORT=33577

cd "$(dirname "$0")/.."

cleanup() { docker rm -f "$CONTAINER" >/dev/null 2>&1 || true; }
trap cleanup EXIT
cleanup

echo "Starting vtgate ($IMAGE)..."
docker run -d --name "$CONTAINER" --platform linux/amd64 -p "$PORT:$PORT" \
    -e PORT=$((PORT - 3)) -e KEYSPACES=openmrp -e NUM_SHARDS=1 \
    -e MYSQL_MAX_CONNECTIONS=70000 -e MYSQL_BIND_HOST=0.0.0.0 \
    "$IMAGE" >/dev/null

for _ in $(seq 1 90); do
    docker exec "$CONTAINER" mysql -h127.0.0.1 -P"$PORT" -e "SELECT 1" >/dev/null 2>&1 && break
    sleep 2
done

# vtgate rejects the empty statements the mysql client makes of semicolons inside comments, where
# MySQL ignores them, so comment lines are dropped before each file is sent.
apply() {
    local label="$1"
    local out
    if ! out=$({ grep -v '^[[:space:]]*--' || true; } | docker exec -i "$CONTAINER" mysql -h127.0.0.1 -P"$PORT" openmrp 2>&1); then
        echo "FAILED: $label" >&2
        echo "$out" >&2
        exit 1
    fi
}
goose_up() { sed -n '/^-- +goose Up$/,/^-- +goose Down$/{ /^-- +goose Up$/d; /^-- +goose Down$/d; p; }' "$1"; }

echo "Applying schema migrations through vtgate..."
for f in shared/db/migrations/*.sql; do goose_up "$f" | apply "$f"; done
echo "Seeding through vtgate..."
for f in shared/db/seed/*.sql; do apply "$f" <"$f"; done
echo "Applying data migrations through vtgate..."
for f in shared/db/data-migrations/*.sql; do goose_up "$f" | apply "$f"; done

echo "Running queries through vtgate..."
VITESS_SMOKE_DSN="root@tcp(127.0.0.1:$PORT)/openmrp?parseTime=true" \
    go test -tags vitess_smoke -count=1 -run TestVitessSmoke -v ./services/core-service/internal/infrastructure/repository/
