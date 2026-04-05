#!/usr/bin/env bash

# Provision Founder Plan
# Creates the Founder pricing plan, a billing record, and a Stripe subscription
# for a given account. Uses the Stripe v2 billing API.
#
# Usage:
#   1. Fill in the config constants below.
#   2. Ensure STRIPE_SECRET_KEY and DB_URL are set (or in .env).
#   3. Run: ./scripts/provision-founder-plan.sh
#
# Clear the config constants before committing.

set -euo pipefail

# ============================================================
# CONFIG — fill these in before running, clear before committing
# ============================================================

ACCOUNT_ID=""              # ac_...
STRIPE_PRICING_PLAN_ID=""  # bpp_...
STRIPE_CUSTOMER_ID=""      # cus_...
PAYMENT_METHOD_ID=""       # pm_...

# ============================================================

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
for cmd in curl jq mysql go; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        error "Required command not found: $cmd. Install it and try again."
        exit 1
    fi
done

# --- Validate config constants ---

if [ -z "$ACCOUNT_ID" ]; then
    error "ACCOUNT_ID is not set. Edit this script and fill in the config section."
    exit 1
fi
if [ -z "$STRIPE_PRICING_PLAN_ID" ]; then
    error "STRIPE_PRICING_PLAN_ID is not set. Edit this script and fill in the config section."
    exit 1
fi
if [ -z "$STRIPE_CUSTOMER_ID" ]; then
    error "STRIPE_CUSTOMER_ID is not set. Edit this script and fill in the config section."
    exit 1
fi
if [ -z "$PAYMENT_METHOD_ID" ]; then
    error "PAYMENT_METHOD_ID is not set. Edit this script and fill in the config section."
    exit 1
fi

# --- Validate STRIPE_SECRET_KEY ---

STRIPE_SECRET_KEY="${STRIPE_SECRET_KEY:-}"
STRIPE_API_VERSION="2026-03-04.preview"

if [ -z "$STRIPE_SECRET_KEY" ]; then
    error "STRIPE_SECRET_KEY is not set."
    if [ "$ENV_LOADED" = "yes" ]; then
        error "  .env was loaded from: $REPO_ROOT/.env but STRIPE_SECRET_KEY is empty or missing."
    else
        error "  No .env found at $REPO_ROOT/.env. Export STRIPE_SECRET_KEY or add it to .env."
    fi
    exit 1
fi

if [[ "$STRIPE_SECRET_KEY" == sk_live_* ]]; then
    warn "STRIPE_SECRET_KEY is a LIVE key. You are operating against production Stripe."
    read -r -p "Type 'yes' to continue: " CONFIRM
    if [ "$CONFIRM" != "yes" ]; then
        error "Aborted."
        exit 1
    fi
fi

# --- Validate DB_URL and parse connection ---
# Supports: mysql://USER:PASSWORD@HOST[:PORT]/DATABASE[?params]
# Port is optional (defaults to 3306). Works with PlanetScale URLs.

DB_URL="${DB_URL:-}"
if [ -z "$DB_URL" ]; then
    error "DB_URL is not set."
    if [ "$ENV_LOADED" = "yes" ]; then
        error "  .env was loaded from: $REPO_ROOT/.env but DB_URL is empty or missing."
    else
        error "  No .env found at $REPO_ROOT/.env. Export DB_URL or add it to .env."
    fi
    exit 1
fi

if ! echo "$DB_URL" | grep -qiE '^mysql://'; then
    error "DB_URL must start with mysql:// (got: ${DB_URL:0:20}...)."
    exit 1
fi

warn "You are about to run against: $(echo "$DB_URL" | sed -E 's|mysql://[^@]+@|mysql://****@|')"
read -r -p "Type 'yes' to continue: " CONFIRM
if [ "$CONFIRM" != "yes" ]; then
    error "Aborted."
    exit 1
fi

MYSQL_CONN="${DB_URL#mysql://}"
MYSQL_CONN_BASE="${MYSQL_CONN%%\?*}"

# Extract user (everything before first :)
MYSQL_USER="$(echo "$MYSQL_CONN_BASE" | sed -E 's/^([^:]+):.*/\1/')"
# Extract password (between first : and last @)
MYSQL_PASS="$(echo "$MYSQL_CONN_BASE" | sed -E 's/^[^:]+:(.+)@.*/\1/')"
# Extract host+port/db (after last @)
MYSQL_HOSTPART="$(echo "$MYSQL_CONN_BASE" | sed -E 's/^.*@//')"
# Extract database (after first /)
MYSQL_DB="$(echo "$MYSQL_HOSTPART" | sed -E 's|^[^/]+/||')"
# Extract host:port (before first /)
MYSQL_HOSTPORT="$(echo "$MYSQL_HOSTPART" | sed -E 's|/.*||')"

