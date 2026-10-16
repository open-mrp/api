package versiontransforms

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/api/shared/constants"
)

func TestPreview9_AccountUserShape(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeAccountUser, `{"id":"acus_1","object":"account_user","status":"active",
		"portal_access":"invited","work_center":{"id":"dp_1","object":"work_center"},"commission_eligibility":"eligible"}`)
	assert.Equal(t, "dp_1", got["department"].(map[string]any)["id"])
	assert.Equal(t, true, got["is_commission_eligible"])
	for _, gone := range []string{"work_center", "commission_eligibility", "portal_access"} {
		assert.NotContains(t, got, gone)
	}

	portal := downgrade(t, constants.ObjectTypeAccountUser, `{"id":"acus_2","object":"account_user","commission_eligibility":null,"work_center":null}`)
	assert.Contains(t, portal, "is_commission_eligible")
	assert.Nil(t, portal["is_commission_eligible"], "a portal user's hidden eligibility stays null")
	assert.Contains(t, portal, "department")

	nested := downgrade(t, constants.ObjectTypeCustomer, `{"id":"cu_1","object":"customer","defaults":{"sales_rep":{"id":"acus_3","object":"account_user","commission_eligibility":"ineligible"}}}`)
	assert.Equal(t, false, nested["defaults"].(map[string]any)["sales_rep"].(map[string]any)["is_commission_eligible"])
}

func TestPreview9_AccountUserWrites(t *testing.T) {
	t.Parallel()

	created := upgradeBody(constants.ObjectTypeAccountUser, http.MethodPost, accountUsersRoute, map[string]any{
		"email": "a@b.test", "department_id": "dp_1", "is_commission_eligible": true,
	})
	assert.Equal(t, map[string]any{"email": "a@b.test", "work_center_id": "dp_1", "commission_eligibility": "eligible"}, created)

	updated := upgradeBody(constants.ObjectTypeAccountUser, http.MethodPatch, accountUserRoute, map[string]any{
		"department_id": nil, "is_commission_eligible": false,
	})
	assert.Equal(t, map[string]any{"work_center_id": nil, "commission_eligibility": "ineligible"}, updated)
}

func TestRoutePreview9Identity(t *testing.T) {
	t.Parallel()

	var seen seenRequest
	h := RoutePreview9Identity(recordRequest(&seen))

	h.ServeHTTP(nil, versionedRequest(http.MethodPut, "/v1/identity/account-users/acus_1/actions/disable", "", apiversion.V1_0_Forge_Preview9))
	assert.Equal(t, http.MethodPost, seen.method)

	h.ServeHTTP(nil, versionedRequest(http.MethodPut, "/v1/identity/account-users/acus_1/actions/disable", "", apiversion.V1_0_Forge))
	assert.Equal(t, http.MethodPut, seen.method, "forge callers are not rerouted")

	h.ServeHTTP(nil, versionedRequest(http.MethodGet, "/v1/identity/account-users?is_commission_eligible=true", "", apiversion.V1_0_Forge_Preview9))
	q, err := url.ParseQuery(seen.query)
	require.NoError(t, err)
	assert.Equal(t, []string{"active", "disabled"}, q["statuses[]"], "removed members stay hidden by default")
	assert.Equal(t, "eligible", q.Get("commission_eligibility"))
	assert.False(t, q.Has("is_commission_eligible"))

	h.ServeHTTP(nil, versionedRequest(http.MethodGet, "/v1/identity/account-users?removed_scope=included", "", apiversion.V1_0_Forge_Preview9))
	q, err = url.ParseQuery(seen.query)
	require.NoError(t, err)
	assert.False(t, q.Has("statuses[]"), "included lists every status")
	assert.False(t, q.Has("removed_scope"))

	h.ServeHTTP(nil, versionedRequest(http.MethodPost, "/v1/identity/account-users", "{}", apiversion.V1_0_Forge_Preview9))
	assert.Empty(t, seen.query, "create is not given list filters")
}

func TestPreview9_RoleAndAPIKeyShapes(t *testing.T) {
	t.Parallel()

	role := downgrade(t, constants.ObjectTypeRole, `{"id":"rl_1","object":"role","name":"R","type":"user","template":null,
		"access":{"sales":"read"},"grants":[],"revokes":[],"permissions":["sales_orders:read"]}`)
	for _, gone := range []string{"template", "access", "grants", "revokes"} {
		assert.NotContains(t, role, gone)
	}
	assert.Equal(t, []any{"sales_orders:read"}, role["permissions"])

	created := downgrade(t, constants.ObjectTypeCreatedAPIKey, `{"object":"created_api_key","secret":"mrp_sk_test_x",
		"api_key":{"id":"apke_1","object":"api_key","status":"active","client":null}}`)
	assert.Equal(t, "mrp_sk_test_x", created["api_key_secret"])
	info := created["api_key_info"].(map[string]any)
	assert.Equal(t, "apke_1", info["id"])
	assert.NotContains(t, info, "status")
	assert.NotContains(t, info, "client")
	assert.NotContains(t, created, "secret")

	body := upgradeBody(constants.ObjectTypeRole, http.MethodPost, rolesRoute, map[string]any{"name": "Empty"})
	assert.Equal(t, []any{}, body["permissions"], "a preview.9 role created without permissions stays empty")
	kept := upgradeBody(constants.ObjectTypeRole, http.MethodPost, rolesRoute, map[string]any{"name": "R", "permissions": []any{"items:read"}})
	assert.Equal(t, []any{"items:read"}, kept["permissions"])
}
