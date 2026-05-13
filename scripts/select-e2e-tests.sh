#!/usr/bin/env bash
set -euo pipefail

base_ref="${1:-origin/main}"
head_ref="${2:-HEAD}"

changed_files="$(git diff --name-only "${base_ref}" "${head_ref}" || true)"
if [[ -z "${changed_files}" ]]; then
	exit 0
fi

py_script="$(mktemp)"
cleanup() {
	rm -f "${py_script}"
}
trap cleanup EXIT

cat > "${py_script}" <<'PY'
import os
import re
from pathlib import Path

changed = [line.strip() for line in os.environ["SELECT_E2E_CHANGED"].splitlines() if line.strip()]
repo_root = Path(os.environ["SELECT_E2E_REPO_ROOT"])

# Broad/shared changes can affect many endpoint behaviors.
full_run_prefixes = (
    "shared/",
    "infra/production/docker/Dockerfile",
    "docker-compose.e2e.yml",
    "tools/apidocs/",
)

prefix_mappings = [
    ("account-users", "TestAccountUsers_"),
    ("account_groups", "TestAccountGroups_"),
    ("addresses", "TestAddresses_"),
    ("api-keys", "TestAPIKeys_"),
    ("attributes", "TestAttributes_"),
    ("batches", "TestBatches_"),
    ("carriers", "TestCarriers_"),
    ("consumptions", "TestConsumptions_"),
    ("customers", "TestCustomers_"),
    ("item-categories", "TestItemCategories_"),
    ("items", "TestItems_"),
    ("locations", "TestLocations_"),
    ("machines", "TestMachines_"),
    ("materials", "TestMaterials_"),
    ("parts", "TestParts_"),
    ("payment-terms", "TestPaymentTerms_"),
    ("product-lines", "TestProductLines_"),
    ("products", "TestProducts_"),
    ("properties", "TestProperties_"),
    ("rates", "TestRates_"),
    ("roles", "TestRoles_"),
    ("sandboxes", "TestSandboxes_"),
    ("scanning-stations", "TestScanningStations_"),
    ("shipping-terms", "TestShippingTerms_"),
    ("unit-groups", "TestUnitGroups_"),
    ("units", "TestUnits_"),
]

exact_tests = set()
prefix_tests = set()
force_full = False

for path in changed:
    if any(path.startswith(p) for p in full_run_prefixes):
        force_full = True
        break

    if path.startswith("tests/e2e/api/") and path.endswith("_test.go"):
        full_path = repo_root / path
        if full_path.exists():
            content = full_path.read_text(encoding="utf-8", errors="replace")
            for name in re.findall(r"^func\s+(Test[0-9A-Za-z_]+)\s*\(", content, flags=re.MULTILINE):
                exact_tests.add(name)
        continue

    if path.startswith("services/"):
        matched = False
        for token, test_prefix in prefix_mappings:
            if token in path:
                prefix_tests.add(test_prefix)
                matched = True
        if not matched:
            force_full = True
            break

if force_full:
    print(".*")
    raise SystemExit(0)

parts = [f"^{re.escape(name)}$" for name in sorted(exact_tests)]
parts.extend(f"^{re.escape(prefix)}.*$" for prefix in sorted(prefix_tests))

if parts:
    print("(" + "|".join(parts) + ")")
PY

SELECT_E2E_CHANGED="${changed_files}" \
	SELECT_E2E_REPO_ROOT="$(pwd)" \
	python3 "${py_script}"