# Split host and port (port is optional, default 3306)
if echo "$MYSQL_HOSTPORT" | grep -qE ':[0-9]+$'; then
    MYSQL_HOST="$(echo "$MYSQL_HOSTPORT" | sed -E 's/:[0-9]+$//')"
    MYSQL_PORT="$(echo "$MYSQL_HOSTPORT" | sed -E 's/.*://')"
else
    MYSQL_HOST="$MYSQL_HOSTPORT"
    MYSQL_PORT="3306"
fi

if [ -z "$MYSQL_HOST" ] || [ -z "$MYSQL_DB" ] || [ -z "$MYSQL_USER" ]; then
    error "DB_URL could not be parsed. Expected: mysql://USER:PASSWORD@HOST[:PORT]/DATABASE"
    exit 1
fi

# SSL: PlanetScale and other cloud providers require SSL
MYSQL_EXTRA_ARGS=""
MYSQL_PARAMS="${MYSQL_CONN#"$MYSQL_CONN_BASE"}"
if echo "$MYSQL_PARAMS" | grep -qiE 'tls|ssl'; then
    MYSQL_EXTRA_ARGS="--ssl-mode=REQUIRED"
fi
# PlanetScale hosts always need SSL
if echo "$MYSQL_HOST" | grep -qiE 'psdb\.cloud|planetscale'; then
    MYSQL_EXTRA_ARGS="--ssl-mode=REQUIRED"
fi

MYSQL_CMD="mysql -u${MYSQL_USER} -p${MYSQL_PASS} -h${MYSQL_HOST} -P${MYSQL_PORT} --protocol=tcp ${MYSQL_EXTRA_ARGS} ${MYSQL_DB}"

# --- Test DB connection ---

info "Testing database connection..."
if ! $MYSQL_CMD -e "SELECT 1;" &> /dev/null; then
    error "Failed to connect to database. Check DB_URL."
    exit 1
fi
info "Connection successful."

# --- Check account exists and has no billing ---

MYSQL_ERR=$(mktemp)
trap 'rm -f "$MYSQL_ERR"' EXIT

EXISTING_BILLING=$($MYSQL_CMD -sN -e "SELECT account_billing_id FROM account WHERE id = '${ACCOUNT_ID}'" 2>"$MYSQL_ERR") || true
{ grep -v '\[Warning\] Using a password on the command line interface' "$MYSQL_ERR" || true; } > "${MYSQL_ERR}.filtered" && mv "${MYSQL_ERR}.filtered" "$MYSQL_ERR"
if [ -s "$MYSQL_ERR" ]; then
    error "Database query failed (account check)."
    cat "$MYSQL_ERR"
    exit 1
fi

if [ -z "$EXISTING_BILLING" ] || [ "$EXISTING_BILLING" = "NULL" ]; then
    info "Account ${ACCOUNT_ID} has no billing record. Proceeding."
else
    error "Account ${ACCOUNT_ID} already has billing record: ${EXISTING_BILLING}"
    error "To re-provision, clear the account's account_billing_id first."
    exit 1
fi

# --- Generate IDs ---

info "Generating type IDs..."
PLAN_TYPE_ID=$(go run ./cmd/genid pl)
BILLING_ID=$(go run ./cmd/genid acbl)
info "Plan type ID:  $PLAN_TYPE_ID"
info "Billing ID:    $BILLING_ID"

# ============================================================
# PHASE 1: DB Setup
# ============================================================

info "Creating Founder plan..."

$MYSQL_CMD -e "
INSERT IGNORE INTO account_plan (
    type_id, name, plan_type_code, version,
    price_per_seat, price_per_month, seat_minimum,
    display_features, display_order, is_highlighted,
    button_text, includes_previous_plan,
    stripe_pricing_plan_id, effective_at,
    is_publicly_visible, created_at, updated_at
) VALUES (
    '${PLAN_TYPE_ID}',
    'Founder',
    'enterprise',
    1,
    0,
    1.00,
    NULL,
    '[\"Full API Access\",\"Order Management\",\"Purchasing Module\",\"Production Module\",\"Order Fulfillment\",\"Shipping Module\",\"BOM Management\",\"Inventory Management\",\"Request Tracking\",\"Priority Support\",\"Customer Order Portal\",\"Commission Tracking\",\"Implementation Support\",\"Hands-on Migration Support\",\"In-Person Training\",\"Custom Development\",\"EDI Support\",\"ITAR Compliance\",\"SSO/SAML\",\"ISO 13485 Compliance\",\"ISO 9001 Compliance\",\"Private Cloud Integration\"]',
    99,
    0,
    NULL,
    'Pro',
    '${STRIPE_PRICING_PLAN_ID}',
    NOW(),
    0,
    NOW(),
    NOW()
);
"

info "Creating plan limits..."

