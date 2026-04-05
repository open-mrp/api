import argparse
import os
import re
import sys
from typing import Dict, List


ENDPOINTS_ROOT_DEFAULT = "services/api-gateway/endpoints"
CORE_PREFIX = "/v1/core/"
ROUTE_RE = re.compile(r'Route:\s*"(/v1/core/[^"]*)"')


DOMAIN_BY_FOLDER: Dict[str, str] = {
    # Core cross-cutting platform endpoints.
    "sandboxes": "core",
    "analytics": "core",
    "address-validation": "core",
    "utils": "core",
    "request-logs": "core",
    "email-logs": "core",
    "sys-properties": "core",

    # Accounts / workspace admin.
    "accounts": "accounts",
    "account-groups": "accounts",
    "account-users": "accounts",
    "account-integrations": "accounts",
    "account-statuses": "accounts",
    "roles": "accounts",
    "permission-groups": "accounts",
    "users": "accounts",
    "child-accounts": "accounts",
    "territories": "accounts",
    "tenancy": "accounts",

    # Catalog / master data.
    "catalog": "catalog",
    "products": "catalog",
    "product-lines": "catalog",
    "product-types": "catalog",
    "items": "catalog",
    "item-categories": "catalog",
    "properties": "catalog",
    "unit-groups": "catalog",
    "units": "catalog",

    # Sales / customer-facing commercial.
    "customers": "sales",
    "sales-orders": "sales",
    "sales-order-statuses": "sales",
    "sales-targets": "sales",
    "checkout-sessions": "sales",
    "registration-flows": "sales",
    "priorities": "sales",
    "account-prices": "sales",
    "addresses": "sales",
    "order-discounts": "sales",
    "volume-discounts": "sales",
    "account-group-product-line-access": "sales",
    "customer-product-line-access": "sales",
    "address-validation": "core",  # explicit, though already covered above.

    # Operations / supply chain + fulfillment.
    "adjustment-types": "operations",
    "suppliers": "operations",
    "supplier-materials": "operations",
    "purchase-orders": "operations",
    "receiving-orders": "operations",
    "inventories": "operations",
    "inventory-change-logs": "operations",
    "storage-locations": "operations",
    "batches": "operations",
    "picks": "operations",
    "shipments": "operations",
    "shipping-cases": "operations",
    "carriers": "operations",
    "carrier-options": "operations",
    "deliveries": "operations",
    "shipping-terms": "operations",
    "edi": "operations",
    "edi-runs": "operations",
    "edi-dc-locations": "operations",
    "dc-locations": "operations",
    "materials": "operations",
    "parts": "operations",
    "machines": "operations",
    "departments": "operations",
    "production-runs": "operations",
    "production-steps": "operations",
    "production-flows": "operations",
    "consumptions": "operations",
    "quantities": "operations",
    "rates": "operations",
    "scanning-stations": "operations",

    # Finance / money movement.
    "invoices": "finance",
    "receivables": "finance",
    "transactions": "finance",
    "transaction-allocations": "finance",
    "transaction-methods": "finance",
    "transaction-types": "finance",
    "payment-terms": "finance",
    "settlements": "finance",
    # open-credits is currently served from transaction-allocations.
}


CORE_FOLDERS: List[str] = [
    "sandboxes",
    "analytics",
    "address-validation",
    "utils",
    "request-logs",
    "email-logs",
    "sys-properties",
]


def endpoint_folder_name(path: str, endpoints_root: str) -> str:
    rel = os.path.relpath(path, endpoints_root)
    # endpoints/<folder>/<file>.go
    parts = rel.split(os.sep)
    return parts[0] if parts else ""


def rewrite_route_literals(content: str, domain: str) -> str:
    def replace(match: re.Match) -> str:
        route_full = match.group(1)  # "/v1/core/..."
        if domain == "core":
            return f'Route: "{route_full}"'
        rest = route_full[len(CORE_PREFIX) :]  # after "/v1/core/"
        new_route = f"/v1/{domain}/{rest}"
        return f'Route: "{new_route}"'

    return ROUTE_RE.sub(replace, content)


def verify_no_unexpected_core_routes(endpoints_root: str) -> None:
    failures: List[str] = []
    for dirpath, _, filenames in os.walk(endpoints_root):
        for fn in filenames:
            if not fn.endswith(".go"):
                continue
            p = os.path.join(dirpath, fn)
            folder = endpoint_folder_name(p, endpoints_root)
            if not folder:
                continue
            with open(p, "r", encoding="utf-8") as f:
                content = f.read()

            # Only check endpoint Route literals.
            if re.search(r'Route:\s*"/v1/core/', content):
                if folder not in CORE_FOLDERS:
                    failures.append(p)

    if failures:
        raise SystemExit(
            "Verification failed: found /v1/core route literals outside allowed core folders:\n"
            + "\n".join(failures)
        )


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument(
        "--endpoints-root",
        default=ENDPOINTS_ROOT_DEFAULT,
        help="Path to services/api-gateway/endpoints",
    )
    args = ap.parse_args()

    endpoints_root = args.endpoints_root
    if not os.path.isdir(endpoints_root):
        raise SystemExit(f"endpoints-root not found or not a directory: {endpoints_root}")

    # First pass: identify any endpoint file using /v1/core in an unmapped folder.
    unmapped_folders: List[str] = []
    for dirpath, _, filenames in os.walk(endpoints_root):
        for fn in filenames:
            if not fn.endswith(".go"):
                continue
            p = os.path.join(dirpath, fn)
            folder = endpoint_folder_name(p, endpoints_root)
            if not folder:
                continue

            with open(p, "r", encoding="utf-8") as f:
                content = f.read()
            if re.search(r'Route:\s*"/v1/core/', content):
                if folder not in DOMAIN_BY_FOLDER:
                    unmapped_folders.append(folder)

    if unmapped_folders:
        uniq = sorted(set(unmapped_folders))
        raise SystemExit(
            "Codemod aborted: found /v1/core route literals in unmapped endpoint folders:\n"
            + "\n".join(uniq)
        )

    # Second pass: rewrite route literals.
    modified_files: List[str] = []
    for dirpath, _, filenames in os.walk(endpoints_root):
        for fn in filenames:
            if not fn.endswith(".go"):
                continue
            p = os.path.join(dirpath, fn)
            folder = endpoint_folder_name(p, endpoints_root)
            if not folder:
                continue

            with open(p, "r", encoding="utf-8") as f:
                content = f.read()

            if not re.search(r'Route:\s*"/v1/core/', content):
                continue

            domain = DOMAIN_BY_FOLDER[folder]
            new_content = rewrite_route_literals(content, domain)
            if new_content != content:
                with open(p, "w", encoding="utf-8") as f:
                    f.write(new_content)
                modified_files.append(p)

    verify_no_unexpected_core_routes(endpoints_root)

    print(f"Codemod complete. Modified {len(modified_files)} endpoint files.")


if __name__ == "__main__":
    main()

