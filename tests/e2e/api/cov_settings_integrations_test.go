//go:build e2e

package api_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Account Integrations (/v1/settings/integrations) coverage.
//
// This resource has only List/Create/Update(PUT)/Delete — there is no
// retrieve-by-id endpoint. "Get" coverage therefore comes from list-scanning
// for created ids and from the Create/Update/Delete response bodies
// themselves. The resource struct has no `expandable:"true"` fields and the
// list operation accepts no `include` param, so the Expandable Fields
// category is N/A for this group.
//
// Create is an upsert-by-(account,provider): a second create for a provider
// code that already exists on the account updates the existing row instead
// of erroring (confirmed live), and only 3 provider codes exist
// (stripe/shippo/hubspot). The main seed account already has `shippo` seeded
// (SeedAccountIntegrationID) and must not be mutated or deleted. That leaves
// exactly 2 free codes on the main account and 3 free codes on Tenant B
// (which has zero seeded account_integration rows). To avoid parallel tests
// racing on the same (account, provider) pair, codes are allocated as
// follows and never shared across top-level tests:
//   - SeedAccountID (main):        hubspot -> TestCovSettingsIntegrations_CRUD
//                                  stripe  -> TestCovSettingsIntegrations_CreateAndUpdateAllFields
//   - SeedTenantBAccountID:        hubspot, stripe (sequential subtests) -> TestCovSettingsIntegrations_OmittedFields
//                                  shippo  -> TestCovSettingsIntegrations_CreateResponseShape
//
// Idempotency coverage (create-idempotency-key-replay, create-as-upsert with
// a different key, and update-idempotency-key-replay) is folded in as extra
// sequential subtests appended to TestCovSettingsIntegrations_CRUD and
// TestCovSettingsIntegrations_CreateAndUpdateAllFields, reusing the same
// (account, provider) slot only after that test's primary row has already
// been deleted — there is no 6th free (account, provider) slot to allocate
// a fully independent top-level idempotency test.
//
// Validation-only requests never reach the DB insert/update, so they reuse
// provider codes freely across parallel subtests, and null/blank Update
// validation subtests target the read-only seeded SeedAccountIntegrationID
// row (confirmed live: a 400-rejected PUT does not mutate the stored row).

const covSettingsIntegrationsPath = "/v1/settings/integrations"

// covSettingsIntegrationsHubspotCredentials returns a valid hubspot credentials JSON string.
func covSettingsIntegrationsHubspotCredentials(suffix string) string {
	return `{"access_token":"pat-e2e-` + suffix + `"}`
}

// covSettingsIntegrationsStripeCredentials returns valid (production/live) stripe credentials JSON.
func covSettingsIntegrationsStripeCredentials(suffix string) string {
	return `{"private_key":"sk_live_e2e` + suffix + `","publishable_key":"pk_live_e2e` + suffix + `","webhook_secret":"whsec_e2e` + suffix + `"}`
}

// covSettingsIntegrationsShippoCredentials returns valid (production/live) shippo credentials JSON.
func covSettingsIntegrationsShippoCredentials(suffix string) string {
	return `{"api_key":"shippo_live_e2e` + suffix + `"}`
}

