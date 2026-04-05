#!/usr/bin/env bash

# Teardown All Stripe Resources
# Cancels all active subscriptions and deletes all customers in the Stripe account.
# Then clears corresponding fields in all account_billing records.
#
# Usage: ./scripts/teardown-all-stripe.sh
#
# Requires:
#   - STRIPE_SECRET_KEY (or set in .env)
#   - DB_URL (or set in .env)

set -euo pipefail

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Load .env if present
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

if [ -f ".env" ]; then
    while IFS= read -r line; do
        if [[ -n "$line" && ! "$line" =~ ^[[:space:]]*# ]]; then
            export "$line"
        fi
    done < .env
fi

# --- Config ---

STRIPE_SECRET_KEY="${STRIPE_SECRET_KEY:-}"
STRIPE_API_VERSION="2026-03-04.preview"

if [ -z "$STRIPE_SECRET_KEY" ]; then
    error "STRIPE_SECRET_KEY is not set. Export it or add it to .env."
    exit 1
fi

if [[ "$STRIPE_SECRET_KEY" != sk_test_* ]]; then
    error "STRIPE_SECRET_KEY must be a test mode key (sk_test_*). Refusing to run against a live key."
    exit 1
fi

# --- Validate DB_URL and parse connection ---

DB_URL="${DB_URL:-}"
if [ -z "$DB_URL" ]; then
    error "DB_URL is not set. Export it or add it to .env."
    exit 1
fi

if ! echo "$DB_URL" | grep -qiE '^mysql://'; then
    error "DB_URL does not start with mysql://."
    exit 1
fi

if ! echo "$DB_URL" | grep -qiE '@(localhost|127\.0\.0\.1|mysql|seed-db):'; then
    error "DB_URL must connect to localhost, 127.0.0.1, or a Docker container."
    exit 1
fi

MYSQL_CONN="${DB_URL#mysql://}"
MYSQL_USER="$(echo "$MYSQL_CONN" | sed -E 's/(.+):(.+)@(.+):([0-9]+)\/(.+)/\1/')"
MYSQL_PASS="$(echo "$MYSQL_CONN" | sed -E 's/(.+):(.+)@(.+):([0-9]+)\/(.+)/\2/')"
MYSQL_HOST="$(echo "$MYSQL_CONN" | sed -E 's/(.+):(.+)@(.+):([0-9]+)\/(.+)/\3/')"
MYSQL_PORT="$(echo "$MYSQL_CONN" | sed -E 's/(.+):(.+)@(.+):([0-9]+)\/(.+)/\4/')"
MYSQL_DB="$(echo "$MYSQL_CONN" | sed -E 's/(.+):(.+)@(.+):([0-9]+)\/(.+)/\5/')"

MYSQL_CMD="mysql -u${MYSQL_USER} -p${MYSQL_PASS} -h${MYSQL_HOST} -P${MYSQL_PORT} --protocol=tcp ${MYSQL_DB}"

# --- Helper: Stripe API call ---

stripe_v1() {
    local method="$1"
    local endpoint="$2"
    shift 2

    curl -s -X "$method" "https://api.stripe.com${endpoint}" \
        -H "Authorization: Bearer ${STRIPE_SECRET_KEY}" \
        "$@"
}

stripe_v2() {
    local method="$1"
    local endpoint="$2"
    shift 2

    curl -s -X "$method" "https://api.stripe.com${endpoint}" \
        -H "Authorization: Bearer ${STRIPE_SECRET_KEY}" \
        -H "Stripe-Version: ${STRIPE_API_VERSION}" \
        -H "Content-Type: application/json" \
        "$@"
}

# =============================================================================
# Phase 1: Cancel all active subscriptions via Stripe API
# =============================================================================

info "Fetching all active subscriptions from Stripe..."

SUB_COUNT=0
HAS_MORE="true"
STARTING_AFTER=""

while [ "$HAS_MORE" = "true" ]; do
    PARAMS="status=active&limit=100"
    if [ -n "$STARTING_AFTER" ]; then
        PARAMS="${PARAMS}&starting_after=${STARTING_AFTER}"
    fi

    SUBS_RESP=$(stripe_v1 GET "/v1/subscriptions?${PARAMS}")
    HAS_MORE=$(echo "$SUBS_RESP" | jq -r '.has_more')
    SUB_IDS=$(echo "$SUBS_RESP" | jq -r '.data[].id // empty')

    for SUB_ID in $SUB_IDS; do
        info "Canceling subscription: $SUB_ID"
        stripe_v1 DELETE "/v1/subscriptions/${SUB_ID}" >/dev/null 2>&1 || warn "Could not cancel subscription $SUB_ID"
        SUB_COUNT=$((SUB_COUNT + 1))
        STARTING_AFTER="$SUB_ID"
    done

    if [ -z "$SUB_IDS" ]; then
        break
    fi
done

info "Canceled $SUB_COUNT active subscription(s)."

# =============================================================================
# Phase 2: Delete all customers via Stripe API
# =============================================================================

info "Fetching all customers from Stripe..."

CUST_COUNT=0
HAS_MORE="true"
STARTING_AFTER=""

while [ "$HAS_MORE" = "true" ]; do
    PARAMS="limit=100"
    if [ -n "$STARTING_AFTER" ]; then
        PARAMS="${PARAMS}&starting_after=${STARTING_AFTER}"
    fi

    CUSTS_RESP=$(stripe_v1 GET "/v1/customers?${PARAMS}")
    HAS_MORE=$(echo "$CUSTS_RESP" | jq -r '.has_more')
    CUST_IDS=$(echo "$CUSTS_RESP" | jq -r '.data[].id // empty')

    for CUST_ID in $CUST_IDS; do
        info "Deleting customer: $CUST_ID"
        stripe_v1 DELETE "/v1/customers/${CUST_ID}" >/dev/null 2>&1 || warn "Could not delete customer $CUST_ID"
        CUST_COUNT=$((CUST_COUNT + 1))
        STARTING_AFTER="$CUST_ID"
    done

    if [ -z "$CUST_IDS" ]; then
        break
    fi
done

info "Deleted $CUST_COUNT customer(s)."

# =============================================================================
# Phase 3: Clear all account_billing Stripe fields in the database
# =============================================================================

info "Clearing all account_billing Stripe fields..."
$MYSQL_CMD -e "
    UPDATE account_billing SET
        internal_stripe_customer_id = NULL,
        internal_stripe_subscription_id = NULL,
        stripe_billing_profile_id = NULL,
        stripe_billing_cadence_id = NULL,
        stripe_pricing_plan_subscription_id = NULL,
        servicing_status = NULL,
        collection_status = NULL,
        updated_at = NOW()
    WHERE internal_stripe_customer_id IS NOT NULL;
"

info "All Stripe resources torn down."
