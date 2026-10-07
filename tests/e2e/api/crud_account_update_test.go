//go:build e2e

package api_test

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Retrieve and update of the account the caller acts in: GET/PATCH /v1/identity/accounts/{id}.
//
// The dashboard's account settings and customer portal pages read the account with every include and
// save it field by field, sending null for a field the user emptied. Writes go to an account each test
// registers for itself: the seeded account is read by most of the suite, and its slug is what portal and
// registration tests look it up by.

// freshAccountSlots bounds how many accounts the package holds open at once. Self-serve registration on
// the free plan closes once ten active accounts are on it, and a test's account is only shut down at
// cleanup, so an unbounded parallel run could lock out TestRegistration_FullJourney.
var freshAccountSlots = make(chan struct{}, 3)

// e2eAccount is an account registered for one test, with a session for the user who registered it,
// which is the account's admin.
type e2eAccount struct {
	id    string
	owner *Client
}

func (a e2eAccount) path() string { return accountsPath + "/" + a.id }

// registerE2EAccount runs self-serve registration to the end, giving a test an account of its own
// with a portal, branding, and an admin session. It is shut down again at cleanup.
func registerE2EAccount(t *testing.T) e2eAccount {
	t.Helper()
	freshAccountSlots <- struct{}{}
	t.Cleanup(func() { <-freshAccountSlots })

	email := strings.ToLower(uniqueName("e2e-acct")) + "@e2e-test.openmrp.ai"
	status, body, err := apiClient.Post(registrationSessionsPath, map[string]any{
		"email":     email,
		"plan_code": "free",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	sessionID := jsonField(parseJSON(body), "id")
	sessionPath := registrationSessionsPath + "/" + sessionID

	status, body, err = apiClient.Put(verifyTokenPath(registrationVerificationToken(t, sessionID)), nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	resp, err := apiClient.PostFull(sessionPath+"/users", map[string]any{
		"name":     "E2E Account Owner",
		"password": "P@ssw0rd123!",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	token := accessTokenFromSetCookie(t, resp.Header)
	registrant := apiClient.WithBearerToken(token, "")

	status, body, err = registrant.Patch(sessionPath, map[string]any{
		"session_data": map[string]any{"account_name": uniqueName("E2E Account")},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = registrant.Post(sessionPath+"/accounts", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	accountID := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, accountID, "registration returns the account: %s", body)
	t.Cleanup(func() {
		_, _ = authDB(t).Exec("UPDATE account SET onboarding_status_code = 'deactivated' WHERE id = ?", accountID)
	})

	return e2eAccount{id: accountID, owner: apiClient.WithBearerToken(token, accountID)}
}

// allAccountIncludes is what the dashboard asks for on every read and save.
var allAccountIncludes = url.Values{"include": {"branding", "portal", "default_billing_address", "default_shipping_address"}}

const allAccountIncludesQuery = "?include=branding&include=portal&include=default_billing_address&include=default_shipping_address"

// brandingFields are the account's branding values a PATCH sets, and null removes.
var brandingFields = []string{
	"support_email", "phone_number", "website_url",
	"facebook_handle", "instagram_handle", "linkedin_handle", "twitter_handle",
}

// sampleBranding is a valid value for every branding field.
func sampleBranding() map[string]any {
	tag := strings.ToLower(uniqueName("e2e"))
	return map[string]any{
		"support_email":    tag + "@e2e-test.openmrp.ai",
		"phone_number":     "555-" + tag[len(tag)-8:],
		"website_url":      "https://" + tag + ".e2e-test.openmrp.ai",
		"facebook_handle":  tag + "-fb",
		"instagram_handle": tag + "-ig",
		"linkedin_handle":  tag + "-li",
		"twitter_handle":   tag + "-tw",
	}
}

func readAccount(t *testing.T, c *Client, accountID string) map[string]any {
	t.Helper()
	status, body, err := c.GetListRaw(accountsPath+"/"+accountID, allAccountIncludes)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

// patchAccount sends an update with every include and requires it to succeed.
func patchAccount(t *testing.T, c *Client, accountID string, body map[string]any) map[string]any {
	t.Helper()
	status, resp, err := c.Patch(accountsPath+"/"+accountID+allAccountIncludesQuery, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, resp)
	return parseJSON(resp)
}

// accountValues flattens what a client can change on an account into one comparable map. A value the
// account does not have is nil.
func accountValues(account map[string]any) map[string]any {
	values := map[string]any{"name": account["name"]}
	branding := jsonObject(account, "branding")
	for _, f := range brandingFields {
		values[f] = nil
		if branding != nil {
			values[f] = branding[f]
		}
	}
	values["slug"] = nil
	if portal := jsonObject(account, "portal"); portal != nil {
		values["slug"] = portal["slug"]
	}
	for _, f := range []string{"default_billing_address", "default_shipping_address"} {
		values[f+"_id"] = nil
		if addr := jsonObject(account, f); addr != nil {
			values[f+"_id"] = addr["id"]
		}
	}
	return values
}

// valuesWithout returns values minus the named keys: everything an update should not have touched.
func valuesWithout(values map[string]any, keys ...string) map[string]any {
	out := make(map[string]any, len(values))
	for k, v := range values {
		out[k] = v
	}
	for _, k := range keys {
		delete(out, k)
	}
	return out
}

// createAccountAddress creates an address owned by the client's account, as the dashboard does when the
// account has no billing address to edit yet.
func createAccountAddress(t *testing.T, c *Client, name string) string {
	t.Helper()
	status, body, err := c.Post("/v1/sales/addresses", map[string]any{
		"name":          name,
		"street_line_1": "123 Test St",
		"locality":      "Los Angeles",
		"state":         "CA",
		"postal_code":   "90001",
		"country":       "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	return jsonField(parseJSON(body), "id")
}

// ──────────────────────────────────────────────
// Retrieve
// ──────────────────────────────────────────────

// A newly registered account is what every update test below starts from: a portal at a slug made from
// its ID, branding with nothing set, and no default addresses.
func TestAccounts_RegisteredAccountStartsBare(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)

	got := readAccount(t, acct.owner, acct.id)
	assert.Equal(t, acct.id, jsonField(got, "id"))
	assertObjectField(t, got, "account")
	assert.NotEmpty(t, jsonField(got, "name"))
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	branding := jsonObject(got, "branding")
	require.NotNil(t, branding, "registration creates the branding: %v", got)
	assertIDFormat(t, jsonField(branding, "id"), "acbr")
	assertObjectField(t, branding, "account_branding")
	for _, f := range append(append([]string{}, brandingFields...), "logo_url", "favicon_url") {
		assertNilField(t, branding, f)
	}
	assertValidTimestamp(t, jsonField(branding, "created_at"), "branding.created_at")
	assertValidTimestamp(t, jsonField(branding, "updated_at"), "branding.updated_at")

	portal := jsonObject(got, "portal")
	require.NotNil(t, portal, "registration creates the portal: %v", got)
	assertIDFormat(t, jsonField(portal, "id"), "acpo")
	assertObjectField(t, portal, "account_portal")
	assert.Equal(t, strings.ReplaceAll(acct.id, "_", "-"), jsonField(portal, "slug"))
	assertValidTimestamp(t, jsonField(portal, "created_at"), "portal.created_at")
	assertValidTimestamp(t, jsonField(portal, "updated_at"), "portal.updated_at")

	assertNilField(t, got, "default_billing_address")
	assertNilField(t, got, "default_shipping_address")
}

// Each include expands only itself, so a caller asking for one sub-resource is not handed the rest.
func TestAccounts_EachIncludeExpandsOnlyItself(t *testing.T) {
	t.Parallel()
	expandables := map[string]string{
		"branding":                 "account_branding",
		"portal":                   "account_portal",
		"default_billing_address":  "address",
		"default_shipping_address": "address",
	}
	for include, object := range expandables {
		t.Run(include, func(t *testing.T) {
			t.Parallel()
			status, body, err := apiClient.GetListRaw(accountsPath+"/"+SeedAccountID, url.Values{"include": {include}})
			require.NoError(t, err)
			requireStatus(t, 200, status, body)
			got := parseJSON(body)

			expanded := jsonObject(got, include)
			require.NotNil(t, expanded, "?include=%s expands it: %s", include, body)
			assertObjectField(t, expanded, object)
			assert.NotEmpty(t, jsonField(expanded, "id"))
			for other := range expandables {
				if other != include {
					assertNilField(t, got, other)
				}
			}
		})
	}
}

// The dashboard reads the account with all four includes at once; every field it maps must be there.
func TestAccounts_RetrieveWithEveryIncludeAsTheDashboardDoes(t *testing.T) {
	t.Parallel()
	got := readAccount(t, apiClient, SeedAccountID)

	assert.Equal(t, SeedAccountID, jsonField(got, "id"))
	assertObjectField(t, got, "account")
	assert.NotEmpty(t, jsonField(got, "name"))

	branding := jsonObject(got, "branding")
	require.NotNil(t, branding)
	for _, f := range append(append([]string{}, brandingFields...), "logo_url", "favicon_url", "id", "object", "created_at", "updated_at") {
		_, present := branding[f]
		assert.True(t, present, "branding carries %s, null or not: %v", f, branding)
	}

	portal := jsonObject(got, "portal")
	require.NotNil(t, portal)
	assert.Equal(t, SeedAccountSlug, jsonField(portal, "slug"))

	for _, f := range []string{"default_billing_address", "default_shipping_address"} {
		addr := jsonObject(got, f)
		require.NotNil(t, addr, "the seeded account has a %s: %v", f, got)
		assertIDFormat(t, jsonField(addr, "id"), "ad")
		assertObjectField(t, addr, "address")
		assertObjectField(t, jsonObject(addr, "geolocation"), "geolocation")
	}
}

func TestAccounts_RetrieveRejectsAnUnknownInclude(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountsPath+"/"+SeedAccountID, url.Values{"include": {"users"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

// The dashboard reads the account as the signed-in user, not with an API key; both must see the same account.
func TestAccounts_RetrieveAsASignedInUser(t *testing.T) {
	t.Parallel()
	asKey := readAccount(t, apiClient, SeedAccountID)
	asUser := readAccount(t, loginAsSeedUser(t), SeedAccountID)

	assert.Equal(t, SeedAccountID, jsonField(asUser, "id"))
	assert.Equal(t, accountValues(asKey), accountValues(asUser))
}

// Only the account the caller acts in can be read; asking for any other is refused without revealing it.
func TestAccounts_RetrieveOnlyTheAccountActedIn(t *testing.T) {
	t.Parallel()
	for name, id := range map[string]string{
		"another tenant": SeedTenantBAccountID,
		"a customer":     SeedCustomerAccountID,
		"unknown":        "ac_doesnotexist00000",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			status, body, err := apiClient.GetListRaw(accountsPath+"/"+id, allAccountIncludes)
			require.NoError(t, err)
			require.Less(t, status, 500, "must not 5xx: %s", body)
			assert.Contains(t, []int{403, 404}, status, "reading account %s must be refused: %s", id, body)
			assert.Nil(t, parseJSON(body)["name"], "a refusal must not carry the account: %s", body)
		})
	}
}

// Another tenant can neither read this account nor change it, whether it names the account in the path
// or claims it in the OpenMRP-Account header. The writes carry the name the account already has, so a
// wrongly allowed one changes nothing the rest of the suite reads.
func TestAccounts_AnotherTenantCannotReadOrUpdateThisAccount(t *testing.T) {
	t.Parallel()
	before := accountValues(readAccount(t, apiClient, SeedAccountID))
	tenantB := getTenantBClient()

	status, body, err := tenantB.GetListRaw(accountsPath+"/"+SeedAccountID, allAccountIncludes)
	require.NoError(t, err)
	assert.Contains(t, []int{403, 404}, status, "another tenant must not read this account: %s", body)

	status, body, err = tenantB.Patch(accountsPath+"/"+SeedAccountID, map[string]any{"name": before["name"]}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Contains(t, []int{403, 404}, status, "another tenant must not rename this account: %s", body)

	claiming := NewClient(envOr("E2E_BASE_URL", defaultBaseURL), SeedTenantBAPIKey, SeedAccountID)
	status, body, err = claiming.GetListRaw(accountsPath+"/"+SeedAccountID, nil)
	require.NoError(t, err)
	assert.Contains(t, []int{401, 403, 404}, status, "a key cannot act in an account it does not belong to: %s", body)

	status, body, err = claiming.Patch(accountsPath+"/"+SeedAccountID, map[string]any{"name": before["name"]}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Contains(t, []int{401, 403, 404}, status, "a key cannot act in an account it does not belong to: %s", body)

	assert.Equal(t, before, accountValues(readAccount(t, apiClient, SeedAccountID)), "the account is untouched")
}

// Reading and changing the account take the account permission, which a sales rep's role does not
// grant. The dashboard API checked the same permission.
func TestAccounts_ARoleWithoutTheAccountPermissionIsRefused(t *testing.T) {
	t.Parallel()
	salesRep := newSeedAccountUser(t, SeedSalesRepRoleID).session(t)
	before := accountValues(readAccount(t, apiClient, SeedAccountID))

	status, body, err := salesRep.GetListRaw(accountsPath+"/"+SeedAccountID, nil)
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")

	// The name it already has, so a wrongly allowed write changes nothing other tests read.
	status, body, err = salesRep.Patch(accountsPath+"/"+SeedAccountID, map[string]any{"name": before["name"]}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
}

// ──────────────────────────────────────────────
// Update
// ──────────────────────────────────────────────

// Both an API key and a signed-in admin may update the account. Not parallel: it writes the seeded
// account, though only the name it already has.
func TestAccountUpdate_AnAPIKeyAndASignedInAdminCanBothUpdate(t *testing.T) {
	name := jsonField(readAccount(t, apiClient, SeedAccountID), "name")
	for label, c := range map[string]*Client{"api key": apiClient, "session": loginAsSeedUser(t)} {
		got := patchAccount(t, c, SeedAccountID, map[string]any{"name": name})
		assert.Equal(t, name, jsonField(got, "name"), label)
		assert.NotNil(t, jsonObject(got, "branding"), "%s: the update honors ?include", label)
	}
}

// Setting one field leaves every other field as it was. The account starts with every branding field
// set, so a field wrongly reset to null would show.
func TestAccountUpdate_EachFieldChangesOnlyItself(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)
	patchAccount(t, acct.owner, acct.id, sampleBranding())

	next := sampleBranding()
	next["name"] = uniqueName("E2E Renamed")
	next["slug"] = strings.ToLower(uniqueName("e2e-slug"))
	for _, f := range append([]string{"name", "slug"}, brandingFields...) {
		t.Run(f, func(t *testing.T) {
			before := accountValues(readAccount(t, acct.owner, acct.id))
			require.NotEqual(t, next[f], before[f], "the test must change %s", f)

			resp := accountValues(patchAccount(t, acct.owner, acct.id, map[string]any{f: next[f]}))
			assert.Equal(t, next[f], resp[f], "the response carries the new %s", f)

			after := accountValues(readAccount(t, acct.owner, acct.id))
			assert.Equal(t, next[f], after[f], "%s is saved", f)
			assert.Equal(t, valuesWithout(before, f), valuesWithout(after, f), "setting %s leaves the rest alone", f)
		})
	}
}

// The dashboard sends null for a field the user emptied, since its SDK drops empty strings; null removes
// the value, and a field the request leaves out keeps its own.
func TestAccountUpdate_NullRemovesABrandingFieldAndOmittingItKeepsIt(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)
	patchAccount(t, acct.owner, acct.id, sampleBranding())

	for _, f := range brandingFields {
		t.Run(f, func(t *testing.T) {
			before := accountValues(readAccount(t, acct.owner, acct.id))
			require.NotNil(t, before[f], "the test needs %s set", f)

			resp := accountValues(patchAccount(t, acct.owner, acct.id, map[string]any{f: nil}))
			assert.Nil(t, resp[f], "the response shows %s removed", f)

			after := accountValues(readAccount(t, acct.owner, acct.id))
			assert.Nil(t, after[f], "null removes %s", f)
			assert.Equal(t, valuesWithout(before, f), valuesWithout(after, f), "removing %s leaves the rest alone", f)
		})
	}
}

// The settings form saves everything at once, emptied fields as null, along with the billing address it
// just created; one request must land all of it.
func TestAccountUpdate_SavesTheDashboardSettingsForm(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)
	patchAccount(t, acct.owner, acct.id, sampleBranding())
	billingID := createAccountAddress(t, acct.owner, "E2E Settings Billing")

	form := map[string]any{
		"name":                       uniqueName("E2E Settings"),
		"support_email":              "settings@e2e-test.openmrp.ai",
		"phone_number":               nil,
		"website_url":                "https://settings.e2e-test.openmrp.ai",
		"instagram_handle":           "settings-ig",
		"facebook_handle":            nil,
		"twitter_handle":             nil,
		"linkedin_handle":            "settings-li",
		"slug":                       strings.ToLower(uniqueName("e2e-settings")),
		"default_billing_address_id": billingID,
	}
	got := patchAccount(t, acct.owner, acct.id, form)

	want := map[string]any{}
	for k, v := range form {
		want[k] = v
	}
	want["default_shipping_address_id"] = nil
	assert.Equal(t, want, accountValues(got), "the response reflects the whole form")
	assert.Equal(t, want, accountValues(readAccount(t, acct.owner, acct.id)), "the whole form is saved")

	billing := jsonObject(got, "default_billing_address")
	require.NotNil(t, billing)
	assert.Equal(t, "E2E Settings Billing", jsonField(billing, "name"))
	assert.Equal(t, "123 Test St", jsonField(jsonObject(billing, "geolocation"), "street_line_1"))
}

// A new slug moves the customer portal: the public branding lookup finds the account under it, and the
// old slug stops resolving. Saving the slug the account already has is not a conflict with itself.
func TestAccountUpdate_ChangingTheSlugMovesThePortal(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)
	oldSlug := accountValues(readAccount(t, acct.owner, acct.id))["slug"].(string)
	newSlug := strings.ToLower(uniqueName("e2e-portal"))

	got := patchAccount(t, acct.owner, acct.id, map[string]any{"slug": newSlug})
	assert.Equal(t, newSlug, jsonField(jsonObject(got, "portal"), "slug"))

	status, body, err := apiClient.GetListRaw("/v1/settings/branding/"+newSlug, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, acct.id, jsonField(parseJSON(body), "id"), "the new slug resolves to the account")

	status, body, err = apiClient.GetListRaw("/v1/settings/branding/"+oldSlug, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "the old slug no longer resolves: %s", body)

	got = patchAccount(t, acct.owner, acct.id, map[string]any{"slug": newSlug})
	assert.Equal(t, newSlug, jsonField(jsonObject(got, "portal"), "slug"), "re-saving its own slug is allowed")
}

// Slugs are unique across accounts, and the lookup that resolves them ignores case, so a slug that
// differs from a taken one only in case is taken too.
func TestAccountUpdate_SlugTakenByAnotherAccountConflicts(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)
	before := accountValues(readAccount(t, acct.owner, acct.id))

	for _, slug := range []string{SeedAccountSlug, strings.ToUpper(SeedAccountSlug)} {
		status, body, err := acct.owner.Patch(acct.path(), map[string]any{"slug": slug}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 409, status, body)
		assertErrorParam(t, requireErrorResponse(t, body, "resource_conflict", "invalid_request_error"), "slug")
	}

	// A refused slug must not take the rest of the request with it, nor half-apply it.
	status, body, err := acct.owner.Patch(acct.path(), map[string]any{"slug": SeedAccountSlug, "name": uniqueName("E2E Conflict")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	assert.Equal(t, before, accountValues(readAccount(t, acct.owner, acct.id)), "a conflicting update changes nothing")
}

// Every field is checked before anything is written, and a refused update leaves the account as it was.
func TestAccountUpdate_RejectsInvalidValues(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)
	patchAccount(t, acct.owner, acct.id, sampleBranding())
	before := accountValues(readAccount(t, acct.owner, acct.id))

	long := func(n int) string { return strings.Repeat("a", n) }
	cases := []struct {
		name  string
		body  map[string]any
		code  string
		param string
	}{
		{"blank name", map[string]any{"name": ""}, "invalid_format", "name"},
		{"whitespace name", map[string]any{"name": "   "}, "invalid_format", "name"},
		{"null name", map[string]any{"name": nil}, "invalid_format", "name"},
		{"name over 255", map[string]any{"name": long(256)}, "", "name"},
		{"malformed support email", map[string]any{"support_email": "not-an-email"}, "invalid_format", "support_email"},
		{"support email over 255", map[string]any{"support_email": long(250) + "@e2e-test.openmrp.ai"}, "", "support_email"},
		{"phone over 255", map[string]any{"phone_number": long(256)}, "", "phone_number"},
		{"website not a url", map[string]any{"website_url": "not a url"}, "invalid_format", "website_url"},
		{"website without a scheme", map[string]any{"website_url": "e2e-test.openmrp.ai"}, "invalid_format", "website_url"},
		{"website over 2083", map[string]any{"website_url": "https://e2e-test.openmrp.ai/" + long(2060)}, "", "website_url"},
		{"facebook over 255", map[string]any{"facebook_handle": long(256)}, "", "facebook_handle"},
		{"instagram over 255", map[string]any{"instagram_handle": long(256)}, "", "instagram_handle"},
		{"linkedin over 255", map[string]any{"linkedin_handle": long(256)}, "", "linkedin_handle"},
		{"twitter over 255", map[string]any{"twitter_handle": long(256)}, "", "twitter_handle"},
		{"slug under 3", map[string]any{"slug": "ab"}, "", "slug"},
		{"blank slug", map[string]any{"slug": ""}, "invalid_format", "slug"},
		{"null slug", map[string]any{"slug": nil}, "invalid_format", "slug"},
		{"slug over 255", map[string]any{"slug": long(256)}, "", "slug"},
		{"blank billing address", map[string]any{"default_billing_address_id": ""}, "invalid_format", "default_billing_address_id"},
		{"null shipping address", map[string]any{"default_shipping_address_id": nil}, "invalid_format", "default_shipping_address_id"},
		{"logo set through the update", map[string]any{"logo_url": "https://e2e-test.openmrp.ai/logo.png"}, "parameter_unknown", "logo_url"},
		{"an invalid field alongside a valid one", map[string]any{"name": uniqueName("E2E Valid"), "support_email": "nope"}, "invalid_format", "support_email"},
		{"nothing to update", map[string]any{}, "validation_failed", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body, err := acct.owner.Patch(acct.path(), tc.body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 400, status, body)
			errObj := requireErrorResponse(t, body, tc.code, "invalid_request_error")
			if tc.param != "" {
				assertErrorParam(t, errObj, tc.param)
			}
		})
	}
	assert.Equal(t, before, accountValues(readAccount(t, acct.owner, acct.id)), "refused updates change nothing")
}

// The slug is the path segment of the customer portal (<frontend>/<slug>/login), so a slug with a space,
// slash, or query character, which would break the portal links built from it, is refused.
func TestAccountUpdate_RejectsASlugThatIsNotURLSafe(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)

	for _, slug := range []string{"has spaces", "slash/slug", "query?x=1", "frag#ment"} {
		t.Run(slug, func(t *testing.T) {
			before := accountValues(readAccount(t, acct.owner, acct.id))["slug"]
			status, body, err := acct.owner.Patch(acct.path(), map[string]any{"slug": slug}, newIdempotencyKey())
			require.NoError(t, err)
			assert.Equal(t, 400, status, "a slug that is not URL-safe must be refused: %s", body)
			if status == 400 {
				assertErrorParam(t, requireErrorResponse(t, body, "", "invalid_request_error"), "slug")
			}
			assert.Equal(t, before, accountValues(readAccount(t, acct.owner, acct.id))["slug"], "the slug must be left as it was")
		})
	}
}

// The account's defaults are what new orders bill and ship to; both must be settable, each alone, and
// come back through their includes.
func TestAccountUpdate_SetsTheDefaultBillingAndShippingAddresses(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)
	billingID := createAccountAddress(t, acct.owner, "E2E Default Billing")
	shippingID := createAccountAddress(t, acct.owner, "E2E Default Shipping")

	got := patchAccount(t, acct.owner, acct.id, map[string]any{
		"default_billing_address_id":  billingID,
		"default_shipping_address_id": shippingID,
	})
	values := accountValues(got)
	assert.Equal(t, billingID, values["default_billing_address_id"])
	assert.Equal(t, shippingID, values["default_shipping_address_id"])
	assert.Equal(t, "E2E Default Billing", jsonField(jsonObject(got, "default_billing_address"), "name"))
	assert.Equal(t, "E2E Default Shipping", jsonField(jsonObject(got, "default_shipping_address"), "name"))

	// One address may be both; setting only billing leaves shipping where it was.
	before := accountValues(readAccount(t, acct.owner, acct.id))
	got = patchAccount(t, acct.owner, acct.id, map[string]any{"default_billing_address_id": shippingID})
	assert.Equal(t, shippingID, accountValues(got)["default_billing_address_id"])
	after := accountValues(readAccount(t, acct.owner, acct.id))
	assert.Equal(t, shippingID, after["default_billing_address_id"])
	assert.Equal(t, valuesWithout(before, "default_billing_address_id"), valuesWithout(after, "default_billing_address_id"))

	// Without the include the default stays unexpanded, as every expandable does.
	status, body, err := acct.owner.GetListRaw(acct.path(), nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assertNilField(t, parseJSON(body), "default_billing_address")
	assertNilField(t, parseJSON(body), "default_shipping_address")
}

// A default must be one of the account's own addresses: another tenant's, a deleted one, or one that
// does not exist is refused on the field that named it, and the defaults stay put.
func TestAccountUpdate_RefusesADefaultAddressTheAccountDoesNotHave(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)
	ownID := createAccountAddress(t, acct.owner, "E2E Own Address")
	patchAccount(t, acct.owner, acct.id, map[string]any{"default_billing_address_id": ownID, "default_shipping_address_id": ownID})
	before := accountValues(readAccount(t, acct.owner, acct.id))

	deletedID := createAccountAddress(t, acct.owner, "E2E Deleted Address")
	status, body, err := acct.owner.Delete("/v1/sales/addresses/" + deletedID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	for name, addressID := range map[string]string{
		"another tenant's":   createE2EAddress(t, uniqueName("e2e-other-tenant-address")),
		"a deleted one":      deletedID,
		"one that is absent": mustGenID(t, "ad"),
	} {
		for _, param := range []string{"default_billing_address_id", "default_shipping_address_id"} {
			t.Run(name+" as "+param, func(t *testing.T) {
				status, body, err := acct.owner.Patch(acct.path(), map[string]any{param: addressID}, newIdempotencyKey())
				require.NoError(t, err)
				require.Less(t, status, 500, "must not 5xx: %s", body)
				assert.Contains(t, []int{400, 404}, status, "%s address must be refused: %s", name, body)
				if status == 400 {
					assertErrorParam(t, requireErrorResponse(t, body, "validation_failed", "invalid_request_error"), param)
				}
			})
		}
	}
	assert.Equal(t, before, accountValues(readAccount(t, acct.owner, acct.id)), "the defaults are unchanged")
}

// A retried save replays the first response instead of applying twice, and the key cannot be reused
// for a different update.
func TestAccountUpdate_ReplayingAnIdempotencyKeyReturnsTheFirstResult(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)
	key := newIdempotencyKey()
	first := uniqueName("E2E First")

	r1, err := acct.owner.PatchFull(acct.path(), map[string]any{"name": first}, key)
	require.NoError(t, err)
	requireStatus(t, 200, r1.StatusCode, r1.Body)

	r2, err := acct.owner.PatchFull(acct.path(), map[string]any{"name": first}, key)
	require.NoError(t, err)
	requireStatus(t, 200, r2.StatusCode, r2.Body)
	assert.Equal(t, parseJSON(r1.Body), parseJSON(r2.Body), "the replay is the first response")
	assert.Equal(t, "true", r2.Header.Get("Idempotent-Replayed"))

	status, body, err := acct.owner.Patch(acct.path(), map[string]any{"name": uniqueName("E2E Second")}, key)
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "validation_failed", "idempotency_error")
	assert.Equal(t, first, jsonField(readAccount(t, acct.owner, acct.id), "name"))
}

// accountUpdateEvent polls the account's own audit log until an update event's changes satisfy match.
func accountUpdateEvent(t *testing.T, acct e2eAccount, match func(changes []any) bool) []any {
	t.Helper()
	var found []any
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		status, body, err := acct.owner.GetListRaw(auditEventsPath, url.Values{
			"resource_ids":   {acct.id},
			"resource_types": {"account"},
			"actions":        {"update"},
			"include":        {"changes"},
			"limit":          {"25"},
		})
		if err != nil {
			return err
		}
		if status != 200 {
			return fmt.Errorf("audit events: %d %s", status, body)
		}
		for _, item := range jsonArray(parseJSON(body), "data") {
			event, _ := item.(map[string]any)
			if changes := jsonListData(event, "changes"); match(changes) {
				found = changes
				return nil
			}
		}
		return fmt.Errorf("no matching account update event yet")
	})
	return found
}

// Renaming the account is recorded in its audit log with the old and new name.
func TestAccountUpdate_RecordsAnAuditEvent(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)
	oldName := jsonField(readAccount(t, acct.owner, acct.id), "name")
	newName := uniqueName("E2E Audited")
	patchAccount(t, acct.owner, acct.id, map[string]any{"name": newName})

	changes := accountUpdateEvent(t, acct, func(changes []any) bool {
		c, ok := changeForField(changes, "name")
		return ok && jsonField(c, "new_value") == newName
	})
	c, _ := changeForField(changes, "name")
	assert.Equal(t, oldName, jsonField(c, "old_value"))
}

// An audit change is named by the field's client-facing name, and untagged fields are left out
// (.claude/skills/audit-events/SKILL.md, "audit struct tags"). Branding is a nested struct, so a branding
// edit must be recorded field by field rather than as one change carrying Go field names (PhoneNumber,
// LogoURL) and the untagged ID/CreatedAt/UpdatedAt, whose UpdatedAt would make saving a value the account
// already has record a change.
func TestAccountUpdate_AuditEventNamesBrandingFieldsAsClientsDo(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)
	phone := "555-0142"
	patchAccount(t, acct.owner, acct.id, map[string]any{"phone_number": phone})

	mentionsPhone := func(changes []any) bool { return strings.Contains(fmt.Sprint(changes), phone) }
	changes := accountUpdateEvent(t, acct, mentionsPhone)

	rendered := fmt.Sprint(changes)
	var leaked []string
	for _, goName := range []string{"PhoneNumber", "SupportEmail", "LogoURL", "UpdatedAt", "CreatedAt"} {
		if strings.Contains(rendered, goName+":") {
			leaked = append(leaked, goName)
		}
	}
	assert.Empty(t, leaked, "audit changes carry Go struct fields: %v", changes)
	_, flat := changeForField(changes, "phone_number")
	_, dotted := changeForField(changes, "branding.phone_number")
	assert.True(t, flat || dotted, "the phone change is recorded under its client-facing name: %v", changes)

	// Saving the phone it already has alongside a rename: the event must record the rename alone.
	newName := uniqueName("E2E Audited")
	patchAccount(t, acct.owner, acct.id, map[string]any{"phone_number": phone, "name": newName})
	changes = accountUpdateEvent(t, acct, func(changes []any) bool {
		c, ok := changeForField(changes, "name")
		return ok && jsonField(c, "new_value") == newName
	})
	for _, item := range changes {
		c, _ := item.(map[string]any)
		assert.Equal(t, "name", jsonField(c, "field"), "an unchanged phone number must not be recorded as a change: %v", c)
	}
}

// Registration creates an account_branding row, but the seeded tenant B, like every account made before
// branding existed, has none. A branding update there must be saved, not dropped while the response
// reports success.
func TestAccountUpdate_AnAccountWithoutBrandingKeepsTheUpdate(t *testing.T) {
	t.Parallel()
	tenantB := getTenantBClient()
	email := strings.ToLower(uniqueName("e2e-tenant-b")) + "@e2e-test.openmrp.ai"
	t.Cleanup(func() {
		_, _, _ = tenantB.Patch(accountsPath+"/"+SeedTenantBAccountID, map[string]any{"support_email": nil}, newIdempotencyKey())
	})

	status, body, err := tenantB.Patch(accountsPath+"/"+SeedTenantBAccountID+"?include=branding", map[string]any{"support_email": email}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", body)
	if status != 200 {
		return // refusing an account without branding is an honest answer too
	}
	assert.Equal(t, email, jsonField(jsonObject(parseJSON(body), "branding"), "support_email"),
		"a reported success must have saved the support email: %s", body)
	assert.Equal(t, email, accountValues(readAccount(t, tenantB, SeedTenantBAccountID))["support_email"],
		"the support email must be readable after a successful update")
}

// ──────────────────────────────────────────────
// Logo and favicon
// ──────────────────────────────────────────────

const (
	oneMiB = 1 << 20
	// imageUploadLimit is the largest image an upload takes.
	imageUploadLimit = 10 * oneMiB
)

// pngOfSize is a PNG signature padded to size bytes.
func pngOfSize(t *testing.T, size int) []byte {
	t.Helper()
	png := onePixelPNG(t)
	return append(png, bytes.Repeat([]byte{0}, size-len(png))...)
}

// oversizedPNG is a PNG one byte over the image upload limit.
func oversizedPNG(t *testing.T) []byte {
	t.Helper()
	return pngOfSize(t, imageUploadLimit+1)
}

// assertSignsAccountAsset checks a branding URL is one a browser can load: an absolute URL to the
// object the upload wrote, matching what the asset's own read endpoint hands out.
func assertSignsAccountAsset(t *testing.T, acct e2eAccount, brandingURL, assetPath, object string) {
	t.Helper()
	require.NotEmpty(t, brandingURL, "branding carries the uploaded %s", object)
	assert.True(t, strings.HasPrefix(brandingURL, "http://") || strings.HasPrefix(brandingURL, "https://"),
		"branding must hand out a loadable URL, not the stored key: %q", brandingURL)
	assert.True(t, strings.HasSuffix(signedObjectPath(t, brandingURL), "/"+acct.id+"/"+object),
		"the URL points at the uploaded object: %q", brandingURL)

	status, body, err := acct.owner.GetListRaw(acct.path()+"/"+assetPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, signedObjectPath(t, jsonField(parseJSON(body), "url")), signedObjectPath(t, brandingURL),
		"branding and the %s endpoint point at the same object", assetPath)
}

// The settings page shows the logo from the account's branding, so after an upload branding must carry
// a URL a browser can load rather than the storage key. An update to other fields keeps it.
func TestAccountLogo_UploadSurfacesAsABrandingLogoURL(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)
	assertNilField(t, jsonObject(readAccount(t, acct.owner, acct.id), "branding"), "logo_url")

	status, body, err := acct.owner.PutBytes(acct.path()+"/photo", "image/png", onePixelPNG(t))
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "true", jsonField(parseJSON(body), "success"))

	branding := jsonObject(readAccount(t, acct.owner, acct.id), "branding")
	assertSignsAccountAsset(t, acct, jsonField(branding, "logo_url"), "logo", "logo.png")

	updated := jsonObject(patchAccount(t, acct.owner, acct.id, map[string]any{"phone_number": "555-0123"}), "branding")
	assertSignsAccountAsset(t, acct, jsonField(updated, "logo_url"), "logo", "logo.png")
}

