#!/usr/bin/env bash

# Force-remove e2e containers by the names pinned in docker-compose.e2e.yml.
# Compose down only removes the current directory's project, so a stack started
# from another worktree keeps those names and the next `up` fails with a conflict.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_FILE="$REPO_ROOT/docker-compose.e2e.yml"

names="$(awk '/^[[:space:]]*container_name:/ {print $2}' "$COMPOSE_FILE")"

to_remove=()
for name in $names; do
	if docker inspect "$name" >/dev/null 2>&1; then
		to_remove+=("$name")
	fi
done

if ((${#to_remove[@]})); then
	docker rm -f "${to_remove[@]}"
fi
