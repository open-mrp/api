#!/usr/bin/env bash

# Seed Stripe Subscription
# Creates a Stripe test subscription for the seeded account using the v2 billing API,
# then updates the local account_billing record with the resulting Stripe IDs.
#
# Usage: ./scripts/seed-stripe-subscription.sh
#
# Requires:
#   - STRIPE_SECRET_KEY (or set in .env)
#   - DB_URL (or set in .env)
#
# This script is idempotent — it checks for an existing Stripe customer ID
# on the account_billing record before proceeding.

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

ENV_LOADED="no"
if [ -f ".env" ]; then
    set -a
    # shellcheck source=/dev/null
    . ./.env
    set +a
    ENV_LOADED="yes"
fi

# --- Require tools ---
for cmd in curl jq mysql; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        error "Required command not found: $cmd. Install it and try again."
        exit 1
    fi
done

# --- Config ---

STRIPE_SECRET_KEY="${STRIPE_SECRET_KEY:-}"
STRIPE_API_VERSION="2026-03-04.preview"
ACCOUNT_BILLING_ID="acbl_01seedacmebilling0000"
ACCOUNT_ID="ac_01k0a5smf9ekb8rqg12555zjqa"

if [ -z "$STRIPE_SECRET_KEY" ]; then
    error "STRIPE_SECRET_KEY is not set."
    if [ "$ENV_LOADED" = "yes" ]; then
        error "  .env was loaded from: $REPO_ROOT/.env but STRIPE_SECRET_KEY is empty or missing in that file."
    else
        error "  No .env found at $REPO_ROOT/.env. Export STRIPE_SECRET_KEY or add it to .env."
    fi
    exit 1
fi

if [[ "$STRIPE_SECRET_KEY" != sk_test_* ]]; then
    error "STRIPE_SECRET_KEY must be a test mode key (sk_test_*). Refusing to run against a live key."
    exit 1
fi

# --- Validate DB_URL and parse connection ---

DB_URL="${DB_URL:-}"
if [ -z "$DB_URL" ]; then
    error "DB_URL is not set."
    if [ "$ENV_LOADED" = "yes" ]; then
        error "  .env was loaded from: $REPO_ROOT/.env but DB_URL is empty or missing in that file."
    else
        error "  No .env found at $REPO_ROOT/.env. Export DB_URL or add it to .env."
    fi
    exit 1
fi

if ! echo "$DB_URL" | grep -qiE '^mysql://'; then
    error "DB_URL must start with mysql:// (got: ${DB_URL:0:20}...)."
    exit 1
fi

if ! echo "$DB_URL" | grep -qiE '@(localhost|127\.0\.0\.1|mysql|seed-db):'; then
    error "DB_URL must connect to localhost, 127.0.0.1, mysql, or seed-db (safety check)."
    exit 1
fi

MYSQL_CONN="${DB_URL#mysql://}"
MYSQL_USER="$(echo "$MYSQL_CONN" | sed -E 's/(.+):(.+)@(.+):([0-9]+)\/(.+)/\1/')"
MYSQL_PASS="$(echo "$MYSQL_CONN" | sed -E 's/(.+):(.+)@(.+):([0-9]+)\/(.+)/\2/')"
MYSQL_HOST="$(echo "$MYSQL_CONN" | sed -E 's/(.+):(.+)@(.+):([0-9]+)\/(.+)/\3/')"
MYSQL_PORT="$(echo "$MYSQL_CONN" | sed -E 's/(.+):(.+)@(.+):([0-9]+)\/(.+)/\4/')"
MYSQL_DB="$(echo "$MYSQL_CONN" | sed -E 's/(.+):(.+)@(.+):([0-9]+)\/(.+)/\5/')"

if [ -z "$MYSQL_HOST" ] || [ "$MYSQL_HOST" = "$MYSQL_CONN" ]; then
    error "DB_URL could not be parsed. Expected format: mysql://USER:PASSWORD@HOST:PORT/DATABASE"
    error "  Example: mysql://root:secret@localhost:3306/openmrp"
    exit 1
fi

MYSQL_CMD="mysql -u${MYSQL_USER} -p${MYSQL_PASS} -h${MYSQL_HOST} -P${MYSQL_PORT} --protocol=tcp ${MYSQL_DB}"

# --- Check idempotency ---

