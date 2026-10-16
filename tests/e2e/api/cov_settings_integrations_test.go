//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Account Integrations (/v1/settings/integrations) coverage.
//
// There is no retrieve-by-id endpoint, so reads come from the list and from the write responses. The resource has no
// expandable fields.
//
// An account has one integration per provider and only 3 providers exist (stripe/shippo/hubspot). The main seed
// account already has `shippo` seeded (SeedAccountIntegrationID), which must not be mutated or deleted. That leaves 2
// free providers on the main account and 3 on Tenant B. To keep parallel tests off the same (account, provider) pair,
// each top-level test owns its slots and runs everything on them sequentially:
//   - SeedAccountID (main):   hubspot -> TestCovSettingsIntegrations_CRUD
//     stripe  -> TestCovSettingsIntegrations_CreateAndUpdateAllFields (and the webhook tests, under stripeConnectionMu)
//   - SeedTenantBAccountID:   hubspot, stripe -> TestCovSettingsIntegrations_OmittedFields
//     shippo  -> TestCovSettingsIntegrations_Preview9
//
// Validation-only requests never reach an insert or update, so they reuse providers freely. In the e2e stack the
// provider check is a stub: credentials containing "invalid" are rejected and those containing "unreachable" get no
// answer.

const covSettingsIntegrationsPath = "/v1/settings/integrations"

// stripeConnectionMu guards the main account's Stripe connection. The tests that connect Stripe
// hold the write side; tests asserting the account is unconnected hold the read side, or they
// would see this file's credentials mid-flight.
var stripeConnectionMu sync.RWMutex

func lockStripeConnected(t *testing.T) {
	t.Helper()
	stripeConnectionMu.Lock()
	t.Cleanup(stripeConnectionMu.Unlock)
}

func lockStripeUnconnected(t *testing.T) {
	t.Helper()
	stripeConnectionMu.RLock()
	t.Cleanup(stripeConnectionMu.RUnlock)
}

// covSettingsIntegrationsHubspotCredentials returns valid hubspot credentials.
func covSettingsIntegrationsHubspotCredentials(suffix string) map[string]any {
	return map[string]any{"access_token": "pat-e2e-" + suffix}
}

// covSettingsIntegrationsStripeCredentials returns valid (production/live) stripe credentials.
func covSettingsIntegrationsStripeCredentials(suffix string) map[string]any {
	return map[string]any{"private_key": "sk_live_e2e" + suffix, "publishable_key": "pk_live_e2e" + suffix, "webhook_secret": "whsec_e2e" + suffix}
}

// covSettingsIntegrationsShippoCredentials returns shippo credentials valid on the e2e stack, which
// is not production and so takes only test keys.
func covSettingsIntegrationsShippoCredentials(suffix string) map[string]any {
	return map[string]any{"api_key": "shippo_test_e2e" + suffix}
}

// assertIntegration checks every field of an integration response.
func assertIntegration(t *testing.T, got map[string]any, name, provider, status string) {
	t.Helper()
	assert.ElementsMatch(t, []string{"id", "object", "name", "provider", "status", "connection_status", "created_at", "updated_at"}, mapKeys(got))
	assertIDFormat(t, jsonField(got, "id"), "acig")
	assertObjectField(t, got, "account_integration")
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, provider, jsonField(got, "provider"))
	assert.Equal(t, status, jsonField(got, "status"))
	assert.Equal(t, "connected", jsonField(got, "connection_status"), "the provider accepted the credentials when they were saved")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
}

func integrationPath(id string) string { return covSettingsIntegrationsPath + "/" + id }

func rotateCredentialsPath(id string) string {
	return integrationPath(id) + "/actions/rotate-credentials"
}

// ──────────────────────────────────────────────
// CRUD lifecycle, duplicate provider, rotation
// ──────────────────────────────────────────────

