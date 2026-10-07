//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Update of a user's global profile: PATCH /v1/identity/users/{id}, plus the photo upload that
// repoints its image.
//
// The dashboard's profile page saves only the signed-in user's own name. Each test works on users it
// registers and adds to the seeded account, so the seeded users other tests read keep their names.

const identityUsersPath = "/v1/identity/users"

// e2eUser is a user registered for one test and added to the seeded account.
type e2eUser struct {
	id            string
	email         string
	accountUserID string
}

func (u e2eUser) path() string { return identityUsersPath + "/" + u.id }

// session signs the user in to the seeded account.
func (u e2eUser) session(t *testing.T) *Client {
	t.Helper()
	return loginAsUser(t, u.email, covAuthUsersPassword, SeedAccountID)
}

// newSeedAccountUser registers a user and adds them to the seeded account under roleID. They are
// removed from the account at cleanup.
func newSeedAccountUser(t *testing.T, roleID string) e2eUser {
	t.Helper()
	email := strings.ToLower(uniqueName("e2e-user")) + "@e2e-test.openmrp.ai"
	status, body, err := apiClient.Post(covAuthUsersRegisterPath, map[string]any{
		"email":    email,
		"password": covAuthUsersPassword,
		"name":     "E2E User",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	userID := jsonField(parseJSON(body), "id")

	status, body, err = apiClient.Post(accountUsersPath, map[string]any{
		"name":    "E2E User",
		"email":   email,
		"role_id": roleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	accountUserID := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { removeAccountUser(accountUserID) })

	return e2eUser{id: userID, email: email, accountUserID: accountUserID}
}

func readUser(t *testing.T, c *Client, userID string) map[string]any {
	t.Helper()
	status, body, err := c.GetListRaw(identityUsersPath+"/"+userID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

// userValues is what an update can change on a user.
func userValues(user map[string]any) map[string]any {
	return map[string]any{
		"name":              user["name"],
		"image_url":         user["image_url"],
		"email_verified_at": user["email_verified_at"],
	}
}

// ──────────────────────────────────────────────
// Update
// ──────────────────────────────────────────────

// A renamed user comes back renamed with every other field intact, from the update itself, from a
// read, and on their membership in the account.
func TestUserUpdate_NameRoundTrips(t *testing.T) {
	t.Parallel()
	u := newSeedAccountUser(t, SeedAdminRoleID)
	name := uniqueName("E2E Renamed User")

	status, body, err := apiClient.Patch(u.path(), map[string]any{"name": name}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	assert.Equal(t, u.id, jsonField(got, "id"))
	assertObjectField(t, got, "user")
	assert.Equal(t, u.email, jsonField(got, "email"))
	assert.Equal(t, name, jsonField(got, "name"))
	assertNilField(t, got, "username")
	assertNilField(t, got, "image_url")
	assertNilField(t, got, "email_verified_at")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	assert.Equal(t, name, jsonField(readUser(t, apiClient, u.id), "name"))

	status, body, err = apiClient.GetListRaw(accountUsersPath+"/"+u.accountUserID, url.Values{"include": {"user"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, name, jsonField(jsonObject(parseJSON(body), "user"), "name"), "the account's view of the user is renamed too")
}

// Every field the update accepts lands, and one left out of a later update keeps its value.
func TestUserUpdate_SetsEachFieldAndKeepsTheOnesLeftOut(t *testing.T) {
	t.Parallel()
	u := newSeedAccountUser(t, SeedAdminRoleID)
	name := uniqueName("E2E All Fields")
	avatar := "https://e2e-test.openmrp.ai/avatars/" + u.id + ".png"
	verified := "2026-01-02T03:04:05Z"

	status, body, err := apiClient.Patch(u.path(), map[string]any{
		"name":           name,
		"image_url":      avatar,
		"email_verified": verified,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	want := map[string]any{"name": name, "image_url": avatar, "email_verified_at": verified}
	assert.Equal(t, want, userValues(parseJSON(body)))
	assert.Equal(t, want, userValues(readUser(t, apiClient, u.id)))

	for field, value := range map[string]string{"name": uniqueName("E2E Only Name"), "image_url": avatar + "?v=2"} {
		status, body, err = apiClient.Patch(u.path(), map[string]any{field: value}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		want[field] = value
		assert.Equal(t, want, userValues(readUser(t, apiClient, u.id)), "updating %s alone leaves the rest", field)
	}
}

// An image URL set here is an image hosted elsewhere, shown as it is, until the user uploads a photo,
// which replaces it.
func TestUserUpdate_ImageURLHoldsAnExternalAvatarUntilAPhotoIsUploaded(t *testing.T) {
	t.Parallel()
	u := newSeedAccountUser(t, SeedAdminRoleID)
	avatar := "https://lh3.googleusercontent.com/a/" + u.id + "=s96-c"

	status, body, err := apiClient.Patch(u.path(), map[string]any{"image_url": avatar}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, avatar, jsonField(parseJSON(body), "image_url"))

	status, body, err = u.session(t).GetListRaw("/v1/identity/me", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, avatar, jsonField(parseJSON(body), "image_url"), "/me shows the external avatar as it is")

	status, body, err = apiClient.PutBytes(u.path()+"/photo", "image/png", onePixelPNG(t))
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.NotEqual(t, avatar, jsonField(readUser(t, apiClient, u.id), "image_url"), "the upload replaces the external avatar")

	status, body, err = apiClient.GetListRaw(u.path()+"/photo", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.NotEmpty(t, jsonField(parseJSON(body), "url"), "the uploaded photo is what is served now")
}

// Invalid values are refused on the field that carries them, and the user is left as they were.
func TestUserUpdate_RejectsInvalidValues(t *testing.T) {
	t.Parallel()
	u := newSeedAccountUser(t, SeedAdminRoleID)
	before := userValues(readUser(t, apiClient, u.id))

	cases := []struct {
		name  string
		body  map[string]any
		code  string
		param string
	}{
		{"blank name", map[string]any{"name": ""}, "invalid_format", "name"},
		{"whitespace name", map[string]any{"name": "   "}, "invalid_format", "name"},
		{"null name", map[string]any{"name": nil}, "invalid_format", "name"},
		{"name over 255", map[string]any{"name": strings.Repeat("a", 256)}, "", "name"},
		{"blank image url", map[string]any{"image_url": ""}, "invalid_format", "image_url"},
		{"image url over 2083", map[string]any{"image_url": "https://e2e-test.openmrp.ai/" + strings.Repeat("a", 2060)}, "", "image_url"},
		{"email verified not a time", map[string]any{"email_verified": "yesterday"}, "", ""},
		{"email is not updatable here", map[string]any{"email": "changed@e2e-test.openmrp.ai"}, "parameter_unknown", "email"},
		{"nothing to update", map[string]any{}, "validation_failed", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body, err := apiClient.Patch(u.path(), tc.body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 400, status, body)
			errObj := requireErrorResponse(t, body, tc.code, "invalid_request_error")
			if tc.param != "" {
				assertErrorParam(t, errObj, tc.param)
			}
		})
	}
	assert.Equal(t, before, userValues(readUser(t, apiClient, u.id)), "refused updates change nothing")
}

// A value of the wrong type, or a malformed timestamp, is refused naming the field it was sent for.
// field.Optional and field.Clearable decode their value with a json.Unmarshal of their own, where the
// decoder attaches no field, so the gateway names it from the body.
func TestUserUpdate_AMalformedValueNamesItsField(t *testing.T) {
	t.Parallel()
	u := newSeedAccountUser(t, SeedAdminRoleID)

	for name, tc := range map[string]struct {
		body  map[string]any
		param string
	}{
		"name as a number":           {map[string]any{"name": 12}, "name"},
		"email verified as a number": {map[string]any{"email_verified": 12}, "email_verified"},
		"email verified not a time":  {map[string]any{"email_verified": "yesterday"}, "email_verified"},
	} {
		t.Run(name, func(t *testing.T) {
			status, body, err := apiClient.Patch(u.path(), tc.body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 400, status, body)
			errObj := requireErrorResponse(t, body, "", "invalid_request_error")
			assert.Equal(t, tc.param, errObj["param"], "the error names the field it is about: %s", body)
		})
	}
}

func TestUserUpdate_UnknownUserIsNotFound(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Patch(identityUsersPath+"/"+mustGenID(t, "us"), map[string]any{"name": "E2E Nobody"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// A retried save replays the first response, and the key cannot be reused for a different name.
func TestUserUpdate_ReplayingAnIdempotencyKeyReturnsTheFirstResult(t *testing.T) {
	t.Parallel()
	u := newSeedAccountUser(t, SeedAdminRoleID)
	key := newIdempotencyKey()
	first := uniqueName("E2E First Name")

	status, body1, err := apiClient.Patch(u.path(), map[string]any{"name": first}, key)
	require.NoError(t, err)
	requireStatus(t, 200, status, body1)
	status, body2, err := apiClient.Patch(u.path(), map[string]any{"name": first}, key)
	require.NoError(t, err)
	requireStatus(t, 200, status, body2)
	assert.Equal(t, parseJSON(body1), parseJSON(body2))

	status, body, err := apiClient.Patch(u.path(), map[string]any{"name": uniqueName("E2E Second Name")}, key)
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "validation_failed", "idempotency_error")
	assert.Equal(t, first, jsonField(readUser(t, apiClient, u.id), "name"))
}

// A rename is recorded in the account's audit log with the old and new name.
func TestUserUpdate_RecordsAnAuditEvent(t *testing.T) {
	t.Parallel()
	u := newSeedAccountUser(t, SeedAdminRoleID)
	name := uniqueName("E2E Audited User")

	status, body, err := apiClient.Patch(u.path(), map[string]any{"name": name}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	event := expectAuditEventWithChanges(t, u.id, "user", "update")
	change, ok := changeForField(jsonListData(event, "changes"), "name")
	require.True(t, ok, "the event records the name: %v", event)
	assert.Equal(t, "E2E User", jsonField(change, "old_value"))
	assert.Equal(t, name, jsonField(change, "new_value"))
}

// ──────────────────────────────────────────────
// Who may update whom
// ──────────────────────────────────────────────

// The profile page saves the signed-in user's own name, which an admin can always do.
func TestUserUpdate_AnAdminRenamesThemselves(t *testing.T) {
	t.Parallel()
	u := newSeedAccountUser(t, SeedAdminRoleID)
	name := uniqueName("E2E Self Admin")

	status, body, err := u.session(t).Patch(u.path(), map[string]any{"name": name}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, name, jsonField(readUser(t, apiClient, u.id), "name"))
}

// Every user may change their own profile, as the dashboard API allowed: a user whose role lacks the team
// permission (a sales rep) still saves their own name from the profile page.
func TestUserUpdate_ANonAdminRenamesThemselves(t *testing.T) {
	t.Parallel()
	u := newSeedAccountUser(t, SeedSalesRepRoleID)
	name := uniqueName("E2E Self Rep")

	status, body, err := u.session(t).Patch(u.path(), map[string]any{"name": name}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 200, status, "a user may rename themselves without the team permission: %s", body)
	assert.Equal(t, name, jsonField(readUser(t, apiClient, u.id), "name"), "the user's own rename is saved")
}

// Renaming someone else takes the team permission, which a sales rep's role does not grant.
func TestUserUpdate_ANonAdminCannotRenameSomeoneElse(t *testing.T) {
	t.Parallel()
	rep := newSeedAccountUser(t, SeedSalesRepRoleID)
	other := newSeedAccountUser(t, SeedAdminRoleID)

	status, body, err := rep.session(t).Patch(other.path(), map[string]any{"name": uniqueName("E2E Hijack")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
	assert.Equal(t, "E2E User", jsonField(readUser(t, apiClient, other.id), "name"))
}

// The team permission says the caller may manage users in their own account, not that the user named is
// one of them: another tenant with team:update can neither rename this account's user, repoint their
// image, nor mark their email verified.
func TestUserUpdate_AnotherTenantCannotChangeThisAccountsUser(t *testing.T) {
	t.Parallel()
	u := newSeedAccountUser(t, SeedAdminRoleID)
	before := userValues(readUser(t, apiClient, u.id))
	tenantB := getTenantBClient()

	for field, value := range map[string]any{
		"name":           uniqueName("E2E Cross Tenant"),
		"image_url":      "https://e2e-test.openmrp.ai/cross-tenant.png",
		"email_verified": "2026-01-01T00:00:00Z",
	} {
		t.Run(field, func(t *testing.T) {
			status, body, err := tenantB.Patch(u.path(), map[string]any{field: value}, newIdempotencyKey())
			require.NoError(t, err)
			require.Less(t, status, 500, "must not 5xx: %s", body)
			assert.Contains(t, []int{403, 404}, status, "another tenant must not change this account's user: %s", body)
		})
	}
	assert.Equal(t, before, userValues(readUser(t, apiClient, u.id)), "the user is untouched by another tenant")
}

// Reading a user is scoped like updating one. The read also looks users up by email, so another tenant
// must learn neither this user's name and email nor whether an address has an account at all. Outside
// the TestUserUpdate_ prefix because it is the retrieve endpoint.
func TestUserRetrieve_AnotherTenantsUserIsNotFound(t *testing.T) {
	t.Parallel()
	u := newSeedAccountUser(t, SeedAdminRoleID)
	tenantB := getTenantBClient()

	for name, identifier := range map[string]string{"by id": u.id, "by email": u.email} {
		t.Run(name, func(t *testing.T) {
			status, body, err := tenantB.GetListRaw(identityUsersPath+"/"+url.PathEscape(identifier), nil)
			require.NoError(t, err)
			require.Less(t, status, 500, "must not 5xx: %s", body)
			assert.Contains(t, []int{403, 404}, status, "another tenant must not read this account's user: %s", body)
			assert.NotContains(t, string(body), u.email, "a refusal must not carry the user's email")
		})
	}
}

// ──────────────────────────────────────────────
// Photo
// ──────────────────────────────────────────────

// The photo upload stores only images, judged by their bytes rather than the content type the request
// names.
func TestUserPhoto_RejectsAnUploadThatIsNotAnImage(t *testing.T) {
	t.Parallel()
	u := newSeedAccountUser(t, SeedAdminRoleID)

	for _, up := range nonImageUploads {
		t.Run(up.name, func(t *testing.T) {
			status, body, err := apiClient.PutBytes(u.path()+"/photo", up.contentType, up.body)
			require.NoError(t, err)
			require.Less(t, status, 500, "must not 5xx: %s", body)
			assert.Contains(t, []int{400, 415}, status, "a %s upload is not an image: %s", up.name, body)
		})
	}
	status, body, err := apiClient.GetListRaw(u.path()+"/photo", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Nil(t, parseJSON(body)["url"], "refused uploads leave the user without a photo: %s", body)
}

// As with the account logo, a photo over the image limit is refused rather than cut short and stored, and
// one over 1 MiB but within the limit is taken.
func TestUserPhoto_RejectsAnOversizedUpload(t *testing.T) {
	t.Parallel()
	u := newSeedAccountUser(t, SeedAdminRoleID)

	status, body, err := apiClient.PutBytes(u.path()+"/photo", "image/png", oversizedPNG(t))
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", body)
	assert.Contains(t, []int{400, 413}, status, "a photo over the image limit must be refused, not truncated: %s", body)

	status, body, err = apiClient.PutBytes(u.path()+"/photo", "image/png", pngOfSize(t, 2*oneMiB))
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

// A user may always upload their own photo, as on the profile page.
func TestUserPhoto_AnAdminUploadsTheirOwn(t *testing.T) {
	t.Parallel()
	u := newSeedAccountUser(t, SeedAdminRoleID)
	session := u.session(t)

	status, body, err := session.PutBytes(u.path()+"/photo", "image/png", onePixelPNG(t))
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = session.GetListRaw(u.path()+"/photo", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.NotEmpty(t, jsonField(parseJSON(body), "url"))
}

// Like the rename, a user whose role lacks team:update still uploads their own photo, as the dashboard
// API allowed.
func TestUserPhoto_ANonAdminUploadsTheirOwn(t *testing.T) {
	t.Parallel()
	u := newSeedAccountUser(t, SeedSalesRepRoleID)
	session := u.session(t)

	status, body, err := session.PutBytes(u.path()+"/photo", "image/png", onePixelPNG(t))
	require.NoError(t, err)
	assert.Equal(t, 200, status, "a user may upload their own photo without the team permission: %s", body)

	status, body, err = session.GetListRaw(u.path()+"/photo", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.NotEmpty(t, jsonField(parseJSON(body), "url"), "the user's own photo is stored")
}
