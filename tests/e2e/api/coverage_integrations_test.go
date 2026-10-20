//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Endpoints that talk to somebody else — Stripe, Shippo, HubSpot.
//
// These run against the stack's stub clients rather than the real services, so what is under
// test is our own routing, scoping, and error handling: which events we act on, whose records
// a caller may reach, and what a caller is told when the far side has nothing configured.

const stripeIntPath = "/v1/settings/integrations/stripe"

// ──────────────────────────────────────────────
// Stripe connect status
// ──────────────────────────────────────────────

func TestStripeIntegration_StatusIsReported(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(stripeIntPath+"/status", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)
	assert.NotEmpty(t, jsonField(parseJSON(body), "status"), "the connection state must be named: %s", string(body))
}

// An account that has not connected Stripe has no publishable key to hand out, and saying so
// is what lets the dashboard offer the connect flow instead of rendering a broken checkout.
func TestStripeIntegration_PublishableKeyAbsentUntilConnected(t *testing.T) {
	t.Parallel()
	lockStripeUnconnected(t)

	status, body, err := apiClient.GetListRaw(stripeIntPath+"/publishable-key", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 404, status, "an unconnected account has no publishable key: %s", string(body))
}

// ──────────────────────────────────────────────
// Stripe webhooks
// ──────────────────────────────────────────────

const stripeWebhookPath = "/v1/webhooks/stripe"

// The account-scoped webhook carries a seller's own Stripe events. An account that has not
// connected Stripe has no secret to verify against, so there is nothing to accept.
func TestStripeAccountWebhook_UnconnectedAccountIsRefused(t *testing.T) {
	t.Parallel()
	lockStripeUnconnected(t)

	status, body, err := apiClient.PostSigned(stripeWebhookPath+"/accounts/"+SeedAccountID, StubStripeSignature,
		[]byte(`{"id":"evt_e2e_acct","type":"payment_intent.succeeded","data":{"object":{"id":"pi_e2e"}}}`))
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Contains(t, []int{400, 404}, status, "an unconnected account cannot verify events: %s", string(body))
}

func TestStripeAccountWebhook_UnknownAccountIsRefused(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.PostSigned(stripeWebhookPath+"/accounts/ac_doesnotexist00000", StubStripeSignature,
		[]byte(`{"id":"evt_e2e_acct2","type":"payment_intent.succeeded","data":{"object":{"id":"pi_e2e"}}}`))
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Contains(t, []int{400, 404}, status, "an unknown account cannot verify events: %s", string(body))
}

// ──────────────────────────────────────────────
// Carriers
// ──────────────────────────────────────────────

func TestCarriers_OAuthStatusReflectsWhetherAnAccountIsLinked(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(carriersPath+"/"+SeedShippoCarrierID+"/oauth-status", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)
	assert.Equal(t, "connected", jsonField(parseJSON(body), "status"), "a carrier with a Shippo account reads as connected: %s", string(body))

	status, body, err = apiClient.GetListRaw(carriersPath+"/"+SeedCarrierID+"/oauth-status", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)
	assert.Equal(t, "disconnected", jsonField(parseJSON(body), "status"), "a carrier without one reads as disconnected: %s", string(body))
}

func TestCarriers_OAuthStatusForUnknownCarrierIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(carriersPath+"/cr_doesnotexist00000/oauth-status", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 404, status, "an unknown carrier must 404: %s", string(body))
}

func TestCarriers_SyncOptionsRefreshesServiceLevels(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(carriersPath+"/"+SeedSyncCarrierID+"/actions/sync-options", nil, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)
	assert.Equal(t, SeedSyncCarrierID, jsonField(parseJSON(body), "id"), "the synced carrier is returned: %s", string(body))
}

// Service levels come from the carrier's Shippo account. Without one there is nothing to sync
// from, and the caller needs to be told that rather than shown an empty list.
func TestCarriers_SyncOptionsNeedsAShippoAccount(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(carriersPath+"/"+SeedCarrierID+"/actions/sync-options", nil, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 400, status, "a carrier with no Shippo account cannot sync: %s", string(body))
	assert.Contains(t, string(body), "Shippo", "the refusal must say what is missing: %s", string(body))
}

func TestCarriers_SyncOptionsForUnknownCarrierIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(carriersPath+"/cr_doesnotexist00000/actions/sync-options", nil, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 404, status, "an unknown carrier must 404: %s", string(body))
}

