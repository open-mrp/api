#!/usr/bin/env bash
set -euo pipefail

base_ref="${1:-origin/main}"

pkgs=()
while IFS= read -r dir; do
	[ -d "$dir" ] && pkgs+=("$dir")
done < <(
	git diff --name-only "${base_ref}"...HEAD -- '*.go' \
		| grep -v -E '(^|/)sqlc/|(/|^)proto/|(/|^)tools/|^tests/' \
		| xargs -I{} dirname {} 2>/dev/null \
		| sort -u \
		| sed 's|^|./|'
)

if ((${#pkgs[@]} > 0)); then
	IFS=' '
	echo "${pkgs[*]}"
fi
