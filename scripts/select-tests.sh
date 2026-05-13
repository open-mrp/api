#!/usr/bin/env bash
set -euo pipefail

base_ref="${1:-origin/main}"
module_path="github.com/augno/api"

# Dependency graph can shift broadly when module definitions change.
if git diff --name-only "${base_ref}"...HEAD -- go.mod go.sum | rg . >/dev/null; then
	echo "./..."
	exit 0
fi

changed_go_dirs="$(
	git diff --name-only "${base_ref}"...HEAD -- '*.go' \
		| rg -v '^tests/e2e/' \
		| xargs -I{} dirname "{}" 2>/dev/null \
		| sort -u || true
)"

if [[ -z "${changed_go_dirs}" ]]; then
	exit 0
fi

py_script="$(mktemp)"
cleanup() {
	rm -f "${py_script}"
}
trap cleanup EXIT

cat > "${py_script}" <<'PY'
import json
import os
import sys
from collections import defaultdict, deque

MODULE = os.environ["SELECT_TESTS_MODULE"]
changed_dirs = [
    line.strip()
    for line in os.environ.get("SELECT_TESTS_CHANGED_DIRS", "").splitlines()
    if line.strip()
]
changed_import_paths = set()
for d in changed_dirs:
    if d == ".":
        changed_import_paths.add(MODULE)
    else:
        changed_import_paths.add(f"{MODULE}/{d}")

decoder = json.JSONDecoder()
text = sys.stdin.buffer.read().decode("utf-8", errors="replace")

pkgs = {}
idx = 0
length = len(text)
while idx < length:
    while idx < length and text[idx].isspace():
        idx += 1
    if idx >= length:
        break
    obj, next_idx = decoder.raw_decode(text, idx)
    idx = next_idx
    ip = obj.get("ImportPath")
    if ip:
        pkgs[ip] = obj

reverse = defaultdict(set)
for ip, obj in pkgs.items():
    imports = set(obj.get("Imports", []))
    imports.update(obj.get("TestImports", []))
    imports.update(obj.get("XTestImports", []))
    for dep in imports:
        reverse[dep].add(ip)

queue = deque(ip for ip in changed_import_paths if ip in pkgs)
affected = set(queue)

while queue:
    current = queue.popleft()
    for importer in reverse.get(current, []):
        if importer in affected:
            continue
        affected.add(importer)
        queue.append(importer)

selected = []
for ip in sorted(affected):
    obj = pkgs.get(ip, {})
    if obj.get("TestGoFiles") or obj.get("XTestGoFiles"):
        if ip == MODULE:
            selected.append("./")
        elif ip.startswith(MODULE + "/"):
            selected.append("./" + ip[len(MODULE) + 1 :])

if selected:
    print(" ".join(selected))
PY

SELECT_TESTS_MODULE="${module_path}" \
	SELECT_TESTS_CHANGED_DIRS="${changed_go_dirs}" \
	go list -json ./... | python3 "${py_script}"