MYSQL_ERR=$(mktemp)
trap 'rm -f "$MYSQL_ERR"' EXIT
EXISTING_CUSTOMER=$($MYSQL_CMD -sN -e "SELECT internal_stripe_customer_id FROM account_billing WHERE id = '${ACCOUNT_BILLING_ID}'" 2>"$MYSQL_ERR") || true
{ grep -v '\[Warning\] Using a password on the command line interface' "$MYSQL_ERR" || true; } > "${MYSQL_ERR}.filtered" && mv "${MYSQL_ERR}.filtered" "$MYSQL_ERR"
if [ -s "$MYSQL_ERR" ]; then
    error "Database query failed (idempotency check). Is MySQL running at ${MYSQL_HOST}:${MYSQL_PORT}? Run: make local-db"
    cat "$MYSQL_ERR"
    exit 1
fi

if [ -n "$EXISTING_CUSTOMER" ] && [ "$EXISTING_CUSTOMER" != "NULL" ]; then
    warn "Account billing record already has Stripe customer: $EXISTING_CUSTOMER"
    warn "Skipping Stripe subscription creation. To re-run, clear the account_billing Stripe fields first."
    exit 0
fi

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

check_error() {
    local response="$1"
    local step="$2"
    if echo "$response" | jq -e '.error' >/dev/null 2>&1; then
        error "$step failed. Stripe API response:"
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
        exit 1
    fi
    if [ -z "$response" ] || ! echo "$response" | jq -e '.' >/dev/null 2>&1; then
        error "$step failed: empty or invalid JSON response (check network and Stripe key)."
        echo "$response"
        exit 1
    fi
}

# --- Fetch pricing plan ID from the account plan ---

PRICING_PLAN_ID=$($MYSQL_CMD -sN -e "
    SELECT ap.stripe_pricing_plan_id
    FROM account_billing ab
    JOIN account_plan ap ON ap.type_id = ab.account_plan_id
    WHERE ab.id = '${ACCOUNT_BILLING_ID}'
" 2>"$MYSQL_ERR") || true
{ grep -v '\[Warning\] Using a password on the command line interface' "$MYSQL_ERR" || true; } > "${MYSQL_ERR}.filtered" && mv "${MYSQL_ERR}.filtered" "$MYSQL_ERR"
if [ -s "$MYSQL_ERR" ]; then
    error "Database query failed (fetch pricing plan). Check that account_billing and account_plan exist. Run: make local-db and seed data first."
    cat "$MYSQL_ERR"
    exit 1
fi

if [ -z "$PRICING_PLAN_ID" ] || [ "$PRICING_PLAN_ID" = "NULL" ]; then
    error "No stripe_pricing_plan_id for account billing ${ACCOUNT_BILLING_ID}. The seeded account_plan may not have stripe_pricing_plan_id set."
    exit 1
fi

info "Using pricing plan: $PRICING_PLAN_ID"

# --- Step 1: Create Stripe customer ---

info "Creating Stripe customer..."
CUSTOMER_RESP=$(stripe_v1 POST /v1/customers \
    -d "name=Acme Inc." \
    -d "email=dev@augno.com" \
    -d "metadata[account_id]=${ACCOUNT_ID}" \
    -d "metadata[environment]=development")
check_error "$CUSTOMER_RESP" "Create customer"
CUSTOMER_ID=$(echo "$CUSTOMER_RESP" | jq -r '.id')
info "Created customer: $CUSTOMER_ID"

# --- Step 1b: Attach test payment method ---

info "Attaching test payment method (pm_card_visa)..."
PM_RESP=$(stripe_v1 POST /v1/payment_methods/pm_card_visa/attach \
    -d "customer=${CUSTOMER_ID}")
check_error "$PM_RESP" "Attach payment method"
PAYMENT_METHOD_ID=$(echo "$PM_RESP" | jq -r '.id')
info "Attached payment method: $PAYMENT_METHOD_ID"

info "Setting default payment method..."
DEFAULT_PM_RESP=$(stripe_v1 POST "/v1/customers/${CUSTOMER_ID}" \
    -d "invoice_settings[default_payment_method]=${PAYMENT_METHOD_ID}")
check_error "$DEFAULT_PM_RESP" "Set default payment method"
info "Default payment method set."

# --- Step 2: Create billing profile ---

info "Creating billing profile..."
PROFILE_RESP=$(stripe_v2 POST /v2/billing/profiles \
    --json "{\"customer\":\"${CUSTOMER_ID}\"}")
check_error "$PROFILE_RESP" "Create billing profile"
PROFILE_ID=$(echo "$PROFILE_RESP" | jq -r '.id')
info "Created billing profile: $PROFILE_ID"

# --- Step 3: Create billing cadence (charge_automatically) ---

info "Creating billing cadence..."
CADENCE_RESP=$(stripe_v2 POST /v2/billing/cadences \
    --json "{
        \"payer\":{\"billing_profile\":\"${PROFILE_ID}\"},
        \"billing_cycle\":{\"type\":\"month\",\"interval_count\":1}
    }")
check_error "$CADENCE_RESP" "Create billing cadence"
CADENCE_ID=$(echo "$CADENCE_RESP" | jq -r '.id')
info "Created billing cadence: $CADENCE_ID"

# --- Step 4: Fetch pricing plan live version ---

info "Fetching pricing plan live version..."
PLAN_RESP=$(stripe_v2 GET "/v2/billing/pricing_plans/${PRICING_PLAN_ID}")
check_error "$PLAN_RESP" "Fetch pricing plan"
LIVE_VERSION=$(echo "$PLAN_RESP" | jq -r '.live_version')
if [ -z "$LIVE_VERSION" ] || [ "$LIVE_VERSION" = "null" ]; then
    error "Pricing plan ${PRICING_PLAN_ID} has no live_version set."
    exit 1
fi
info "Live version: $LIVE_VERSION"

# --- Step 5: Create billing intent (subscribe) ---

info "Creating billing intent..."
INTENT_RESP=$(stripe_v2 POST /v2/billing/intents \
    --json "{
        \"cadence\":\"${CADENCE_ID}\",
        \"currency\":\"usd\",
        \"actions\":[{
            \"type\":\"subscribe\",
            \"subscribe\":{
                \"type\":\"pricing_plan_subscription_details\",
                \"pricing_plan_subscription_details\":{
                    \"pricing_plan\":\"${PRICING_PLAN_ID}\",
                    \"pricing_plan_version\":\"${LIVE_VERSION}\",
                    \"component_configurations\":[]
                }
            }
        }]
    }")
