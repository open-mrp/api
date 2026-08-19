//go:build e2e

package api_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Read endpoints the suite had never called.
//
// Grouped by shape rather than domain: each returns a derived view rather than a stored
// resource, which is where a wrong answer is hardest to notice — a report that quietly
// returns nothing looks exactly like a report with nothing to say.

func TestHealthCheck_ReportsHealthy(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw("/healthz", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "healthz must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)
}

// loginAsSeedUser signs the seeded user in and returns a client carrying the resulting
// session cookies. Access tokens are refused in the Authorization header by design, so
// this is the only way to reach an endpoint that wants a person rather than an API key.
func loginAsSeedUser(t *testing.T) *Client {
	t.Helper()

	resp, err := apiClient.PostFull(loginPath, map[string]any{
		"identifier": seedUserEmail,
		"password":   seedUserPassword,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)

	cookies := (&http.Response{Header: resp.Header}).Cookies()
	require.NotEmpty(t, cookies, "login must hand back session cookies")

	return apiClient.WithCookies(cookies, SeedAccountID)
}

func TestIdentityMe_ReturnsTheAuthenticatedUser(t *testing.T) {
	t.Parallel()

	status, body, err := loginAsSeedUser(t).GetListRaw("/v1/identity/me", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "me must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	me := parseJSON(body)
	assert.Equal(t, SeedUserID, jsonField(me, "id"), "me must describe the user who logged in: %s", string(body))
	assert.Equal(t, seedUserEmail, jsonField(me, "email"), "the caller's email is what the dashboard renders")
}

// An API key belongs to an account, not a person, so there is no "me" to return. Answering
// with the key's owner would let a machine credential impersonate whoever provisioned it.
func TestIdentityMe_RejectsAnAPIKey(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw("/v1/identity/me", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 403, status, "an API key has no user identity: %s", string(body))
}

// The resource-type list is what an audit-log filter is built from, so an empty one silently makes the filter unusable.
func TestAuditEvents_ResourceTypesAreListed(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList("/v1/core/audit-events/resource-types", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	assert.NotEmpty(t, list.Data, "the audit filter needs resource types to offer")
}

func TestSalesOrders_StatusesAreListed(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/statuses", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "statuses must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)
	for _, want := range []string{"estimate", "issued"} {
		assert.Contains(t, string(body), want, "status %q should be listed", want)
	}
}

// ──────────────────────────────────────────────
// Catalog reads
// ──────────────────────────────────────────────

func TestItems_CostsBreakDownIntoTheirParts(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(itemsPath+"/"+SeedItemID+"/costs", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "costs must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	costs := parseJSON(body)
	// The parts have to be present even at zero, because the breakdown is the answer — a bare total tells a merchant nothing about where the cost came from.
	for _, field := range []string{"direct_material_cost", "direct_labor_cost", "overhead_cost", "total_cost"} {
		assert.Contains(t, costs, field, "the cost breakdown must report %s: %s", field, string(body))
	}
}

func TestItems_CostsForUnknownItemIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(itemsPath+"/it_doesnotexist00000/costs", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must 404 rather than 5xx: %s", string(body))
	assert.Equal(t, 404, status, "costs for an unknown item must 404: %s", string(body))
}

// A purchased material is bought, not made, so there is no recipe to price. That has to read as
// not-found rather than a breakdown of zeroes, which would look like a made item that costs nothing.
func TestItems_CostsForAPurchasedMaterialIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(itemsPath+"/"+SeedPurchasedItemID+"/costs", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must 404 rather than 5xx: %s", string(body))
	assert.Equal(t, 404, status, "an item with no production flow must 404: %s", string(body))
}

func TestItems_TrendsReturnAPointSeries(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(itemsPath+"/"+SeedItemID+"/trends", url.Values{"trend_type": {"inventory"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "trends must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	trends := parseJSON(body)
	assert.Equal(t, "inventory", jsonField(trends, "trend_type"), "the series must report the metric asked for: %s", string(body))
	assert.Contains(t, trends, "points", "a trend without points is not a trend: %s", string(body))
}

// The metric is what makes the series meaningful, so an absent or unrecognized one is rejected rather than defaulted.
func TestItems_TrendsRequireAKnownMetric(t *testing.T) {
	t.Parallel()

	for name, params := range map[string]url.Values{
		"omitted":      nil,
		"empty":        {"trend_type": {""}},
		"unrecognized": {"trend_type": {"weather"}},
	} {
		t.Run(name, func(t *testing.T) {
			status, body, err := apiClient.GetListRaw(itemsPath+"/"+SeedItemID+"/trends", params)
			require.NoError(t, err)
			require.Less(t, status, 500, "must not 5xx: %s", string(body))
			assert.Equal(t, 400, status, "a %s trend_type must be rejected: %s", name, string(body))
		})
	}
}

func TestItems_TrendsForUnknownItemIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(itemsPath+"/it_doesnotexist00000/trends", url.Values{"trend_type": {"inventory"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "must 404 rather than 5xx: %s", string(body))
	assert.Equal(t, 404, status, "trends for an unknown item must 404: %s", string(body))
}

func TestItems_ExportReturnsAFile(t *testing.T) {
	t.Parallel()

	resp, err := apiClient.GetFull(itemsPath+"/actions/export", nil)
	require.NoError(t, err)
	require.Less(t, resp.StatusCode, 500, "export must not 5xx: %s", string(resp.Body))
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	assert.NotEmpty(t, resp.Body, "an export with no bytes is not an export")
}

// ──────────────────────────────────────────────
// Analytics and exports
// ──────────────────────────────────────────────

func TestAnalytics_WeeksOfSales(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw("/v1/core/analytics/weeks-of-sales", url.Values{"period_in_weeks": {"13"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "weeks-of-sales must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)
}

// A period has to be a positive number of weeks; zero or negative would divide sales by nothing.
func TestAnalytics_WeeksOfSalesRejectsANonsensePeriod(t *testing.T) {
	t.Parallel()

	for _, period := range []string{"0", "-4", "not-a-number"} {
		status, body, err := apiClient.GetListRaw("/v1/core/analytics/weeks-of-sales", url.Values{"period_in_weeks": {period}})
		require.NoError(t, err)
		require.Less(t, status, 500, "period %q must not 5xx: %s", period, string(body))
		assert.Equal(t, 400, status, "period %q must be rejected: %s", period, string(body))
	}
}

func TestInventoryChangeLogs_Export(t *testing.T) {
	t.Parallel()

	resp, err := apiClient.GetFull("/v1/operations/inventory-change-logs/actions/export", nil)
	require.NoError(t, err)
	require.Less(t, resp.StatusCode, 500, "export must not 5xx: %s", string(resp.Body))
	requireStatus(t, 200, resp.StatusCode, resp.Body)
}

// ──────────────────────────────────────────────
// Messaging counters
// ──────────────────────────────────────────────

// The unread badge is read on every page load, so it has to answer with numbers rather than nulls even when nothing is unread.
func TestNotifications_UnreadCountAlwaysReportsNumbers(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw("/v1/messaging/notifications/unread-count", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "unread-count must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	counts := parseJSON(body)
	for _, field := range []string{"notifications", "conversations", "total"} {
		assert.Contains(t, counts, field, "unread-count must report %s: %s", field, string(body))
	}
}

func TestNotifications_UnreadSummaryAcrossAccounts(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw("/v1/messaging/notifications/unread-summary", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "unread-summary must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	summary := parseJSON(body)
	assert.Contains(t, summary, "total", "the cross-account summary must carry a total: %s", string(body))
	assert.Contains(t, summary, "accounts", "the cross-account summary must carry per-account rows: %s", string(body))
}

// ──────────────────────────────────────────────
// Slug lookups — public-facing, so a wrong answer leaks another account
// ──────────────────────────────────────────────

func TestBranding_BySlug(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw("/v1/settings/branding/"+SeedAccountSlug, nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "branding must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)
	assert.Equal(t, SeedAccountSlug, jsonField(parseJSON(body), "slug"), "the slug asked for is the slug returned")
}

func TestBranding_UnknownSlugIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw("/v1/settings/branding/zzz-no-such-account", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must 404 rather than 5xx: %s", string(body))
	assert.Equal(t, 404, status, "an unknown slug must 404: %s", string(body))
}

func TestPortalProfile_BySlug(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw("/v1/settings/portal-profiles/"+SeedAccountSlug, nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "portal profile must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)
	assert.Equal(t, SeedAccountSlug, jsonField(parseJSON(body), "slug"))
}

func TestPortalProfile_UnknownSlugIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw("/v1/settings/portal-profiles/zzz-no-such-account", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must 404 rather than 5xx: %s", string(body))
	assert.Equal(t, 404, status, "an unknown slug must 404: %s", string(body))
}

func TestRegistrationFlow_BySlug(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw("/v1/sales/registration-flows/by-slug/"+SeedAccountSlug, nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "by-slug must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)
	assert.NotEmpty(t, jsonField(parseJSON(body), "id"), "the account's registration flow: %s", string(body))
}

func TestRegistrationFlow_UnknownSlugIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw("/v1/sales/registration-flows/by-slug/zzz-no-such-account", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must 404 rather than 5xx: %s", string(body))
	assert.Equal(t, 404, status, "an unknown slug must 404: %s", string(body))
}

// ──────────────────────────────────────────────
// System properties
// ──────────────────────────────────────────────

// The latest value is what the next generated number is based on, so an endpoint that cannot answer would hand out a duplicate.
func TestSystemProperties_LatestValue(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw("/v1/settings/properties/customer_number/latest-value", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "latest-value must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)
	assert.Contains(t, parseJSON(body), "value", "the latest value must be reported: %s", string(body))
}

// Reading a counter initializes it, so an unrecognized code must be caught before it
// reaches the database — otherwise the failure arrives as a foreign key conflict.
func TestSystemProperties_LatestValueRejectsAnUnknownType(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw("/v1/settings/properties/zzz_no_such_type/latest-value", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 400, status, "an unknown property type must be rejected: %s", string(body))
	assert.Contains(t, string(body), "type_code", "the error must name the offending parameter: %s", string(body))
}

// Every documented counter has to answer, since each one is what its resource's next number is drawn from.
func TestSystemProperties_EveryCounterTypeAnswers(t *testing.T) {
	t.Parallel()

	for _, code := range []string{
		"transaction_number", "settlement_number", "sales_order_number", "purchase_order_number",
		"supplier_number", "customer_number", "sscc_count", "production_run_number",
	} {
		t.Run(code, func(t *testing.T) {
			status, body, err := apiClient.GetListRaw("/v1/settings/properties/"+code+"/latest-value", nil)
			require.NoError(t, err)
			require.Less(t, status, 500, "%s must not 5xx: %s", code, string(body))
			requireStatus(t, 200, status, body)
			assert.NotEmpty(t, jsonField(parseJSON(body), "value"), "%s must report a value: %s", code, string(body))
		})
	}
}

// ──────────────────────────────────────────────
// EDI
// ──────────────────────────────────────────────

func TestEdiRuns_RetrieveUnknownRunIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw("/v1/operations/edi-runs/edir_doesnotexist0", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must 404 rather than 5xx: %s", string(body))
	assert.Equal(t, 404, status, "an unknown EDI run must 404: %s", string(body))
}