func TestAccountFavicon_UploadSurfacesAsABrandingFaviconURL(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)
	assertNilField(t, jsonObject(readAccount(t, acct.owner, acct.id), "branding"), "favicon_url")

	status, body, err := acct.owner.PutBytes(acct.path()+"/favicon", "image/png", onePixelPNG(t))
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	branding := jsonObject(readAccount(t, acct.owner, acct.id), "branding")
	assertSignsAccountAsset(t, acct, jsonField(branding, "favicon_url"), "favicon", "favicon.png")

	// The public branding lookup the portal's pre-login pages use serves the same favicon.
	slug := accountValues(readAccount(t, acct.owner, acct.id))["slug"].(string)
	status, body, err = apiClient.GetListRaw("/v1/settings/branding/"+slug, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, signedObjectPath(t, jsonField(branding, "favicon_url")), signedObjectPath(t, jsonField(parseJSON(body), "favicon_url")))
}

// nonImageUploads are bodies no image endpoint should store.
var nonImageUploads = []struct {
	name, contentType string
	body              []byte
}{
	{"html", "text/html", []byte("<html><script>alert(1)</script></html>")},
	{"plain text", "text/plain", []byte("not an image")},
	{"pdf", "application/pdf", []byte("%PDF-1.4\n")},
	{"empty png", "image/png", nil},
}

