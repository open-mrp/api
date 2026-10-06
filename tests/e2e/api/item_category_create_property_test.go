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

// Creating a property from a category is one request under item_categories:update: the new property is created in the account and attached to the category together.

func categoryPropertiesPath(categoryID string) string {
	return itemCategoriesPath + "/" + categoryID + "/properties"
}

// trackCategoryProperty detaches and deletes a property the test created on categoryID once the test ends.
func trackCategoryProperty(t *testing.T, categoryID, propertyID string) {
	t.Helper()
	t.Cleanup(func() {
		_, _, _ = apiClient.Delete(categoryPropertiesPath(categoryID) + "/" + propertyID)
		_, _, _ = apiClient.Delete(propertiesPath + "/" + propertyID)
	})
}

func categoryPropertyIDs(t *testing.T, categoryID string) []string {
	t.Helper()
	status, body, err := apiClient.GetListRaw(itemCategoriesPath+"/"+categoryID, url.Values{"include": {"properties"}})
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)
	var ids []string
	for _, raw := range jsonListData(parseJSON(body), "properties") {
		ids = append(ids, jsonField(raw.(map[string]any), "id"))
	}
	return ids
}

func TestItemCategoryProperties_CreateUnderItemCategoriesUpdateAlone(t *testing.T) {
	t.Parallel()
	categoryID := productCategoryInEaches(t)
	editor := customRoleClient(t, "item_categories:update")
	name := uniqueName("e2e-catprop")
	key := newIdempotencyKey()

	resp, err := editor.PostFull(withQuery(categoryPropertiesPath(categoryID), url.Values{"include": {"attributes"}}), map[string]any{"name": name}, key)
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, resp.StatusCode, resp.Body)
	created := parseJSON(resp.Body)
	propertyID := jsonField(created, "id")
	trackCategoryProperty(t, categoryID, propertyID)

	assertIDFormat(t, propertyID, "pp")
	assert.Equal(t, "property", jsonField(created, "object"))
	assert.Equal(t, name, jsonField(created, "name"))
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")
	attributes := jsonObject(created, "attributes")
	require.NotNil(t, attributes, "attributes are included on request")
	assert.Empty(t, jsonArray(attributes, "data"), "a new property has no attributes yet")
	assert.Equal(t, "/v1/catalog/properties/"+propertyID, resp.Header.Get("Location"))

	replay, err := editor.PostFull(withQuery(categoryPropertiesPath(categoryID), url.Values{"include": {"attributes"}}), map[string]any{"name": name}, key)
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, replay.StatusCode, replay.Body)
	assert.Equal(t, propertyID, jsonField(parseJSON(replay.Body), "id"), "a replay answers with the property the first request created")

	assert.Equal(t, []string{propertyID}, categoryPropertyIDs(t, categoryID), "the category carries the new property, once")
	status, body, err := apiClient.GetListRaw(propertiesPath+"/"+propertyID, nil)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)
	assert.Equal(t, name, jsonField(parseJSON(body), "name"), "the property is one of the account's properties")

	expectAuditEvent(t, propertyID, "property", "create")
	event := expectAuditEventWithChanges(t, categoryID, "item_category", "update")
	_, ok := changeForField(jsonListData(event, "changes"), "properties")
	assert.True(t, ok, "the category's update event records the property it gained")
}

func TestItemCategoryProperties_CreateRefusals(t *testing.T) {
	t.Parallel()
	categoryID := productCategoryInEaches(t)
	foreignCategoryID := dashItemsTenantBRefs(t).categoryID

	status, body, err := apiClient.GetListRaw(propertiesPath+"/"+SeedPropertyID, nil)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)
	takenName := jsonField(parseJSON(body), "name")

	cases := []struct {
		name       string
		client     *Client
		categoryID string
		body       map[string]any
		check      func(t *testing.T, status int, body []byte)
	}{
		{"a role holding properties:create alone", customRoleClient(t, "properties:create"), categoryID, map[string]any{"name": uniqueName("e2e-catprop-ref")},
			func(t *testing.T, status int, body []byte) {
				requirePermissionRefused(t, status, body, "item_categories:update")
			}},
		{"a name the account already uses", apiClient, categoryID, map[string]any{"name": takenName},
			func(t *testing.T, status int, body []byte) {
				requireStatus(t, http.StatusConflict, status, body)
				assertErrorParam(t, requireErrorResponse(t, body, "resource_conflict", "invalid_request_error"), "name")
			}},
		{"no name", apiClient, categoryID, map[string]any{},
			func(t *testing.T, status int, body []byte) {
				requireStatus(t, http.StatusBadRequest, status, body)
				assertErrorParam(t, parseJSON(body)["error"].(map[string]any), "name")
			}},
		{"a name longer than 255 characters", apiClient, categoryID, map[string]any{"name": strings.Repeat("n", 256)},
			func(t *testing.T, status int, body []byte) {
				requireStatus(t, http.StatusBadRequest, status, body)
				assertErrorParam(t, parseJSON(body)["error"].(map[string]any), "name")
			}},
		{"another tenant's category", apiClient, foreignCategoryID, map[string]any{"name": uniqueName("e2e-catprop-ref")},
			func(t *testing.T, status int, body []byte) {
				requireStatus(t, http.StatusNotFound, status, body)
				requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
			}},
		{"a default category", apiClient, "shipping", map[string]any{"name": uniqueName("e2e-catprop-ref")},
			func(t *testing.T, status int, body []byte) {
				requireStatus(t, http.StatusForbidden, status, body)
				requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
			}},
	}
	for _, tc := range cases {
		status, body, err := tc.client.Post(categoryPropertiesPath(tc.categoryID), tc.body, newIdempotencyKey())
		require.NoError(t, err)
		if status == http.StatusCreated {
			trackCategoryProperty(t, tc.categoryID, jsonField(parseJSON(body), "id"))
		}
		t.Run(tc.name, func(t *testing.T) { tc.check(t, status, body) })
	}

	assert.Empty(t, categoryPropertyIDs(t, categoryID), "no refused request attached a property")
}
