//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/shared/id"
)

const (
	permissionsCatalogPath = "/v1/identity/permissions"
	roleTemplatesPath      = "/v1/identity/role-templates"
	rolePreviewPath        = "/v1/identity/roles/actions/preview"
	forgeAPIKeysPath       = "/v1/auth/api-keys"
)

// createTeamMember adds a member to your own team and removes them when the test ends.
func createTeamMember(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	name := uniqueName("e2e-idn")
	if _, ok := body["name"]; !ok {
		body["name"] = name
	}
	if _, ok := body["email"]; !ok {
		body["email"] = name + "@e2e-test.openmrp.ai"
	}
	status, raw, err := apiClient.Post(accountUsersPath+"?include[]=work_center&include[]=role", body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	got := parseJSON(raw)
	accountUserID := jsonField(got, "id")
	t.Cleanup(func() { removeAccountUser(accountUserID) })
	return got
}

func postAction(t *testing.T, c *Client, path string, want int) map[string]any {
	t.Helper()
	status, raw, err := c.Post(path, nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, want, status, raw)
	return parseJSON(raw)
}

// --- Account users ---

func TestForgeAccountUsers_CreateAllFields(t *testing.T) {
	t.Parallel()
	markOutbox(t)
	name := uniqueName("e2e-idn-all")
	email := name + "@e2e-test.openmrp.ai"
	got := createTeamMember(t, map[string]any{
		"name": name, "email": email, "role_id": SeedSalesRepRoleID, "work_center_id": SeedDepartmentID,
		"commission_eligibility": "eligible",
	})

	assertIDFormat(t, jsonField(got, "id"), id.AccountUserIDPrefix)
	assertObjectField(t, got, "account_user")
	assert.Equal(t, "active", jsonField(got, "status"))
	assert.Equal(t, "invited", jsonField(got, "portal_access"), "your own team members are invited when added")
	assert.Equal(t, "eligible", jsonField(got, "commission_eligibility"))
	assert.Equal(t, SeedDepartmentID, jsonField(jsonObject(got, "work_center"), "id"))
	assert.Equal(t, "work_center", jsonField(jsonObject(got, "work_center"), "object"))
	assert.Equal(t, SeedSalesRepRoleID, jsonField(jsonObject(got, "role"), "id"))
	assertNilField(t, got, "user")
	assertNilField(t, got, "last_used_at")
	assertNilField(t, got, "notification_types")
	for _, gone := range []string{"department", "is_commission_eligible"} {
		assert.NotContains(t, got, gone)
	}
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	welcome := queuedSendEmail(t, email)
	assert.True(t, strings.Contains(emailParam(t, welcome, "LoginLink"), "/auth/password-reset?t="), "a sign-in link, never a password")
	assert.Empty(t, emailParam(t, welcome, "Password"))

	plain := parseJSON(mustGet(t, accountUsersPath+"/"+jsonField(got, "id")))
	assertNilField(t, plain, "work_center")
	assertNilField(t, plain, "role")
}

func TestForgeAccountUsers_OmittedFieldsAndUpdate(t *testing.T) {
	t.Parallel()
	got := createTeamMember(t, map[string]any{"role_id": SeedAdminRoleID})
	assert.Equal(t, "ineligible", jsonField(got, "commission_eligibility"), "eligibility defaults to ineligible")
	assertNilField(t, got, "work_center")

	accountUserID := jsonField(got, "id")
	status, raw, err := apiClient.Patch(accountUsersPath+"/"+accountUserID+"?include[]=work_center", map[string]any{
		"work_center_id": SeedDepartmentID, "commission_eligibility": "eligible",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	updated := parseJSON(raw)
	assert.Equal(t, SeedDepartmentID, jsonField(jsonObject(updated, "work_center"), "id"))
	assert.Equal(t, "eligible", jsonField(updated, "commission_eligibility"))

	status, raw, err = apiClient.Patch(accountUsersPath+"/"+accountUserID+"?include[]=work_center", map[string]any{"work_center_id": nil}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	cleared := parseJSON(raw)
	assertNilField(t, cleared, "work_center")
	assert.Equal(t, "eligible", jsonField(cleared, "commission_eligibility"), "an omitted field is kept")
}

func TestForgeAccountUsers_OldFieldsAreRefused(t *testing.T) {
	t.Parallel()
	for _, body := range []map[string]any{
		{"email": "x@e2e-test.openmrp.ai", "department_id": SeedDepartmentID},
		{"email": "x@e2e-test.openmrp.ai", "is_commission_eligible": true},
	} {
		status, raw, err := apiClient.Post(accountUsersPath, body, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, raw)
		requireErrorResponse(t, raw, "parameter_unknown", "invalid_request_error")
	}
	status, raw, err := apiClient.GetListRaw(accountUsersPath, url.Values{"removed_scope": {"included"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, raw)
}

func TestForgeAccountUsers_ActionsAndStatuses(t *testing.T) {
	t.Parallel()
	got := createTeamMember(t, map[string]any{"role_id": SeedSalesRepRoleID})
	accountUserID := jsonField(got, "id")
	base := accountUsersPath + "/" + accountUserID + "/actions/"

	disabled := postAction(t, apiClient, base+"disable", 200)
	assertObjectField(t, disabled, "account_user")
	assert.Equal(t, "disabled", jsonField(disabled, "status"), "an action answers with the updated account user")

	invite := postAction(t, apiClient, base+"invite", 409)
	_ = invite
	removed := postAction(t, apiClient, base+"remove", 200)
	assert.Equal(t, "removed", jsonField(removed, "status"))

	// A removed member is still listed and retrievable: statuses[] omitted lists every status.
	assertListContainsID(t, accountUsersPath, nil, accountUserID)
	assertListContainsID(t, accountUsersPath, url.Values{"statuses[]": {"removed"}}, accountUserID)
	assert.Nil(t, listFindByField(t, accountUsersPath, url.Values{"statuses[]": {"active", "disabled"}}, "id", accountUserID))
	assert.Equal(t, "removed", jsonField(parseJSON(mustGet(t, accountUsersPath+"/"+accountUserID)), "status"))

	status, raw, err := apiClient.Post(base+"disable", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, raw)
	requireErrorResponse(t, raw, "invalid_state_transition", "invalid_request_error")

	activated := postAction(t, apiClient, base+"activate", 200)
	assert.Equal(t, "active", jsonField(activated, "status"))

	status, raw, err = apiClient.Put(base+"activate", nil)
	require.NoError(t, err)
	assert.Contains(t, []int{404, 405}, status, "forge has no PUT actions: %s", raw)
}

func TestForgeAccountUsers_InviteContact(t *testing.T) {
	t.Parallel()
	markOutbox(t)
	customerID := createContactsCustomer(t)
	c := apiClient.WithAccountID(customerID)
	name, email := contactEmail("e2e-idn-invite")
	created := createContact(t, c, name, email, nil)
	assert.Equal(t, "none", jsonField(created, "portal_access"))
	assert.Zero(t, queuedEmailCount(t, email), "adding a contact sends nothing")

	path := accountUsersPath + "/" + jsonField(created, "id") + "/actions/invite"
	key := newIdempotencyKey()
	status, raw, err := c.Post(path, nil, key)
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	invited := parseJSON(raw)
	assert.Equal(t, "invited", jsonField(invited, "portal_access"))
	assert.Equal(t, "active", jsonField(invited, "status"))

	replayStatus, replay, err := c.Post(path, nil, key)
	require.NoError(t, err)
	requireStatus(t, 200, replayStatus, replay)
	assert.Equal(t, jsonField(invited, "id"), jsonField(parseJSON(replay), "id"))
	assert.Equal(t, 1, queuedEmailCount(t, email), "a replayed invite sends no second email")

	supplier := apiClient.WithAccountID(createContactsSupplier(t))
	sname, semail := contactEmail("e2e-idn-sup-invite")
	supplierContact := createContact(t, supplier, sname, semail, nil)
	status, raw, err = supplier.Post(accountUsersPath+"/"+jsonField(supplierContact, "id")+"/actions/invite", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, raw)
}

func TestForgeAccountUsers_CommissionEligibilityFilter(t *testing.T) {
	t.Parallel()
	got := createTeamMember(t, map[string]any{"role_id": SeedAdminRoleID, "commission_eligibility": "eligible"})
	accountUserID := jsonField(got, "id")
	assertListContainsID(t, accountUsersPath, url.Values{"commission_eligibility": {"eligible"}}, accountUserID)
	assert.Nil(t, listFindByField(t, accountUsersPath, url.Values{"commission_eligibility": {"ineligible"}}, "id", accountUserID))
	assertListContainsID(t, accountUsersPath, url.Values{"created_after": {jsonField(got, "created_at")}}, accountUserID)
}

// --- Roles and permissions ---

func TestForgeRoles_CreateFromTemplate(t *testing.T) {
	t.Parallel()
	status, raw, err := apiClient.Post(rolesPath+"?include[]=permissions", map[string]any{
		"name": uniqueName("e2e-idn-role"), "template": "warehouse",
		"access":  map[string]any{"production": "read"},
		"revokes": []string{"shipments:delete"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	role := parseJSON(raw)
	roleID := jsonField(role, "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(rolesPath + "/" + roleID) })

	assertObjectField(t, role, "role")
	assert.Equal(t, "warehouse", jsonField(role, "template"))
	assert.Equal(t, "user", jsonField(role, "type"))
	access := jsonObject(role, "access")
	assert.Equal(t, map[string]any{
		"catalog": "read", "sales": "read", "purchasing": "read", "inventory": "edit",
		"production": "read", "finance": "none", "settings": "none",
	}, access)
	assert.ElementsMatch(t, []any{"receiving:create", "receiving:update"}, role["grants"])
	perms := role["permissions"].([]any)
	assert.Contains(t, perms, "inventory:update")
	assert.Contains(t, perms, "machines:read")
	assert.Contains(t, perms, "messaging:create", "every role holds the baseline")
	assert.NotContains(t, perms, "machines:update")
	assert.NotContains(t, perms, "inventory:delete")

	plain := parseJSON(mustGet(t, rolesPath+"/"+roleID))
	assertNilField(t, plain, "permissions")
	assert.Equal(t, access, jsonObject(plain, "access"), "access is always present")

	// PATCH merges one area and replaces grants.
	status, raw, err = apiClient.Patch(rolesPath+"/"+roleID+"?include[]=permissions", map[string]any{
		"access": map[string]any{"finance": "read"}, "grants": []string{},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	updated := parseJSON(raw)
	assert.Equal(t, "read", jsonField(jsonObject(updated, "access"), "finance"))
	assert.Equal(t, "edit", jsonField(jsonObject(updated, "access"), "inventory"), "other areas are kept")
	assert.Empty(t, updated["grants"])
	assert.Contains(t, updated["permissions"], "invoices:read")
	assert.NotContains(t, updated["permissions"], "receiving:create")
}

func TestForgeRoles_Validation(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		body  map[string]any
		param string
	}{
		{map[string]any{"name": uniqueName("e2e-idn-bad"), "grants": []string{"nope:read"}}, "grants"},
		{map[string]any{"name": uniqueName("e2e-idn-bad"), "access": map[string]any{"sales": "most"}}, "access.sales"},
		{map[string]any{"name": uniqueName("e2e-idn-bad"), "template": "janitor"}, "template"},
	} {
		status, raw, err := apiClient.Post(rolesPath, tc.body, newIdempotencyKey())
		require.NoError(t, err)
		require.Contains(t, []int{400, 422}, status, "%v: %s", tc.body, raw)
	}
}

func TestForgeRoles_DeleteInUse(t *testing.T) {
	t.Parallel()
	role := createAndCleanup(t, rolesPath, map[string]any{"name": uniqueName("e2e-idn-inuse"), "access": map[string]any{"sales": "read"}})
	member := createTeamMember(t, map[string]any{"role_id": jsonField(role, "id")})
	status, raw, err := apiClient.Delete(rolesPath + "/" + jsonField(role, "id"))
	require.NoError(t, err)
	requireStatus(t, 409, status, raw)
	assert.Contains(t, jsonField(requireErrorResponse(t, raw, "resource_in_use", "invalid_request_error"), "message"), "account users")
	removeAccountUser(jsonField(member, "id"))
}

func TestForgeRoles_PreviewCatalogAndTemplates(t *testing.T) {
	t.Parallel()
	status, raw, err := apiClient.Post(rolePreviewPath, map[string]any{"template": "read_only", "revokes": []string{"invoices:read"}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	preview := parseJSON(raw)
	codes := map[string]bool{}
	for _, p := range listRows(t, preview) {
		assertObjectField(t, p, "permission")
		codes[jsonField(p, "code")] = true
	}
	assert.True(t, codes["sales_orders:read"])
	assert.False(t, codes["invoices:read"], "a revoke removes the code")
	assert.False(t, codes["sales_orders:create"])

	first, _, err := apiClient.GetList(permissionsCatalogPath, url.Values{"limit": {"5"}, "areas[]": {"purchasing"}})
	require.NoError(t, err)
	require.Len(t, first.Data, 5)
	for _, item := range first.Data {
		p := parseJSON(item)
		assert.Equal(t, "purchasing", jsonField(p, "area"))
		assert.NotEmpty(t, jsonField(p, "name"))
		assert.NotEmpty(t, jsonField(p, "description"))
		assert.Contains(t, []string{"none", "read", "edit", "full"}, jsonField(p, "level"))
	}
	require.NotNil(t, first.PageInfo.NextPageURL, "the catalog pages")
	assert.NotNil(t, listFindByField(t, permissionsCatalogPath, url.Values{"q": {"suppliers:delete"}}, "code", "suppliers:delete"))

	templates, _, err := apiClient.GetList(roleTemplatesPath, nil)
	require.NoError(t, err)
	require.Len(t, templates.Data, 10)
	owner := parseJSON(templates.Data[0])
	assertObjectField(t, owner, "role_template")
	assert.Equal(t, "owner", jsonField(owner, "id"))
	assert.Equal(t, "full", jsonField(jsonObject(owner, "access"), "settings"))
}

// --- API keys ---

func TestForgeAPIKeys_ClientStatusRevokeAndActor(t *testing.T) {
	t.Parallel()
	status, raw, err := apiClient.Post(forgeAPIKeysPath, map[string]any{
		"name": uniqueName("e2e-idn-key"), "role_id": SeedAdminRoleID,
		"client": map[string]any{"name": "Claude Code", "source": "claude-code"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	created := parseJSON(raw)
	assertObjectField(t, created, "created_api_key")
	secret := jsonField(created, "secret")
	require.NotEmpty(t, secret)
	key := jsonObject(created, "api_key")
	keyID := jsonField(key, "id")
	assert.Equal(t, "active", jsonField(key, "status"))
	assert.Equal(t, map[string]any{"name": "Claude Code", "source": "claude-code"}, jsonObject(key, "client"))
	for _, gone := range []string{"api_key_secret", "api_key_info"} {
		assert.NotContains(t, created, gone)
	}

	// What the key does names it, with its client.
	keyClient := NewClient(apiClient.baseURL, secret, SeedAccountID)
	note := createAndCleanupWith(t, keyClient, "/v1/core/notes", map[string]any{"body": "from the key"})
	actor := jsonObject(note, "actor")
	assert.Equal(t, "api_key", jsonField(actor, "type"))
	assert.Equal(t, keyID, jsonField(actor, "id"))
	assert.Equal(t, "Claude Code", jsonField(actor, "client"))

	revoked := postAction(t, apiClient, forgeAPIKeysPath+"/"+keyID+"/actions/revoke", 200)
	assertObjectField(t, revoked, "api_key")
	assert.Equal(t, "revoked", jsonField(revoked, "status"))
	assertListContainsID(t, forgeAPIKeysPath, url.Values{"statuses": {"revoked"}}, keyID)
}

func createAndCleanupWith(t *testing.T, c *Client, path string, body map[string]any) map[string]any {
	t.Helper()
	status, raw, err := c.Post(path, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	got := parseJSON(raw)
	t.Cleanup(func() { _, _, _ = apiClient.Delete(path + "/" + jsonField(got, "id")) })
	return got
}

// --- 1.0.forge-preview.9 ---

func TestPreview9Identity_OldShapesStillWork(t *testing.T) {
	t.Parallel()
	p9 := apiClient.WithAPIVersion(preview9Version)
	name := uniqueName("e2e-idn-p9")
	status, raw, err := p9.Post(accountUsersPath+"?include=department", map[string]any{
		"name": name, "email": name + "@e2e-test.openmrp.ai", "role_id": SeedAdminRoleID,
		"department_id": SeedDepartmentID, "is_commission_eligible": true,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	got := parseJSON(raw)
	accountUserID := jsonField(got, "id")
	t.Cleanup(func() { removeAccountUser(accountUserID) })
	assert.Equal(t, true, got["is_commission_eligible"])
	assert.Equal(t, SeedDepartmentID, jsonField(jsonObject(got, "department"), "id"))
	assert.NotContains(t, got, "portal_access")
	assert.NotContains(t, got, "work_center")

	status, raw, err = p9.Put(accountUsersPath+"/"+accountUserID+"/actions/remove", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	assertListContainsID(t, accountUsersPath, nil, accountUserID)
	p9list, _, err := p9.GetList(accountUsersPath, url.Values{"limit": {"100"}})
	require.NoError(t, err)
	for _, item := range p9list.Data {
		assert.NotEqual(t, accountUserID, DataItemField(item, "id"), "preview.9 hides removed members by default")
	}

	status, raw, err = p9.Post(forgeAPIKeysPath, map[string]any{"name": uniqueName("e2e-idn-p9-key"), "role_id": SeedAdminRoleID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	key := parseJSON(raw)
	assert.NotEmpty(t, jsonField(key, "api_key_secret"))
	info := jsonObject(key, "api_key_info")
	require.NotNil(t, info)
	assert.NotContains(t, info, "status")
	status, raw, err = p9.Delete(forgeAPIKeysPath + "/" + jsonField(info, "id"))
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)

	role := createAndCleanupP9(t, rolesPath, map[string]any{"name": uniqueName("e2e-idn-p9-role")})
	assert.NotContains(t, role, "access")
	full := parseJSON(mustGet(t, rolesPath+"/"+jsonField(role, "id")+"?include[]=permissions"))
	assert.Empty(t, full["permissions"], "a preview.9 role created without permissions holds none")
}