// ──────────────────────────────────────────────
// CRUD Lifecycle (+ folded-in idempotency)
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
	require.NotEmpty(t, id)
	assertIDFormat(t, id, "acig")
	assertObjectField(t, created, "account_integration")
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, "hubspot", jsonField(created, "provider"))
	assert.Equal(t, "active", jsonField(created, "status"))

	// No GET-by-id endpoint exists — verify presence via list scan instead.
	assertListContainsID(t, covSettingsIntegrationsPath, nil, id)

	// UPDATE (PUT, not PATCH).
	newName := uniqueName("e2e-covsi-crud-upd")
	updStatus, updBody, err := apiClient.Put(covSettingsIntegrationsPath+"/"+id, map[string]any{
		"name":   newName,
		"status": "inactive",
	})
	require.NoError(t, err)
	requireStatus(t, 200, updStatus, updBody)

	updated := parseJSON(updBody)
	assert.Equal(t, id, jsonField(updated, "id"))
	assert.Equal(t, newName, jsonField(updated, "name"))
	assert.Equal(t, "inactive", jsonField(updated, "status"))
	assert.Equal(t, "hubspot", jsonField(updated, "provider"), "provider should not change on update")

	assertListContainsID(t, covSettingsIntegrationsPath, nil, id)

	// DELETE — returns the deleted resource body.
	delStatus, delBody, err := apiClient.Delete(covSettingsIntegrationsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)
	deleted := parseJSON(delBody)
	assert.Equal(t, id, jsonField(deleted, "id"))

	// Second delete -> 410 Gone (a real behavioral distinction for this
	// resource: most others return 404 on a second delete).
	del2Status, del2Body, err := apiClient.Delete(covSettingsIntegrationsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 410, del2Status, del2Body)
	requireErrorResponse(t, del2Body, "resource_gone", "invalid_request_error")

	assert.Nil(t, listFindByField(t, covSettingsIntegrationsPath, nil, "id", id),
		"deleted row should no longer appear in the list")

	// --- Idempotency coverage, folded in here to reuse the now-free
	// (SeedAccountID, hubspot) slot sequentially rather than claim a 6th
	// (account, provider) pair that does not exist. ---

	t.Run("CreateIdempotent", func(t *testing.T) {
		idemName := uniqueName("e2e-covsi-idem")
		idemKey := newIdempotencyKey()
		creds := covSettingsIntegrationsHubspotCredentials("idem")

		s1, b1, err := apiClient.Post(covSettingsIntegrationsPath, map[string]any{
			"name":        idemName,
			"provider":    "hubspot",
			"credentials": creds,
		}, idemKey)
		require.NoError(t, err)
		requireStatus(t, 201, s1, b1)
		id1 := jsonField(parseJSON(b1), "id")
		require.NotEmpty(t, id1)
		defer apiClient.Delete(covSettingsIntegrationsPath + "/" + id1)

		s2, b2, err := apiClient.Post(covSettingsIntegrationsPath, map[string]any{
			"name":        idemName,
			"provider":    "hubspot",
			"credentials": creds,
		}, idemKey)
		require.NoError(t, err)
		requireStatus(t, 201, s2, b2)
		assert.Equal(t, id1, jsonField(parseJSON(b2), "id"),
			"repeated Idempotency-Key with identical body should return the same resource")

		// A *different* Idempotency-Key targeting the same provider takes the
		// create-as-upsert path (distinct from idempotency-key caching): 201
		// again, same row id (upsert-by-provider), but the name is updated.
		upsertName := uniqueName("e2e-covsi-idem-upsert")
		s3, b3, err := apiClient.Post(covSettingsIntegrationsPath, map[string]any{
			"name":        upsertName,
			"provider":    "hubspot",
			"credentials": covSettingsIntegrationsHubspotCredentials("idem2"),
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, s3, b3)
		upserted := parseJSON(b3)
		assert.Equal(t, id1, jsonField(upserted, "id"), "create-as-upsert keeps the same row id for the provider")
		assert.Equal(t, upsertName, jsonField(upserted, "name"), "create-as-upsert updates the name")
	})
}

// ──────────────────────────────────────────────
// Create + Update All Fields (+ folded-in update idempotency)
// ──────────────────────────────────────────────