check_error "$INTENT_RESP" "Create billing intent"
INTENT_ID=$(echo "$INTENT_RESP" | jq -r '.id')
info "Created billing intent: $INTENT_ID"

# --- Step 6: Reserve billing intent ---

info "Reserving billing intent..."
RESERVE_RESP=$(stripe_v2 POST "/v2/billing/intents/${INTENT_ID}/reserve")
check_error "$RESERVE_RESP" "Reserve billing intent"
info "Billing intent reserved."

# --- Step 7: Create and confirm PaymentIntent ---

# With charge_automatically (default), Stripe requires a confirmed PaymentIntent
# whose amount matches the billing intent total. Fetch the reserved intent to get
# the exact amount, then create and confirm a PaymentIntent for that amount.

info "Fetching reserved billing intent for total amount..."
RESERVED_INTENT=$(stripe_v2 GET "/v2/billing/intents/${INTENT_ID}")
check_error "$RESERVED_INTENT" "Fetch reserved billing intent"

# Extract the total amount from amount_details.total
TOTAL_AMOUNT=$(echo "$RESERVED_INTENT" | jq -r '.amount_details.total // empty')

if [ -z "$TOTAL_AMOUNT" ] || [ "$TOTAL_AMOUNT" = "null" ]; then
    warn "Could not extract total amount from billing intent. Response:"
    echo "$RESERVED_INTENT" | jq '.' 2>/dev/null || echo "$RESERVED_INTENT"
    error "Cannot determine PaymentIntent amount. Aborting."
    exit 1
fi

info "Billing intent total amount: $TOTAL_AMOUNT"

info "Creating and confirming PaymentIntent..."
PI_RESP=$(stripe_v1 POST /v1/payment_intents \
    -d "amount=${TOTAL_AMOUNT}" \
    -d "currency=usd" \
    -d "customer=${CUSTOMER_ID}" \
    -d "payment_method=${PAYMENT_METHOD_ID}" \
    -d "confirm=true" \
    -d "return_url=https://example.com/return")
check_error "$PI_RESP" "Create PaymentIntent"
PAYMENT_INTENT_ID=$(echo "$PI_RESP" | jq -r '.id')
PI_STATUS=$(echo "$PI_RESP" | jq -r '.status')
info "PaymentIntent created: $PAYMENT_INTENT_ID (status: $PI_STATUS)"

# --- Step 8: Commit billing intent ---