// ──────────────────────────────────────────────
// HubSpot
// ──────────────────────────────────────────────

func TestHubspotSync_RecordsAreListed(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(hubspotSyncPath+"/records", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	assert.NotEmpty(t, list.Data, "the seeded mappings must be listed")
}

// The type filter is how the dashboard shows only the mappings for one kind of record.
func TestHubspotSync_RecordsFilterByType(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(hubspotSyncPath+"/records", url.Values{"augno_type": {"customer"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	assert.NotEmpty(t, list.Data, "the seeded mappings are customers")

	list, status, err = apiClient.GetList(hubspotSyncPath+"/records", url.Values{"augno_type": {"contact"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	assert.Empty(t, list.Data, "no contact mappings are seeded, so the filter must exclude the customer ones")
}

func TestHubspotSync_CancelStopsAnInFlightJob(t *testing.T) {
	t.Parallel()
	resetHubspotJob(t, SeedHubspotCancelJobID)

	status, body, err := apiClient.Post(hubspotSyncPath+"/"+SeedHubspotCancelJobID+"/actions/cancel", nil, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	// A cancelled sync is recorded as failed, with the reason, so a half-applied backfill is
	// never mistaken for one that finished.
	assert.Equal(t, "failed", jsonField(parseJSON(body), "status"), "a cancelled sync must not read as complete: %s", string(body))
}

// A finished sync has nothing left to stop, and cancelling it would rewrite a completed run's
// outcome as a failure.
func TestHubspotSync_CancelRejectsAFinishedJob(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(hubspotSyncPath+"/"+SeedHubspotCompletedJobID+"/actions/cancel", nil, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 400, status, "a finished sync cannot be cancelled: %s", string(body))
}

func TestHubspotSync_CancelUnknownJobIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(hubspotSyncPath+"/igjb_doesnotexist0/actions/cancel", nil, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 404, status, "an unknown sync job must 404: %s", string(body))
}

// A sync job belongs to one account, so another tenant must not be able to stop it.
func TestHubspotSync_CancelRejectsAnotherTenantsJob(t *testing.T) {
	t.Parallel()

	tenantB := apiClient.WithBearerToken(SeedTenantBAPIKey, SeedTenantBAccountID)

	status, body, err := tenantB.Post(hubspotSyncPath+"/"+SeedHubspotSyncJobID+"/actions/cancel", nil, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 404, status, "another tenant's sync job must not be reachable: %s", string(body))
}

// ──────────────────────────────────────────────
// Receivables export
// ──────────────────────────────────────────────

func TestReceivables_ExportByCustomer(t *testing.T) {
	t.Parallel()

	resp, err := apiClient.GetFull("/v1/finance/receivables/accounts/"+SeedCustomerAccountID+"/actions/export", nil)
	require.NoError(t, err)
	require.Less(t, resp.StatusCode, 500, "must not 5xx: %s", string(resp.Body))
	requireStatus(t, 200, resp.StatusCode, resp.Body)

	// A ledger export without its header is not something a finance team can open.
	assert.Contains(t, string(resp.Body), "Invoice Number", "the export must carry a header row: %s", string(resp.Body))
}

// The cutoff is what makes an aging report reproducible, so an unparseable one must be
// rejected rather than silently treated as "now".
func TestReceivables_ExportRejectsAnUnparseableCutoff(t *testing.T) {
	t.Parallel()

	resp, err := apiClient.GetFull("/v1/finance/receivables/accounts/"+SeedCustomerAccountID+"/actions/export",
		url.Values{"cutoff_at": {"last-tuesday"}})
	require.NoError(t, err)
	require.Less(t, resp.StatusCode, 500, "must not 5xx: %s", string(resp.Body))
	assert.Equal(t, 400, resp.StatusCode, "an unparseable cutoff must be rejected: %s", string(resp.Body))
}

func TestReceivables_ExportForUnknownCustomerIs404(t *testing.T) {
	t.Parallel()

	resp, err := apiClient.GetFull("/v1/finance/receivables/accounts/ac_doesnotexist00000/actions/export", nil)
	require.NoError(t, err)
	require.Less(t, resp.StatusCode, 500, "must not 5xx: %s", string(resp.Body))
	assert.Equal(t, 404, resp.StatusCode, "an unknown customer must 404: %s", string(resp.Body))
}