func TestCovSettingsIntegrations_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-covsi-allf")
	status, body, err := apiClient.Post(covSettingsIntegrationsPath, map[string]any{
		"name":        name,
		"provider":    "stripe",
		"credentials": covSettingsIntegrationsStripeCredentials("allf"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	created := parseJSON(body)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)

	// Every field on the AccountIntegration resource struct.
	assertIDFormat(t, id, "acig")
	assertObjectField(t, created, "account_integration")
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, "stripe", jsonField(created, "provider"))
	assert.Equal(t, "active", jsonField(created, "status"))
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")
	origCreatedAt := jsonField(created, "created_at")
	origUpdatedAt := jsonField(created, "updated_at")

	// UPDATE both mutable fields.
	updatedName := uniqueName("e2e-covsi-allf-upd")
	updStatus, updBody, err := apiClient.Put(covSettingsIntegrationsPath+"/"+id, map[string]any{
		"name":   updatedName,
		"status": "inactive",
	})
	require.NoError(t, err)
	requireStatus(t, 200, updStatus, updBody)

	updated := parseJSON(updBody)
	assert.Equal(t, id, jsonField(updated, "id"), "id must not change")
	assertObjectField(t, updated, "account_integration")
	assert.Equal(t, updatedName, jsonField(updated, "name"))
	assert.Equal(t, "inactive", jsonField(updated, "status"))
	assert.Equal(t, "stripe", jsonField(updated, "provider"), "provider is preserved (not updatable via PUT)")
	assert.Equal(t, origCreatedAt, jsonField(updated, "created_at"), "created_at is preserved")
	assertValidTimestamp(t, jsonField(updated, "updated_at"), "updated_at")
	assert.NotEqual(t, origUpdatedAt, jsonField(updated, "updated_at"), "updated_at should advance after update")

	delStatus, delBody, err := apiClient.Delete(covSettingsIntegrationsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// --- Update-idempotency coverage, folded in here to reuse the now-free
	// (SeedAccountID, stripe) slot sequentially. ---
	t.Run("UpdateIdempotent", func(t *testing.T) {
		freshName := uniqueName("e2e-covsi-allf-idemput")
		cStatus, cBody, err := apiClient.Post(covSettingsIntegrationsPath, map[string]any{
			"name":        freshName,
			"provider":    "stripe",
			"credentials": covSettingsIntegrationsStripeCredentials("idemput"),
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, cStatus, cBody)
		freshID := jsonField(parseJSON(cBody), "id")
		require.NotEmpty(t, freshID)
		defer apiClient.Delete(covSettingsIntegrationsPath + "/" + freshID)

		idemKey := newIdempotencyKey()
		putBody := map[string]any{"name": uniqueName("e2e-covsi-allf-idemput-renamed")}

		// apiClient.Put() never sends an Idempotency-Key header — use Do
		// directly to exercise Update's idempotency-key propagation.
		s1, b1, err := apiClient.Do(http.MethodPut, covSettingsIntegrationsPath+"/"+freshID, putBody, idemKey)
		require.NoError(t, err)
		requireStatus(t, 200, s1, b1)

		s2, b2, err := apiClient.Do(http.MethodPut, covSettingsIntegrationsPath+"/"+freshID, putBody, idemKey)
		require.NoError(t, err)
		requireStatus(t, 200, s2, b2)

		assert.JSONEq(t, string(b1), string(b2), "repeated Idempotency-Key on PUT should return an identical response")
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
		id := jsonField(got, "id")
		require.NotEmpty(t, id)
		defer tenantB.Delete(covSettingsIntegrationsPath + "/" + id)

		assertObjectField(t, got, "account_integration")
		assert.Equal(t, name, jsonField(got, "name"))
		assert.Equal(t, "hubspot", jsonField(got, "provider"))
		assert.Equal(t, "active", jsonField(got, "status"), "status should default to active")
		assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	})

	t.Run("CreateMissingName", func(t *testing.T) {
		status, body, err := tenantB.Post(covSettingsIntegrationsPath, map[string]any{
			"provider":    "stripe",
			"credentials": covSettingsIntegrationsStripeCredentials("missingname"),
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "", "invalid_request_error")
		assertErrorParam(t, errObj, "name")
	})

	t.Run("CreateMissingProvider", func(t *testing.T) {
		status, body, err := tenantB.Post(covSettingsIntegrationsPath, map[string]any{
			"name":        uniqueName("e2e-covsi-noprov"),
			"credentials": "{}",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "", "invalid_request_error")
		assertErrorParam(t, errObj, "provider")
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
		require.NotEmpty(t, id)
		defer tenantB.Delete(covSettingsIntegrationsPath + "/" + id)
		origCreatedAt := jsonField(created, "created_at")

		newName := uniqueName("e2e-covsi-pres-upd")
		updStatus, updBody, err := tenantB.Put(covSettingsIntegrationsPath+"/"+id, map[string]any{
			"name": newName,
		})
		require.NoError(t, err)
		requireStatus(t, 200, updStatus, updBody)

		got := parseJSON(updBody)
		assert.Equal(t, newName, jsonField(got, "name"))
		assert.Equal(t, "stripe", jsonField(got, "provider"), "provider preserved")
		assert.Equal(t, "active", jsonField(got, "status"), "status preserved (omitted from PUT body)")
		assert.Equal(t, origCreatedAt, jsonField(got, "created_at"), "created_at should not change")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	})
}

// ──────────────────────────────────────────────
// Response Shape
// ──────────────────────────────────────────────

func TestCovSettingsIntegrations_CreateResponseShape(t *testing.T) {
	t.Parallel()
	tenantB := apiClient.WithBearerToken(SeedTenantBAPIKey, SeedTenantBAccountID)

	name := uniqueName("e2e-covsi-shape")
	status, body, err := tenantB.Post(covSettingsIntegrationsPath, map[string]any{
		"name":        name,
		"provider":    "shippo",
		"credentials": covSettingsIntegrationsShippoCredentials("shape"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	got := parseJSON(body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	defer tenantB.Delete(covSettingsIntegrationsPath + "/" + id)

	assertIDFormat(t, id, "acig")
	assertObjectField(t, got, "account_integration")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
}

// ──────────────────────────────────────────────
// List
// ──────────────────────────────────────────────

func TestCovSettingsIntegrations_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(covSettingsIntegrationsPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 1, "should have at least the seeded shippo/quickbooks rows")
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
	list, _, err := apiClient.GetList(covSettingsIntegrationsPath, url.Values{"q": {"Shippo"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1)

	found := false
	for _, item := range list.Data {
		if DataItemField(item, "id") == SeedAccountIntegrationID {
			found = true
			assert.Equal(t, "Shippo Integration", DataItemField(item, "name"))
			assert.Equal(t, "shippo", DataItemField(item, "provider"))
			assert.Equal(t, "account_integration", DataItemField(item, "object"))
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

func TestCovSettingsIntegrations_ListInvalidCursor(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(covSettingsIntegrationsPath, url.Values{"cursor": {"bogus___not_a_real_cursor"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "", "invalid_request_error")
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

	t.Run("MissingName", func(t *testing.T) {
		t.Parallel()
		status, body, err := apiClient.Post(covSettingsIntegrationsPath, map[string]any{
			"provider":    "stripe",
			"credentials": covSettingsIntegrationsStripeCredentials("val1"),
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "", "invalid_request_error")
		assertErrorParam(t, errObj, "name")
	})

	t.Run("EmptyName", func(t *testing.T) {
		t.Parallel()
		status, body, err := apiClient.Post(covSettingsIntegrationsPath, map[string]any{
			"name":        "",
			"provider":    "stripe",
			"credentials": covSettingsIntegrationsStripeCredentials("val2"),
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "", "invalid_request_error")
		assertErrorParam(t, errObj, "name")
	})

	t.Run("NameTooLong", func(t *testing.T) {
		t.Parallel()
		status, body, err := apiClient.Post(covSettingsIntegrationsPath, map[string]any{
			"name":        strings.Repeat("a", 256),
			"provider":    "stripe",
			"credentials": covSettingsIntegrationsStripeCredentials("val3"),
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "", "invalid_request_error")
		assertErrorParam(t, errObj, "name")
	})

	t.Run("MissingProvider", func(t *testing.T) {
		t.Parallel()
		status, body, err := apiClient.Post(covSettingsIntegrationsPath, map[string]any{
			"name":        uniqueName("e2e-covsi-val"),
			"credentials": "{}",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "", "invalid_request_error")
		assertErrorParam(t, errObj, "provider")
	})

	t.Run("InvalidProviderEnum", func(t *testing.T) {
		t.Parallel()
		status, body, err := apiClient.Post(covSettingsIntegrationsPath, map[string]any{
			"name":        uniqueName("e2e-covsi-val"),
			"provider":    "quickbooks",
			"credentials": "{}",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "", "invalid_request_error")
		assertErrorParam(t, errObj, "provider")
	})

	t.Run("MissingCredentials", func(t *testing.T) {
		t.Parallel()
		status, body, err := apiClient.Post(covSettingsIntegrationsPath, map[string]any{
			"name":     uniqueName("e2e-covsi-val"),
			"provider": "stripe",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "", "invalid_request_error")
		assertErrorParam(t, errObj, "credentials")
	})

	t.Run("MalformedCredentialsJSON", func(t *testing.T) {
		t.Parallel()
		status, body, err := apiClient.Post(covSettingsIntegrationsPath, map[string]any{
			"name":        uniqueName("e2e-covsi-val"),
			"provider":    "stripe",
			"credentials": "not-json",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "", "invalid_request_error")
		assertErrorParam(t, errObj, "credentials")
	})

	t.Run("StripeMissingKeyPrefix", func(t *testing.T) {
		t.Parallel()
		status, body, err := apiClient.Post(covSettingsIntegrationsPath, map[string]any{
			"name":        uniqueName("e2e-covsi-val"),
			"provider":    "stripe",
			"credentials": `{"private_key":"abc","publishable_key":"def","webhook_secret":"ghi"}`,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "", "invalid_request_error")
		assertErrorParam(t, errObj, "credentials")
	})

	t.Run("StripeSandboxKeysOnProductionAccount", func(t *testing.T) {
		t.Parallel()
		status, body, err := apiClient.Post(covSettingsIntegrationsPath, map[string]any{
			"name":        uniqueName("e2e-covsi-val"),
			"provider":    "stripe",
			"credentials": `{"private_key":"sk_test_x","publishable_key":"pk_test_x","webhook_secret":"whsec_x"}`,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "", "invalid_request_error")
		assertErrorParam(t, errObj, "credentials")
	})

	t.Run("ShippoMissingKeyPrefix", func(t *testing.T) {
		t.Parallel()
		status, body, err := apiClient.Post(covSettingsIntegrationsPath, map[string]any{
			"name":        uniqueName("e2e-covsi-val"),
			"provider":    "shippo",
			"credentials": `{"api_key":"abc"}`,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "", "invalid_request_error")
		assertErrorParam(t, errObj, "credentials")
	})

	t.Run("ShippoSandboxKeyOnProductionAccount", func(t *testing.T) {
		t.Parallel()
		status, body, err := apiClient.Post(covSettingsIntegrationsPath, map[string]any{
			"name":        uniqueName("e2e-covsi-val"),
			"provider":    "shippo",
			"credentials": `{"api_key":"shippo_test_x"}`,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "", "invalid_request_error")
		assertErrorParam(t, errObj, "credentials")
	})

	t.Run("HubspotMissingTokenPrefix", func(t *testing.T) {
		t.Parallel()
		status, body, err := apiClient.Post(covSettingsIntegrationsPath, map[string]any{
			"name":        uniqueName("e2e-covsi-val"),
			"provider":    "hubspot",
			"credentials": `{"access_token":"abc"}`,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "", "invalid_request_error")
		assertErrorParam(t, errObj, "credentials")
	})

	t.Run("MalformedJSONUnknownFieldRejected", func(t *testing.T) {
		t.Parallel()
		status, body, err := apiClient.Post(covSettingsIntegrationsPath, map[string]any{
			bogusE2EJSONField: "1",
			"name":            uniqueName("e2e-covsi-val"),
			"provider":        "stripe",
			"credentials":     covSettingsIntegrationsStripeCredentials("valunk"),
		}, newIdempotencyKey())
		require.NoError(t, err)
		assertJSONUnknownFieldRejected(t, "POST", covSettingsIntegrationsPath, status, body)
	})
}

// ──────────────────────────────────────────────
// Update Validation / Not Found / Cross-Tenant
// ──────────────────────────────────────────────

func TestCovSettingsIntegrations_UpdateValidation(t *testing.T) {
	t.Parallel()

	// These target the seeded, read-only SeedAccountIntegrationID row.
	// Confirmed live: a 400-rejected PUT never mutates the stored row (name
	// and updated_at are unchanged after each of these calls), so it is safe
	// to reuse this shared row across parallel validation-only subtests
	// without a create/delete lifecycle of its own.

	t.Run("NullNameRejected", func(t *testing.T) {
		t.Parallel()
		status, body, err := apiClient.Put(covSettingsIntegrationsPath+"/"+SeedAccountIntegrationID, map[string]any{
			"name": nil,
		})
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "", "invalid_request_error")
		assertErrorParam(t, errObj, "name")
	})

	t.Run("BlankNameRejected", func(t *testing.T) {
		t.Parallel()
		status, body, err := apiClient.Put(covSettingsIntegrationsPath+"/"+SeedAccountIntegrationID, map[string]any{
			"name": "",
		})
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "", "invalid_request_error")
		assertErrorParam(t, errObj, "name")
	})

	t.Run("NullStatusRejected", func(t *testing.T) {
		t.Parallel()
		status, body, err := apiClient.Put(covSettingsIntegrationsPath+"/"+SeedAccountIntegrationID, map[string]any{
			"status": nil,
		})
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "", "invalid_request_error")
		assertErrorParam(t, errObj, "status")
	})

	// Regression-pin: `status` is validated against the
	// AccountIntegrationStatus enum (active/inactive), mirroring the enum
	// validation already proven for `provider` on create. Verified live
	// against the running stack that a garbage value is rejected with 400
	// rather than silently coerced to `inactive`.
	t.Run("BogusStatusEnumRejected", func(t *testing.T) {
		t.Parallel()
		status, body, err := apiClient.Put(covSettingsIntegrationsPath+"/"+SeedAccountIntegrationID, map[string]any{
			"status": "bogus",
		})
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "", "invalid_request_error")
		assertErrorParam(t, errObj, "status")
	})

	t.Run("UpdateNonexistentIDReturns404", func(t *testing.T) {
		t.Parallel()
		status, body, err := apiClient.Put(covSettingsIntegrationsPath+"/acig_doesnotexist00000000000", map[string]any{
			"name": uniqueName("e2e-covsi-404"),
		})
		require.NoError(t, err)
		requireStatus(t, 404, status, body)
	})

	t.Run("DeleteNonexistentIDReturns404", func(t *testing.T) {
		t.Parallel()
		status, body, err := apiClient.Delete(covSettingsIntegrationsPath + "/acig_doesnotexist00000000000")
		require.NoError(t, err)
		requireStatus(t, 404, status, body)
	})

	t.Run("CrossTenantUpdateReturns404", func(t *testing.T) {
		t.Parallel()
		tenantB := apiClient.WithBearerToken(SeedTenantBAPIKey, SeedTenantBAccountID)
		status, body, err := tenantB.Put(covSettingsIntegrationsPath+"/"+SeedAccountIntegrationID, map[string]any{
			"name": uniqueName("e2e-covsi-crosstenant"),
		})
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
