//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// State transition and business logic tests verify that the API enforces
// business rules beyond basic CRUD: status transitions, revocation behavior,
// workflow constraints, etc.

// ──────────────────────────────────────────────
// Customer status transitions
// ──────────────────────────────────────────────

func TestStateTransition_CustomerStatus_NormalToHoldShipment(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-state-cust")))
	id := jsonField(created, "id")
	assert.Equal(t, "normal", jsonField(created, "status"))

	// Transition to hold_shipment.
	status, body, err := apiClient.Patch(customersPath+"/"+id, map[string]any{
		"status": "hold_shipment",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "hold_shipment", jsonField(parseJSON(body), "status"))
}

func TestStateTransition_CustomerStatus_HoldShipmentBackToNormal(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-state-unsus")))
	id := jsonField(created, "id")

	// Hold.
	s1, b1, err := apiClient.Patch(customersPath+"/"+id, map[string]any{
		"status": "hold_shipment",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, s1, b1)

	// Resume.
	s2, b2, err := apiClient.Patch(customersPath+"/"+id, map[string]any{
		"status": "normal",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, s2, b2)
	assert.Equal(t, "normal", jsonField(parseJSON(b2), "status"))
}

func TestStateTransition_CustomerStatus_InvalidTransition(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-state-inv")))
	id := jsonField(created, "id")

	status, body, err := apiClient.Patch(customersPath+"/"+id, map[string]any{
		"status": "totally_fake_status",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Invalid status transition should return 400 or 422, got %d: %s", status, string(body))
}

// ──────────────────────────────────────────────
// API key revocation behavior
// ──────────────────────────────────────────────

func TestStateTransition_RevokedAPIKey_CannotAuthenticate(t *testing.T) {
	t.Parallel()

	// Create a new API key.
	createStatus, createBody, err := apiClient.Post(apiKeysPath, map[string]any{
		"name":    uniqueName("e2e-state-revoke"),
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	parsed := parseJSON(createBody)
	secret := jsonField(parsed, "api_key_secret")
	info := jsonObject(parsed, "api_key_info")
	id := jsonField(info, "id")

	// Create a client using the new key.
	baseURL := envOr("E2E_BASE_URL", defaultBaseURL)
	keyClient := NewClient(baseURL, secret, SeedAccountID)

	// Verify it works.
	verifyStatus, _, err := keyClient.GetListRaw(apiKeysPath, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, verifyStatus, "New API key should work before revocation")

	// Revoke it.
	delStatus, delBody, err := apiClient.Delete(apiKeysPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Verify revoked key is rejected.
	revokedStatus, _, err := keyClient.GetListRaw(apiKeysPath, nil)
	require.NoError(t, err)
	assert.Equal(t, 401, revokedStatus, "Revoked API key should return 401")
}

func TestStateTransition_RevokedAPIKey_StillVisibleInList(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-state-revlist")
	createStatus, createBody, err := apiClient.Post(apiKeysPath, map[string]any{
		"name":    name,
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	id := jsonField(jsonObject(parseJSON(createBody), "api_key_info"), "id")

	// Revoke.
	apiClient.Delete(apiKeysPath + "/" + id)

	// Should still be visible via GET.
	getStatus, getBody, err := apiClient.GetListRaw(apiKeysPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.NotNil(t, parseJSON(getBody)["revoked_at"], "Revoked key should have revoked_at set")
}

// ──────────────────────────────────────────────
// API key rotation workflow
// ──────────────────────────────────────────────

func TestStateTransition_RotatedKey_OldKeyRevoked(t *testing.T) {
	t.Parallel()

	createStatus, createBody, err := apiClient.Post(apiKeysPath, map[string]any{
		"name":    uniqueName("e2e-state-rot"),
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	origParsed := parseJSON(createBody)
	origSecret := jsonField(origParsed, "api_key_secret")
	origID := jsonField(jsonObject(origParsed, "api_key_info"), "id")

	// Rotate.
	rotStatus, rotBody, err := apiClient.Post(apiKeysPath+"/"+origID+"/actions/rotate", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, rotStatus, rotBody)

	rotParsed := parseJSON(rotBody)
	newSecret := jsonField(rotParsed, "api_key_secret")
	newID := jsonField(jsonObject(rotParsed, "api_key_info"), "id")
	t.Cleanup(func() { apiClient.Delete(apiKeysPath + "/" + newID) })

	assert.NotEqual(t, origSecret, newSecret, "Rotated key should have a different secret")
	assert.NotEqual(t, origID, newID, "Rotated key should have a new ID")

	// Old key should be revoked.
	baseURL := envOr("E2E_BASE_URL", defaultBaseURL)
	oldClient := NewClient(baseURL, origSecret, SeedAccountID)
	oldStatus, _, err := oldClient.GetListRaw(apiKeysPath, nil)
	require.NoError(t, err)
	assert.Equal(t, 401, oldStatus, "Old rotated key should return 401")

	// New key should work.
	newClient := NewClient(baseURL, newSecret, SeedAccountID)
	newStatus, _, err := newClient.GetListRaw(apiKeysPath, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, newStatus, "New rotated key should work")
}

// ──────────────────────────────────────────────
// Customer update preserves non-updated fields
// ──────────────────────────────────────────────

func TestStateTransition_PartialUpdatePreservesFields(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-state-pres")
	presPayload := validCustomerBody(name)
	presPayload["note"] = "original note"
	presPayload["commission_policy"] = "commission_exempt"
	created := createAndCleanup(t, customersPath, presPayload)
	id := jsonField(created, "id")
	originalNumber := jsonField(created, "number")

	// Update only the note.
	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+id, map[string]any{
		"note": "updated note",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, name, jsonField(updated, "name"), "name should be preserved")
	assert.Equal(t, originalNumber, jsonField(updated, "number"), "number should be preserved")
	assert.Equal(t, "commission_exempt", jsonField(updated, "commission_policy"), "commission_policy should be preserved")
	assert.Equal(t, "updated note", jsonField(updated, "note"), "note should be updated")
}

// ──────────────────────────────────────────────
// Account group type constraints
// ──────────────────────────────────────────────

func TestStateTransition_AccountGroupType_FilterByType(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, "/v1/sales/account-groups", map[string]any{
		"name": uniqueName("e2e-state-grp"),
		"type": "type_group",
	})
	id := jsonField(created, "id")

	// Verify it appears when filtering by type_group.
	list, _, err := apiClient.GetList("/v1/sales/account-groups", url.Values{"type": {"type_group"}})
	require.NoError(t, err)

	found := false
	for _, item := range list.Data {
		if DataItemField(item, "id") == id {
			found = true
			break
		}
	}
	assert.True(t, found, "Account group should appear when filtering by its type")
}

// ──────────────────────────────────────────────
// Immutable fields
// ──────────────────────────────────────────────

func TestStateTransition_IDIsImmutable(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-state-immid")))
	originalID := jsonField(created, "id")

	// Update and verify ID doesn't change.
	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+originalID, map[string]any{
		"note": "test immutability",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	assert.Equal(t, originalID, jsonField(parseJSON(patchBody), "id"), "ID must be immutable across updates")
}

func TestStateTransition_NumberIsImmutable(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-state-immnum")))
	id := jsonField(created, "id")
	originalNumber := jsonField(created, "number")

	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+id, map[string]any{
		"note": "test number immutability",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	assert.Equal(t, originalNumber, jsonField(parseJSON(patchBody), "number"), "number must be immutable across updates")
}
