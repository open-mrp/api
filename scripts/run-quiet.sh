#!/usr/bin/env bash
set -euo pipefail

label="${1:?usage: $0 <label> <command> [args...]}"
shift

log_file="$(mktemp)"
trap 'rm -f "$log_file"' EXIT

echo "[INFO] $label..."

if ! "$@" >"$log_file" 2>&1; then
	echo "[ERROR] $label failed." >&2
	sed 's/^/  /' "$log_file" >&2
	exit 1
fi
