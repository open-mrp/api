#!/usr/bin/env bash
# Runs API E2E tests with gotestsum when available:
# - pkgname-and-test-fails: quiet progress for passing packages; prints full output for failures only.
set -euo pipefail

timeout="${1:?usage: $0 <go-test-timeout e.g. 300s or 600s> [optional-go-test-run-regex]}"
run_regex="${2:-}"
pkg="./tests/e2e/api/..."
args=(-tags e2e -count=1 -timeout "$timeout")

if [[ -n "${run_regex}" ]]; then
	args+=(-run "$run_regex")
fi

if command -v gotestsum >/dev/null 2>&1; then
	exec gotestsum -f pkgname-and-test-fails --format-icons=text -- \
		"${args[@]}" "$pkg"
fi

echo "gotestsum not found (run: make install-tools). Using plain go test; failures are harder to scan." >&2
exec go test "${args[@]}" "$pkg"