func TestCovSettingsIntegrations_CRUD(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-covsi-crud")
	status, body, err := apiClient.Post(covSettingsIntegrationsPath, map[string]any{
		"name":        name,
		"provider":    "hubspot",
		"credentials": covSettingsIntegrationsHubspotCredentials("crud"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	created := parseJSON(body)
	id := jsonField(created, "id")
	assertIntegration(t, created, name, "hubspot", "active")
	assertListContainsID(t, covSettingsIntegrationsPath, nil, id)

	t.Run("SecondIntegrationForTheProviderConflicts", func(t *testing.T) {
		status, body, err := apiClient.Post(covSettingsIntegrationsPath, map[string]any{
			"name":        uniqueName("e2e-covsi-dup"),
			"provider":    "hubspot",
			"credentials": covSettingsIntegrationsHubspotCredentials("dup"),
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 409, status, body)
		errObj := requireErrorResponse(t, body, "resource_exists", "invalid_request_error")
		assertErrorParam(t, errObj, "provider")
	})

	t.Run("RotateCredentials", func(t *testing.T) {
		key := newIdempotencyKey()
		rotate := map[string]any{"credentials": covSettingsIntegrationsHubspotCredentials("rotated")}
		s1, b1, err := apiClient.Post(rotateCredentialsPath(id), rotate, key)
		require.NoError(t, err)
		requireStatus(t, 200, s1, b1)
		rotated := parseJSON(b1)
		assertIntegration(t, rotated, name, "hubspot", "active")
		assert.Equal(t, id, jsonField(rotated, "id"))
		assert.Equal(t, jsonField(created, "created_at"), jsonField(rotated, "created_at"))

		s2, b2, err := apiClient.Post(rotateCredentialsPath(id), rotate, key)
		require.NoError(t, err)
		requireStatus(t, 200, s2, b2)
		assert.JSONEq(t, string(b1), string(b2), "a repeated Idempotency-Key replays the rotation")
	})

	t.Run("RotateRejectedByTheProvider", func(t *testing.T) {
		status, body, err := apiClient.Post(rotateCredentialsPath(id), map[string]any{
			"credentials": map[string]any{"access_token": "pat-invalid"},
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 422, status, body)
		errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
		assertErrorParam(t, errObj, "credentials")
	})

	t.Run("RotateWhenTheProviderIsUnreachable", func(t *testing.T) {
		status, body, err := apiClient.Post(rotateCredentialsPath(id), map[string]any{
			"credentials": map[string]any{"access_token": "pat-unreachable"},
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 502, status, body)
		requireErrorResponse(t, body, "external_service_error", "api_error")
	})

	t.Run("RotateWithAnotherProvidersKeys", func(t *testing.T) {
		status, body, err := apiClient.Post(rotateCredentialsPath(id), map[string]any{
			"credentials": covSettingsIntegrationsShippoCredentials("wrong"),
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 422, status, body)
		errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
		assertErrorFieldParams(t, errObj, "credentials.access_token", "credentials.api_key")
	})

	t.Run("RotateMissingCredentials", func(t *testing.T) {
		status, body, err := apiClient.Post(rotateCredentialsPath(id), map[string]any{}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 422, status, body)
		errObj := requireErrorResponse(t, body, "", "invalid_request_error")
		assertErrorParam(t, errObj, "credentials")
	})

	newName := uniqueName("e2e-covsi-crud-upd")
	updStatus, updBody, err := apiClient.Patch(integrationPath(id), map[string]any{"name": newName, "status": "disabled"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, updStatus, updBody)
	updated := parseJSON(updBody)
	assertIntegration(t, updated, newName, "hubspot", "disabled")
	assert.Equal(t, id, jsonField(updated, "id"))

	delStatus, delBody, err := apiClient.Delete(integrationPath(id))
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)
	assertDeletedStub(t, delBody, id, "account_integration")

	del2Status, del2Body, err := apiClient.Delete(integrationPath(id))
	require.NoError(t, err)
	requireStatus(t, 410, del2Status, del2Body)
	requireErrorResponse(t, del2Body, "resource_gone", "invalid_request_error")
	assert.Nil(t, listFindByField(t, covSettingsIntegrationsPath, nil, "id", id), "a deleted integration is not listed")

	rotStatus, rotBody, err := apiClient.Post(rotateCredentialsPath(id), map[string]any{"credentials": covSettingsIntegrationsHubspotCredentials("gone")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, rotStatus, rotBody)

	// The hubspot slot is free again: the same Idempotency-Key replays one create.
	t.Run("CreateIdempotent", func(t *testing.T) {
		idemName := uniqueName("e2e-covsi-idem")
		idemKey := newIdempotencyKey()
		req := map[string]any{"name": idemName, "provider": "hubspot", "credentials": covSettingsIntegrationsHubspotCredentials("idem")}

		s1, b1, err := apiClient.Post(covSettingsIntegrationsPath, req, idemKey)
		require.NoError(t, err)
		requireStatus(t, 201, s1, b1)
		id1 := jsonField(parseJSON(b1), "id")
		defer apiClient.Delete(integrationPath(id1))

		s2, b2, err := apiClient.Post(covSettingsIntegrationsPath, req, idemKey)
		require.NoError(t, err)
		requireStatus(t, 201, s2, b2)
		assert.Equal(t, id1, jsonField(parseJSON(b2), "id"), "repeated Idempotency-Key with identical body returns the same integration")
	})
}

// assertErrorFieldParams checks a validation_failed error lists exactly these failing params.
func assertErrorFieldParams(t *testing.T, errObj map[string]any, params ...string) {
	t.Helper()
	rows, _ := errObj["errors"].([]any)
	got := make([]string, 0, len(rows))
	for _, r := range rows {
		if row, ok := r.(map[string]any); ok {
			got = append(got, jsonField(row, "param"))
		}
	}
	assert.ElementsMatch(t, params, got)
}

// ──────────────────────────────────────────────
// Create + Update All Fields
// ──────────────────────────────────────────────

func TestCovSettingsIntegrations_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()
	lockStripeConnected(t)

	name := uniqueName("e2e-covsi-allf")
	status, body, err := apiClient.Post(covSettingsIntegrationsPath, map[string]any{
		"name":        name,
		"provider":    "stripe",
		"credentials": covSettingsIntegrationsStripeCredentials("allf"),
		"status":      "disabled",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	created := parseJSON(body)
	id := jsonField(created, "id")
	assertIntegration(t, created, name, "stripe", "disabled")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(integrationPath(id)) })

	disabled, _, err := apiClient.GetList(covSettingsIntegrationsPath, url.Values{"statuses": {"disabled"}})
	require.NoError(t, err)
	found := false
	for _, item := range disabled.Data {
		assert.Equal(t, "disabled", DataItemField(item, "status"), "statuses[] narrows the list")
		found = found || DataItemField(item, "id") == id
	}
	assert.True(t, found, "statuses=disabled lists the disabled integration")

	updatedName := uniqueName("e2e-covsi-allf-upd")
	updStatus, updBody, err := apiClient.Patch(integrationPath(id), map[string]any{"name": updatedName, "status": "active"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, updStatus, updBody)
	updated := parseJSON(updBody)
	assertIntegration(t, updated, updatedName, "stripe", "active")
	assert.Equal(t, id, jsonField(updated, "id"))
	assert.Equal(t, jsonField(created, "created_at"), jsonField(updated, "created_at"))
	assert.NotEqual(t, jsonField(created, "updated_at"), jsonField(updated, "updated_at"), "updated_at advances")

	t.Run("UpdateIdempotent", func(t *testing.T) {
		idemKey := newIdempotencyKey()
		patch := map[string]any{"name": uniqueName("e2e-covsi-allf-idem")}
		s1, b1, err := apiClient.Patch(integrationPath(id), patch, idemKey)
		require.NoError(t, err)
		requireStatus(t, 200, s1, b1)
		s2, b2, err := apiClient.Patch(integrationPath(id), patch, idemKey)
		require.NoError(t, err)
		requireStatus(t, 200, s2, b2)
		assert.JSONEq(t, string(b1), string(b2), "a repeated Idempotency-Key replays the update")
	})

	t.Run("PutIsNotAForgeRoute", func(t *testing.T) {
		status, body, err := apiClient.Put(integrationPath(id), map[string]any{"name": "x"})
		require.NoError(t, err)
		requireStatus(t, 405, status, body)
	})
}

// ──────────────────────────────────────────────
// Omitted Fields
// ──────────────────────────────────────────────

func TestCovSettingsIntegrations_OmittedFields(t *testing.T) {
	t.Parallel()
	tenantB := apiClient.WithBearerToken(SeedTenantBAPIKey, SeedTenantBAccountID)

	t.Run("CreateWithOnlyRequiredFields", func(t *testing.T) {
		name := uniqueName("e2e-covsi-omit")
		status, body, err := tenantB.Post(covSettingsIntegrationsPath, map[string]any{
			"name":        name,
			"provider":    "hubspot",
			"credentials": covSettingsIntegrationsHubspotCredentials("omit"),
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)
		got := parseJSON(body)
		defer tenantB.Delete(integrationPath(jsonField(got, "id")))
		assertIntegration(t, got, name, "hubspot", "active")
	})

	t.Run("UpdatePreservesOmittedFields", func(t *testing.T) {
		name := uniqueName("e2e-covsi-pres")
		createStatus, createBody, err := tenantB.Post(covSettingsIntegrationsPath, map[string]any{
			"name":        name,
			"provider":    "stripe",
			"credentials": covSettingsIntegrationsStripeCredentials("pres"),
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, createStatus, createBody)
		created := parseJSON(createBody)
		id := jsonField(created, "id")
		defer tenantB.Delete(integrationPath(id))

		updStatus, updBody, err := tenantB.Patch(integrationPath(id), map[string]any{"status": "disabled"}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, updStatus, updBody)
		got := parseJSON(updBody)
		assertIntegration(t, got, name, "stripe", "disabled")
		assert.Equal(t, jsonField(created, "created_at"), jsonField(got, "created_at"))
	})
}

// ──────────────────────────────────────────────
// 1.0.forge-preview.9 compatibility
// ──────────────────────────────────────────────

// A preview.9 caller still sends credentials as a JSON string, reads inactive, updates with PUT and reconnects a
// provider by creating it again.
func TestCovSettingsIntegrations_Preview9(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithBearerToken(SeedTenantBAPIKey, SeedTenantBAccountID).WithAPIVersion(preview9Version)

	name := uniqueName("e2e-covsi-p9")
	status, body, err := preview.Post(covSettingsIntegrationsPath, map[string]any{
		"name": name, "provider": "shippo", "credentials": `{"api_key":"shippo_test_e2ep9"}`,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	created := parseJSON(body)
	id := jsonField(created, "id")
	t.Cleanup(func() { _, _, _ = preview.Delete(integrationPath(id)) })
	assert.ElementsMatch(t, []string{"id", "object", "name", "provider", "status", "created_at", "updated_at"}, mapKeys(created))
	assert.Equal(t, "active", jsonField(created, "status"))

	againName := uniqueName("e2e-covsi-p9-again")
	status, body, err = preview.Post(covSettingsIntegrationsPath, map[string]any{
		"name": againName, "provider": "shippo", "credentials": `{"api_key":"shippo_test_e2ep9b"}`,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	again := parseJSON(body)
	assert.Equal(t, id, jsonField(again, "id"), "creating a connected provider again rotates it in place")
	assert.Equal(t, againName, jsonField(again, "name"))

	status, body, err = preview.Put(integrationPath(id), map[string]any{"status": "inactive"})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "inactive", jsonField(parseJSON(body), "status"))

	forge := apiClient.WithBearerToken(SeedTenantBAPIKey, SeedTenantBAccountID)
	list, _, err := forge.GetList(covSettingsIntegrationsPath, url.Values{"statuses": {"disabled"}})
	require.NoError(t, err)
	found := false
	for _, item := range list.Data {
		found = found || DataItemField(item, "id") == id
	}
	assert.True(t, found, "preview.9's inactive is forge's disabled")

	status, body, err = preview.Post(covSettingsIntegrationsPath, map[string]any{
		"name": "x", "provider": "shippo", "credentials": "not-json",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

// ──────────────────────────────────────────────
// List
// ──────────────────────────────────────────────

func TestCovSettingsIntegrations_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(covSettingsIntegrationsPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 1, "should have at least the seeded shippo row")
}

func TestCovSettingsIntegrations_ListPagination(t *testing.T) {
	t.Parallel()
	// The main-account list is shared with other parallel tests in this file
	// that create/delete their own rows, so use the churn-tolerant helper
	// rather than asserting an exact row count.
	assertCursorPaginationAdvances(t, covSettingsIntegrationsPath, nil)
}

func TestCovSettingsIntegrations_ListSearch(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(covSettingsIntegrationsPath, url.Values{"q": {"Shippo"}, "statuses": {"active"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1)

	found := false
	for _, item := range list.Data {
		if DataItemField(item, "id") == SeedAccountIntegrationID {
			found = true
			assert.Equal(t, "Shippo Integration", DataItemField(item, "name"))
			assert.Equal(t, "shippo", DataItemField(item, "provider"))
			assert.Equal(t, "active", DataItemField(item, "status"))
			// Other tests rate and buy labels through this Shippo account, which settles it to connected.
			assert.Contains(t, []string{"unknown", "connected"}, DataItemField(item, "connection_status"))
		}
	}
	assert.True(t, found, "q=Shippo should find the seeded shippo integration (SeedAccountIntegrationID)")
}

func TestCovSettingsIntegrations_ListSearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(covSettingsIntegrationsPath, url.Values{"q": {"zzzznotaresource99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

func TestCovSettingsIntegrations_ListInvalidFilters(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(covSettingsIntegrationsPath, url.Values{"cursor": {"bogus___not_a_real_cursor"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "", "invalid_request_error")

	status, body, err = apiClient.GetListRaw(covSettingsIntegrationsPath, url.Values{"statuses": {"inactive"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "", "invalid_request_error"), "statuses")
}

func TestCovSettingsIntegrations_ListUnknownQueryParamRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(covSettingsIntegrationsPath, url.Values{bogusE2EQueryParam: {"1"}})
	require.NoError(t, err)
	assertUnknownQueryParamRejected(t, covSettingsIntegrationsPath, status, body)
}

// ──────────────────────────────────────────────
// Create Validation
// ──────────────────────────────────────────────

func TestCovSettingsIntegrations_CreateValidation(t *testing.T) {
	t.Parallel()

	post := func(t *testing.T, req map[string]any) (int, []byte) {
		t.Helper()
		status, body, err := apiClient.Post(covSettingsIntegrationsPath, req, newIdempotencyKey())
		require.NoError(t, err)
		return status, body
	}
	expect422 := func(t *testing.T, req map[string]any, param string) {
		t.Helper()
		status, body := post(t, req)
		requireStatus(t, 422, status, body)
		assertErrorParam(t, requireErrorResponse(t, body, "", "invalid_request_error"), param)
	}
	name := func() string { return uniqueName("e2e-covsi-val") }

	cases := []struct {
		label string
		req   map[string]any
		param string
	}{
		{"MissingName", map[string]any{"provider": "stripe", "credentials": covSettingsIntegrationsStripeCredentials("v1")}, "name"},
		{"EmptyName", map[string]any{"name": "", "provider": "stripe", "credentials": covSettingsIntegrationsStripeCredentials("v2")}, "name"},
		{"NameTooLong", map[string]any{"name": strings.Repeat("a", 256), "provider": "stripe", "credentials": covSettingsIntegrationsStripeCredentials("v3")}, "name"},
		{"MissingProvider", map[string]any{"name": name(), "credentials": covSettingsIntegrationsStripeCredentials("v4")}, "provider"},
		{"InvalidProviderEnum", map[string]any{"name": name(), "provider": "quickbooks", "credentials": map[string]any{}}, "provider"},
		{"MissingCredentials", map[string]any{"name": name(), "provider": "stripe"}, "credentials"},
		{"InvalidStatus", map[string]any{"name": name(), "provider": "hubspot", "credentials": covSettingsIntegrationsHubspotCredentials("v5"), "status": "inactive"}, "status"},
		{"StripeMissingKey", map[string]any{"name": name(), "provider": "stripe", "credentials": map[string]any{"private_key": "sk_live_x", "webhook_secret": "whsec_x"}}, "credentials.publishable_key"},
		{"KeyForAnotherProvider", map[string]any{"name": name(), "provider": "hubspot", "credentials": map[string]any{"access_token": "pat-x", "api_key": "shippo_test_x"}}, "credentials.api_key"},
		{"StripeMissingKeyPrefix", map[string]any{"name": name(), "provider": "stripe", "credentials": map[string]any{"private_key": "abc", "publishable_key": "def", "webhook_secret": "ghi"}}, "credentials.private_key"},
		{"StripeSandboxKeysOnProductionAccount", map[string]any{"name": name(), "provider": "stripe", "credentials": map[string]any{"private_key": "sk_test_x", "publishable_key": "pk_test_x", "webhook_secret": "whsec_x"}}, "credentials.private_key"},
		{"ShippoMissingKeyPrefix", map[string]any{"name": name(), "provider": "shippo", "credentials": map[string]any{"api_key": "abc"}}, "credentials.api_key"},
		// Only a production deployment may spend with a live key; the e2e stack is not one.
		{"ShippoLiveKeyOutsideProduction", map[string]any{"name": name(), "provider": "shippo", "credentials": map[string]any{"api_key": "shippo_live_x"}}, "credentials.api_key"},
		{"HubspotMissingTokenPrefix", map[string]any{"name": name(), "provider": "hubspot", "credentials": map[string]any{"access_token": "abc"}}, "credentials.access_token"},
		{"RejectedByTheProvider", map[string]any{"name": name(), "provider": "stripe", "credentials": covSettingsIntegrationsStripeCredentials("invalid")}, "credentials"},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			t.Parallel()
			expect422(t, tc.req, tc.param)
		})
	}

	t.Run("CredentialsAsAString", func(t *testing.T) {
		t.Parallel()
		expect422(t, map[string]any{"name": name(), "provider": "stripe", "credentials": "not-json"}, "credentials")
	})

	t.Run("ProviderUnreachable", func(t *testing.T) {
		t.Parallel()
		status, body := post(t, map[string]any{"name": name(), "provider": "shippo", "credentials": covSettingsIntegrationsShippoCredentials("unreachable")})
		requireStatus(t, 502, status, body)
		requireErrorResponse(t, body, "external_service_error", "api_error")
	})

	t.Run("MalformedJSONUnknownFieldRejected", func(t *testing.T) {
		t.Parallel()
		status, body := post(t, map[string]any{
			bogusE2EJSONField: "1",
			"name":            name(),
			"provider":        "stripe",
			"credentials":     covSettingsIntegrationsStripeCredentials("valunk"),
		})
		assertJSONUnknownFieldRejected(t, "POST", covSettingsIntegrationsPath, status, body)
	})

	t.Run("UnknownCredentialKeyRejected", func(t *testing.T) {
		t.Parallel()
		status, body := post(t, map[string]any{"name": name(), "provider": "hubspot", "credentials": map[string]any{"access_token": "pat-x", "secret": "y"}})
		requireStatus(t, 400, status, body)
		requireErrorResponse(t, body, "parameter_unknown", "invalid_request_error")
	})
}

// ──────────────────────────────────────────────
// Update Validation / Not Found / Cross-Tenant
// ──────────────────────────────────────────────

func TestCovSettingsIntegrations_UpdateValidation(t *testing.T) {
	t.Parallel()

	// These target the seeded, read-only SeedAccountIntegrationID row; a rejected PATCH never mutates it.
	patch := func(t *testing.T, client *Client, id string, req map[string]any) (int, []byte) {
		t.Helper()
		status, body, err := client.Patch(integrationPath(id), req, newIdempotencyKey())
		require.NoError(t, err)
		return status, body
	}

	for label, tc := range map[string]struct {
		req   map[string]any
		param string
	}{
		"NullNameRejected":        {map[string]any{"name": nil}, "name"},
		"BlankNameRejected":       {map[string]any{"name": ""}, "name"},
		"NullStatusRejected":      {map[string]any{"status": nil}, "status"},
		"BogusStatusEnumRejected": {map[string]any{"status": "bogus"}, "status"},
		"InactiveIsNotAStatus":    {map[string]any{"status": "inactive"}, "status"},
		"CredentialsNotUpdatable": {map[string]any{"credentials": covSettingsIntegrationsShippoCredentials("x")}, ""},
		"ProviderNotUpdatable":    {map[string]any{"provider": "hubspot"}, ""},
	} {
		t.Run(label, func(t *testing.T) {
			t.Parallel()
			status, body := patch(t, apiClient, SeedAccountIntegrationID, tc.req)
			if tc.param == "" {
				requireStatus(t, 400, status, body)
				requireErrorResponse(t, body, "parameter_unknown", "invalid_request_error")
				return
			}
			requireStatus(t, 422, status, body)
			assertErrorParam(t, requireErrorResponse(t, body, "", "invalid_request_error"), tc.param)
		})
	}

	t.Run("UpdateNonexistentIDReturns404", func(t *testing.T) {
		t.Parallel()
		status, body := patch(t, apiClient, "acig_doesnotexist00000000000", map[string]any{"name": uniqueName("e2e-covsi-404")})
		requireStatus(t, 404, status, body)
	})

	t.Run("DeleteNonexistentIDReturns404", func(t *testing.T) {
		t.Parallel()
		status, body, err := apiClient.Delete(integrationPath("acig_doesnotexist00000000000"))
		require.NoError(t, err)
		requireStatus(t, 404, status, body)
	})

	t.Run("CrossTenantUpdateReturns404", func(t *testing.T) {
		t.Parallel()
		tenantB := apiClient.WithBearerToken(SeedTenantBAPIKey, SeedTenantBAccountID)
		status, body := patch(t, tenantB, SeedAccountIntegrationID, map[string]any{"name": uniqueName("e2e-covsi-crosstenant")})
		requireStatus(t, 404, status, body)
	})

	t.Run("CrossTenantRotateReturns404", func(t *testing.T) {
		t.Parallel()
		tenantB := apiClient.WithBearerToken(SeedTenantBAPIKey, SeedTenantBAccountID)
		status, body, err := tenantB.Post(rotateCredentialsPath(SeedAccountIntegrationID), map[string]any{"credentials": covSettingsIntegrationsShippoCredentials("x")}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 404, status, body)
	})

	t.Run("CrossTenantListDoesNotLeakOtherAccountRows", func(t *testing.T) {
		t.Parallel()
		tenantB := apiClient.WithBearerToken(SeedTenantBAPIKey, SeedTenantBAccountID)
		list, _, err := tenantB.GetList(covSettingsIntegrationsPath, nil)
		require.NoError(t, err)
		for _, item := range list.Data {
			assert.NotEqual(t, SeedAccountIntegrationID, DataItemField(item, "id"),
				"Tenant B's list must not include the main account's seeded integration")
		}
	})
}
