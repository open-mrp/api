//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests pin the client to 1.0.forge-preview.1 to exercise the version
// transformer: account_user responses must be downgraded to the old shape
// (profile fields hoisted onto the account_user, user as an entity reference)
// even though the backend now serves the 1.0.forge-preview.2 shape.

const previousAPIVersion = "1.0.forge-preview.1"

func assertPreview1AccountUserShape(t *testing.T, m map[string]any) {
	t.Helper()

	assert.Equal(t, "account_user", jsonField(m, "object"))
	assert.NotEmpty(t, jsonField(m, "id"))
	assert.NotEmpty(t, jsonField(m, "status"))

	// Profile fields are hoisted back onto the account_user.
	for _, key := range []string{"name", "email", "username", "image_url"} {
		_, present := m[key]
		assert.True(t, present, "%s must be present on preview.1 account_user responses", key)
	}

	// user is downgraded to a polymorphic entity reference.
	user := jsonObject(m, "user")
	require.NotNil(t, user, "preview.1 account_user must expose a user entity reference")
	assert.Equal(t, "entity", jsonField(user, "object"))
	assert.Equal(t, "user", jsonField(user, "type"))
	userID := jsonField(user, "id")
	assert.NotEmpty(t, userID, "user entity reference must have an id")
	assert.NotEqual(t, jsonField(m, "id"), userID, "user.id must differ from the account_user id")
}

func TestVersionCompat_AccountUsers_Get(t *testing.T) {
	t.Parallel()
	oldClient := apiClient.WithAPIVersion(previousAPIVersion)

	status, body, err := oldClient.GetListRaw(accountUsersPath+"/"+SeedAccountUserID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assertPreview1AccountUserShape(t, got)
	assert.NotEmpty(t, jsonField(got, "name"), "seeded account user should have a hoisted name")
	assert.NotEmpty(t, jsonField(got, "email"), "seeded account user should have a hoisted email")
	assert.Equal(t, jsonField(got, "name"), jsonField(jsonObject(got, "user"), "name"),
		"entity name should mirror the hoisted name")
	assert.Equal(t, jsonField(got, "email"), jsonField(jsonObject(got, "user"), "handle"),
		"entity handle should mirror the hoisted email")
}

func TestVersionCompat_AccountUsers_List(t *testing.T) {
	t.Parallel()
	oldClient := apiClient.WithAPIVersion(previousAPIVersion)

	list, _, err := oldClient.GetList(accountUsersPath, nil)
	require.NoError(t, err)
	require.NotEmpty(t, list.Data)

	for _, item := range list.Data {
		m := parseJSON(item)
		require.NotNil(t, m)
		assertPreview1AccountUserShape(t, m)
	}
}

func TestVersionCompat_AccountUsers_CreateAndPatch(t *testing.T) {
	t.Parallel()
	oldClient := apiClient.WithAPIVersion(previousAPIVersion)

	name := uniqueName("e2e-vc-acuser")
	email := name + "@e2e-test.augno.com"

	createStatus, createBody, err := oldClient.Post(accountUsersPath, map[string]any{
		"name":    name,
		"email":   email,
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	created := parseJSON(createBody)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	defer removeAccountUser(id)

	assertPreview1AccountUserShape(t, created)
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, email, jsonField(created, "email"))

	newName := uniqueName("e2e-vc-acuser-u")
	patchStatus, patchBody, err := oldClient.Patch(accountUsersPath+"/"+id, map[string]any{
		"name": newName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	assertPreview1AccountUserShape(t, patched)
	assert.Equal(t, newName, jsonField(patched, "name"))
}

func TestVersionCompat_AccountUsers_LatestVersionUnaffected(t *testing.T) {
	t.Parallel()
	// The default client is on the latest version: no hoisted profile fields,
	// user null unless included.
	status, body, err := apiClient.GetListRaw(accountUsersPath+"/"+SeedAccountUserID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	for _, key := range []string{"name", "email", "username", "image_url"} {
		_, present := got[key]
		assert.False(t, present, "%s must not be present on latest-version account_user responses", key)
	}
	assert.Nil(t, got["user"], "user should be null without ?include=user on the latest version")
}