info "Committing billing intent..."
COMMIT_RESP=$(stripe_v2 POST "/v2/billing/intents/${INTENT_ID}/commit" \
    --json "{\"payment_intent\":\"${PAYMENT_INTENT_ID}\"}")
check_error "$COMMIT_RESP" "Commit billing intent"
info "Billing intent committed."

# --- Step 9: Find the pricing plan subscription ---
# v2 billing: commit creates the pricing plan subscription; ID may be in commit response,
# in the committed billing intent (GET), or via list by cadence. See Stripe's pricing-plan
# subscription docs: https://docs.stripe.com/billing/subscriptions/usage-based/pricing-plans
# Parse path matches billing-service: actions[].subscribe.pricing_plan_subscription_details.pricing_plan_subscription

SUBSCRIPTION_ID=""
SUBSCRIPTION_ID=$(echo "$COMMIT_RESP" | jq -r '
  [.actions[]? | .subscribe?.pricing_plan_subscription_details?.pricing_plan_subscription // empty] | first // empty
' 2>/dev/null || true)

if [ -z "$SUBSCRIPTION_ID" ]; then
    INTENT_AFTER=$(stripe_v2 GET "/v2/billing/intents/${INTENT_ID}" 2>/dev/null || true)
    SUBSCRIPTION_ID=$(echo "$INTENT_AFTER" | jq -r '
      [.actions[]? | .subscribe?.pricing_plan_subscription_details?.pricing_plan_subscription // empty] | first // empty
    ' 2>/dev/null || true)
fi

MAX_SUB_RETRIES=5
SUB_RETRY_DELAY=2
if [ -z "$SUBSCRIPTION_ID" ]; then
    for i in $(seq 1 "$MAX_SUB_RETRIES"); do
        SUB_LIST=$(stripe_v2 GET "/v2/billing/pricing_plan_subscriptions?billing_cadence=${CADENCE_ID}" 2>/dev/null || true)
        if [ -n "$SUB_LIST" ] && echo "$SUB_LIST" | jq -e '.data' >/dev/null 2>&1; then
            SUBSCRIPTION_ID=$(echo "$SUB_LIST" | jq -r '.data[0].id // empty' 2>/dev/null || true)
        fi
        if [ -z "$SUBSCRIPTION_ID" ]; then
            SUB_LIST=$(stripe_v2 GET "/v2/billing/cadences/${CADENCE_ID}/pricing_plan_subscriptions" 2>/dev/null || true)
            if [ -n "$SUB_LIST" ] && echo "$SUB_LIST" | jq -e '.data' >/dev/null 2>&1; then
                SUBSCRIPTION_ID=$(echo "$SUB_LIST" | jq -r '.data[0].id // empty' 2>/dev/null || true)
            fi
        fi
        [ -n "$SUBSCRIPTION_ID" ] && break
        [ "$i" -lt "$MAX_SUB_RETRIES" ] && { info "Subscription not yet listed, retrying in ${SUB_RETRY_DELAY}s (attempt $i/$MAX_SUB_RETRIES)..."; sleep "$SUB_RETRY_DELAY"; }
    done
fi

if [ -z "$SUBSCRIPTION_ID" ]; then
    error "Could not find subscription ID after commit. Cannot update DB without it."
    error "See https://docs.stripe.com/billing/subscriptions/usage-based/pricing-plans for the v2 billing intent and pricing plan subscription flow."
    exit 1
fi

# --- Step 10: Update local DB ---

info "Updating account_billing record..."

$MYSQL_CMD -e "
    UPDATE account_billing SET
        internal_stripe_customer_id = '${CUSTOMER_ID}',
        stripe_billing_profile_id = '${PROFILE_ID}',
        stripe_billing_cadence_id = '${CADENCE_ID}',
        stripe_pricing_plan_subscription_id = '${SUBSCRIPTION_ID}',
        servicing_status = 'active',
        collection_status = 'current',
        updated_at = NOW()
    WHERE id = '${ACCOUNT_BILLING_ID}';
"

info "Done! Stripe subscription seeded successfully."
info ""
info "Summary:"
info "  Customer:     $CUSTOMER_ID"
info "  Profile:      $PROFILE_ID"
info "  Cadence:      $CADENCE_ID"
info "  Intent:       $INTENT_ID"
if [ -n "$SUBSCRIPTION_ID" ]; then info "  Subscription: $SUBSCRIPTION_ID"; fi
