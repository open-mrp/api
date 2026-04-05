//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const accountGroupsPath = "/v1/sales/account-groups"

func TestAccountGroups_CRUD(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-acgrp")
	createStatus, createBody, err := apiClient.Post(accountGroupsPath, map[string]any{
		"name": name,
		"type": "type_group",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	created := parseJSON(createBody)
	assert.Equal(t, "account_group", jsonField(created, "object"))
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, "type_group", jsonField(created, "type"))
	assert.Equal(t, "commission_exempt", jsonField(created, "commission_policy"))
	assert.Equal(t, "billed_freight", jsonField(created, "freight_policy"))
	assertNilField(t, created, "description")
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")

	getStatus, getBody, err := apiClient.GetListRaw(accountGroupsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, name, jsonField(parseJSON(getBody), "name"))

	newName := uniqueName("e2e-acgrp-upd")
	patchStatus, patchBody, err := apiClient.Patch(accountGroupsPath+"/"+id, map[string]any{
		"name":              newName,
		"commission_policy": "commission_applied",
		"freight_policy":    "free_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	updated := parseJSON(patchBody)
	assert.Equal(t, newName, jsonField(updated, "name"))
	assert.Equal(t, "commission_applied", jsonField(updated, "commission_policy"))
	assert.Equal(t, "free_freight", jsonField(updated, "freight_policy"))
	assert.Equal(t, "type_group", jsonField(updated, "type"), "type should be preserved")
	assertValidTimestamp(t, jsonField(updated, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(updated, "updated_at"), "updated_at")

	delStatus, delBody, err := apiClient.Delete(accountGroupsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	getStatus2, _, err := apiClient.GetListRaw(accountGroupsPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus2)
}

func TestAccountGroups_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()

	// ── CREATE with all fields ──
	name := uniqueName("e2e-acgrp-allf")
	createStatus, createBody, err := apiClient.Post(accountGroupsPath, map[string]any{
		"name":              name,
		"type":              "type_group",
		"description":       "Test description",
		"commission_policy": "commission_applied",
		"freight_policy":    "free_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	got := parseJSON(createBody)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(accountGroupsPath + "/" + id)

	assert.Equal(t, "account_group", jsonField(got, "object"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, "type_group", jsonField(got, "type"))
	assert.Equal(t, "Test description", jsonField(got, "description"))
	assert.Equal(t, "commission_applied", jsonField(got, "commission_policy"))
	assert.Equal(t, "free_freight", jsonField(got, "freight_policy"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))

	// ── UPDATE with different values ──
	updatedName := uniqueName("e2e-acgrp-allf-u")
	patchStatus, patchBody, err := apiClient.Patch(accountGroupsPath+"/"+id, map[string]any{
		"name":              updatedName,
		"description":       "Updated description",
		"commission_policy": "commission_exempt",
		"freight_policy":    "billed_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, id, jsonField(updated, "id"), "ID must not change")
	assert.Equal(t, updatedName, jsonField(updated, "name"))
	assert.Equal(t, "Updated description", jsonField(updated, "description"))
	assert.Equal(t, "commission_exempt", jsonField(updated, "commission_policy"))
	assert.Equal(t, "billed_freight", jsonField(updated, "freight_policy"))
	assert.Equal(t, "type_group", jsonField(updated, "type"), "type should be preserved")
}

func TestAccountGroups_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(accountGroupsPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 1)

	found := false
	for _, item := range list.Data {
		if DataItemField(item, "name") == SeedCustomerGroupName {
			found = true
			break
		}
	}
	assert.True(t, found, "Seeded account group %q should appear in list", SeedCustomerGroupName)
}

func TestAccountGroups_ListSearchByName(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(accountGroupsPath, url.Values{"q": {"DME"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Search for 'DME' should return at least 1 result")
}

func TestAccountGroups_ListSearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(accountGroupsPath, url.Values{"q": {"zzzznotagroup99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

func TestAccountGroups_FilterByType(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(accountGroupsPath, url.Values{"type": {"type_group"}})
	require.NoError(t, err)
	assert.NotEmpty(t, list.Data)
}

func TestAccountGroups_Idempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-idem-acgrp")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Post(accountGroupsPath, map[string]any{"name": name, "type": "type_group"}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(accountGroupsPath, map[string]any{"name": name, "type": "type_group"}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"))

	apiClient.Delete(accountGroupsPath + "/" + id1)
}

// ──────────────────────────────────────────────
// Omitted Fields
// ──────────────────────────────────────────────

func TestAccountGroups_OmittedFields(t *testing.T) {
	t.Parallel()

	t.Run("CreateWithOnlyRequiredFields", func(t *testing.T) {
		name := uniqueName("e2e-acgrp-omit")
		status, body, err := apiClient.Post(accountGroupsPath, map[string]any{
			"name": name,
			"type": "type_group",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)

		got := parseJSON(body)
		id := jsonField(got, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(accountGroupsPath + "/" + id)

		assertObjectField(t, got, "account_group")
		assert.Equal(t, name, jsonField(got, "name"))
		assert.Equal(t, "type_group", jsonField(got, "type"))
		assertNilField(t, got, "description")
		assert.Equal(t, "commission_exempt", jsonField(got, "commission_policy"))
		assert.Equal(t, "billed_freight", jsonField(got, "freight_policy"))
		assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	})

	t.Run("UpdatePreservesOmittedFields", func(t *testing.T) {
		name := uniqueName("e2e-acgrp-pres")
		createStatus, createBody, err := apiClient.Post(accountGroupsPath, map[string]any{
			"name":              name,
			"type":              "type_group",
			"description":       "Original desc",
			"commission_policy": "commission_applied",
			"freight_policy":    "free_freight",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, createStatus, createBody)

		created := parseJSON(createBody)
		id := jsonField(created, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(accountGroupsPath + "/" + id)
		origCreatedAt := jsonField(created, "created_at")

		// Update ONLY name
		newName := uniqueName("e2e-acgrp-pres-u")
		patchStatus, patchBody, err := apiClient.Patch(accountGroupsPath+"/"+id, map[string]any{
			"name": newName,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, patchStatus, patchBody)

		got := parseJSON(patchBody)
		assert.Equal(t, newName, jsonField(got, "name"))
		assert.Equal(t, "type_group", jsonField(got, "type"), "type should be preserved")
		assert.Equal(t, "Original desc", jsonField(got, "description"), "description should be preserved")
		assert.Equal(t, "commission_applied", jsonField(got, "commission_policy"), "commission_policy should be preserved")
		assert.Equal(t, "free_freight", jsonField(got, "freight_policy"), "freight_policy should be preserved")
		assert.Equal(t, origCreatedAt, jsonField(got, "created_at"), "created_at should not change")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	})
}

func TestAccountGroups_CreateValidation_EmptyName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(accountGroupsPath, map[string]any{
		"name": "",
		"type": "type_group",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Empty name should return 400 or 422, got %d: %s", status, string(body))
}
