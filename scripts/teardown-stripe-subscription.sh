#!/usr/bin/env bash

# Teardown Stripe Subscription
# Deletes Stripe resources (cadence, customer) created by seed-stripe-subscription.sh
# and clears the corresponding fields in the local account_billing record.
#
# Usage: ./scripts/teardown-stripe-subscription.sh
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
ACCOUNT_BILLING_ID="acbl_01seedacmebilling0000"

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

# --- Read Stripe IDs from DB ---

STRIPE_IDS=$($MYSQL_CMD -sN -e "
    SELECT internal_stripe_customer_id, stripe_billing_cadence_id, stripe_pricing_plan_subscription_id
    FROM account_billing
    WHERE id = '${ACCOUNT_BILLING_ID}'
" 2>/dev/null || echo "")

CUSTOMER_ID=$(echo "$STRIPE_IDS" | awk '{print $1}')
CADENCE_ID=$(echo "$STRIPE_IDS" | awk '{print $2}')
SUBSCRIPTION_ID=$(echo "$STRIPE_IDS" | awk '{print $3}')

if [ -z "$CUSTOMER_ID" ] || [ "$CUSTOMER_ID" = "NULL" ]; then
    info "No Stripe customer found on account_billing record. Nothing to tear down."
    exit 0
fi

info "Tearing down Stripe resources for customer: $CUSTOMER_ID"

# --- Deactivate pricing plan subscription via billing intent ---

if [ -n "$SUBSCRIPTION_ID" ] && [ "$SUBSCRIPTION_ID" != "NULL" ] && [ -n "$CADENCE_ID" ] && [ "$CADENCE_ID" != "NULL" ]; then
    info "Deactivating pricing plan subscription: $SUBSCRIPTION_ID..."
    DEACTIVATE_RESP=$(stripe_v2 POST /v2/billing/intents \
        --json "{
            \"cadence\":\"${CADENCE_ID}\",
            \"currency\":\"usd\",
            \"actions\":[{
                \"type\":\"deactivate\",
                \"deactivate\":{
                    \"type\":\"pricing_plan_subscription_details\",
                    \"pricing_plan_subscription_details\":{
                        \"pricing_plan_subscription\":\"${SUBSCRIPTION_ID}\"
                    }
                }
            }]
        }")
    DEACTIVATE_INTENT_ID=$(echo "$DEACTIVATE_RESP" | jq -r '.id // empty')
    if [ -n "$DEACTIVATE_INTENT_ID" ]; then
        stripe_v2 POST "/v2/billing/intents/${DEACTIVATE_INTENT_ID}/reserve" >/dev/null 2>&1
        stripe_v2 POST "/v2/billing/intents/${DEACTIVATE_INTENT_ID}/commit" --json "{}" >/dev/null 2>&1
        info "Subscription deactivated."
    else
        warn "Could not create deactivate intent (subscription may already be canceled)."
    fi
fi

# --- Delete billing cadence ---

if [ -n "$CADENCE_ID" ] && [ "$CADENCE_ID" != "NULL" ]; then
    info "Deleting billing cadence: $CADENCE_ID..."
    stripe_v2 DELETE "/v2/billing/cadences/${CADENCE_ID}" >/dev/null 2>&1 || warn "Could not delete cadence (may already be deleted)."
    info "Cadence deleted."
fi

# --- Delete Stripe customer (cascades to payment methods, etc.) ---

info "Deleting Stripe customer: $CUSTOMER_ID..."
DELETE_RESP=$(stripe_v1 DELETE "/v1/customers/${CUSTOMER_ID}")
DELETED=$(echo "$DELETE_RESP" | jq -r '.deleted // false')
if [ "$DELETED" = "true" ]; then
    info "Customer deleted."
else
    warn "Could not delete customer (may already be deleted)."
fi

# --- Clear DB fields ---

info "Clearing account_billing Stripe fields..."
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
    WHERE id = '${ACCOUNT_BILLING_ID}';
"

info "Stripe teardown complete."
