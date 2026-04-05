#!/usr/bin/env bash
# Runs API E2E tests with gotestsum when available:
# - pkgname-and-test-fails: quiet progress for passing packages; prints full output for failures only.
set -euo pipefail

timeout="${1:?usage: $0 <go-test-timeout e.g. 300s or 600s>}"
pkg="./tests/e2e/api/..."

if command -v gotestsum >/dev/null 2>&1; then
	exec gotestsum -f pkgname-and-test-fails --format-icons=text -- \
		-tags e2e -count=1 -timeout "$timeout" "$pkg"
fi

echo "gotestsum not found (run: make install-tools). Using plain go test; failures are harder to scan." >&2
exec go test -tags e2e -count=1 -timeout "$timeout" "$pkg"