// The logo and favicon are served from a public CDN under the type they were stored with, so only images
// are stored, judged by their bytes: an HTML upload labeled anything must never be served back as
// text/html.
func TestAccountLogo_RejectsAnUploadThatIsNotAnImage(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)

	for _, asset := range []string{"photo", "favicon"} {
		for _, up := range nonImageUploads {
			t.Run(asset+" "+up.name, func(t *testing.T) {
				status, body, err := acct.owner.PutBytes(acct.path()+"/"+asset, up.contentType, up.body)
				require.NoError(t, err)
				require.Less(t, status, 500, "must not 5xx: %s", body)
				assert.Contains(t, []int{400, 415}, status, "a %s upload is not an image: %s", up.name, body)
			})
		}
	}
	branding := jsonObject(readAccount(t, acct.owner, acct.id), "branding")
	assert.Nil(t, branding["logo_url"], "a refused upload leaves no logo")
	assert.Nil(t, branding["favicon_url"], "a refused upload leaves no favicon")
}

// An image over the upload limit is refused, never cut off at the limit, stored corrupt, and reported as
// a success.
func TestAccountLogo_RejectsAnOversizedUpload(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)

	for _, asset := range []string{"photo", "favicon"} {
		t.Run(asset, func(t *testing.T) {
			status, body, err := acct.owner.PutBytes(acct.path()+"/"+asset, "image/png", oversizedPNG(t))
			require.NoError(t, err)
			require.Less(t, status, 500, "must not 5xx: %s", body)
			assert.Contains(t, []int{400, 413}, status, "an image over the upload limit must be refused, not truncated: %s", body)
		})
	}
}

// Like the branding update, a logo uploaded to an account without an account_branding row must be one the
// account points at, so the logo endpoint reports it.
func TestAccountLogo_AnAccountWithoutBrandingKeepsTheUpload(t *testing.T) {
	t.Parallel()
	tenantB := getTenantBClient()
	path := accountsPath + "/" + SeedTenantBAccountID

	status, body, err := tenantB.PutBytes(path+"/photo", "image/png", onePixelPNG(t))
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", body)
	if status != 200 {
		return // refusing an account without branding is an honest answer too
	}

	status, body, err = tenantB.GetListRaw(path+"/logo", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.NotEmpty(t, jsonField(parseJSON(body), "url"), "a reported upload must be readable back: %s", body)
}