$MYSQL_CMD -e "
INSERT IGNORE INTO account_plan_limit (account_plan_id, \`key\`, value, created_at, updated_at) VALUES
    ('${PLAN_TYPE_ID}', 'seats_maximum', NULL, NOW(), NOW()),
    ('${PLAN_TYPE_ID}', 'invoices_maximum', NULL, NOW(), NOW()),
    ('${PLAN_TYPE_ID}', 'sandboxes_maximum', 10, NOW(), NOW()),
    ('${PLAN_TYPE_ID}', 'batches_maximum', NULL, NOW(), NOW());
"

info "Creating plan features..."

$MYSQL_CMD -e "
INSERT IGNORE INTO account_plan_feature (account_plan_id, \`key\`, enabled, created_at, updated_at) VALUES
    ('${PLAN_TYPE_ID}', 'customer_portal', 1, NOW(), NOW()),
    ('${PLAN_TYPE_ID}', 'sales_rep_dashboards', 1, NOW(), NOW()),
    ('${PLAN_TYPE_ID}', 'commission_tracking', 1, NOW(), NOW());
"

info "Creating billing record..."

$MYSQL_CMD -e "
INSERT INTO account_billing (
    id, account_plan_id, subscription_status, created_at, updated_at
) VALUES (
    '${BILLING_ID}', '${PLAN_TYPE_ID}', 'active', NOW(), NOW()
);
"

info "Linking billing to account..."

$MYSQL_CMD -e "
UPDATE account
SET account_billing_id = '${BILLING_ID}', updated_at = NOW()
WHERE id = '${ACCOUNT_ID}' AND account_billing_id IS NULL;
"

ROWS_AFFECTED=$($MYSQL_CMD -sN -e "SELECT ROW_COUNT();" 2>/dev/null || echo "0")
if [ "$ROWS_AFFECTED" = "0" ]; then
    warn "No rows updated — account may already have billing or does not exist."
fi

info "Phase 1 complete: DB setup done."

# ============================================================
# PHASE 2: Stripe Setup
# ============================================================

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

# --- Step 1: Set default payment method ---

info "Setting default payment method on customer..."
DEFAULT_PM_RESP=$(stripe_v1 POST "/v1/customers/${STRIPE_CUSTOMER_ID}" \
    -d "invoice_settings[default_payment_method]=${PAYMENT_METHOD_ID}")
check_error "$DEFAULT_PM_RESP" "Set default payment method"
info "Default payment method set."

# --- Step 2: Create billing profile ---

info "Creating billing profile..."
PROFILE_RESP=$(stripe_v2 POST /v2/billing/profiles \
    --json "{\"customer\":\"${STRIPE_CUSTOMER_ID}\"}")
check_error "$PROFILE_RESP" "Create billing profile"
PROFILE_ID=$(echo "$PROFILE_RESP" | jq -r '.id')
info "Created billing profile: $PROFILE_ID"

# --- Step 3: Create billing cadence ---

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
PLAN_RESP=$(stripe_v2 GET "/v2/billing/pricing_plans/${STRIPE_PRICING_PLAN_ID}")
check_error "$PLAN_RESP" "Fetch pricing plan"
LIVE_VERSION=$(echo "$PLAN_RESP" | jq -r '.live_version')
if [ -z "$LIVE_VERSION" ] || [ "$LIVE_VERSION" = "null" ]; then
    error "Pricing plan ${STRIPE_PRICING_PLAN_ID} has no live_version set."
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
                    \"pricing_plan\":\"${STRIPE_PRICING_PLAN_ID}\",
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

info "Fetching reserved billing intent for total amount..."
RESERVED_INTENT=$(stripe_v2 GET "/v2/billing/intents/${INTENT_ID}")
check_error "$RESERVED_INTENT" "Fetch reserved billing intent"

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
    -d "customer=${STRIPE_CUSTOMER_ID}" \
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
    exit 1
fi

# ============================================================
# PHASE 3: DB Update
# ============================================================

info "Updating account_billing record with Stripe IDs..."

$MYSQL_CMD -e "
    UPDATE account_billing SET
        internal_stripe_customer_id = '${STRIPE_CUSTOMER_ID}',
        stripe_billing_profile_id = '${PROFILE_ID}',
        stripe_billing_cadence_id = '${CADENCE_ID}',
        stripe_pricing_plan_subscription_id = '${SUBSCRIPTION_ID}',
        servicing_status = 'active',
        collection_status = 'current',
        updated_at = NOW()
    WHERE id = '${BILLING_ID}';
"

info ""
info "Done! Founder plan provisioned successfully."
info ""
info "Summary:"
info "  Account:      $ACCOUNT_ID"
info "  Plan:         $PLAN_TYPE_ID (Founder)"
info "  Billing:      $BILLING_ID"
info "  Customer:     $STRIPE_CUSTOMER_ID"
info "  Profile:      $PROFILE_ID"
info "  Cadence:      $CADENCE_ID"
info "  Subscription: $SUBSCRIPTION_ID"
