//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// /v1/messaging/contacts — read-only messaging directory (single GET, no
// create/update/delete/action endpoints; see TASK-messaging_contacts.md).
// tests/e2e/api/messaging_contacts_test.go already covers the basic
// internal-user list, q substring filtering, and the customer-portal
// "support only" case. This file closes the remaining gaps: every Actor
// json field (handle, avatar_url, role — none of which had any assertion
// anywhere in the repo), self-inclusion, ?include=role (populated + nil
// without), the non-admin (Sales Rep) hydration-degradation prod bug
// suspect, limit/cursor/q validation boundaries, unknown-query-param and
// bad-include rejection, and auth edges. This operationId is excluded from
// every generic spec-driven list/pagination/schema test (see
// spec_test.go:323's excludedListOperations), so this file plus the sibling
// is the endpoint's *only* coverage.
// ──────────────────────────────────────────────

const covMessagingContactsPath = "/v1/messaging/contacts"

// covContactActorID mirrors the sibling file's contactActorID helper under a
// collision-safe name (per the one-new-file naming rule).
func covContactActorID(contact map[string]any) string {
	return jsonField(contact, "id")
}

// TestCovMessagingContacts_AllFields_AdminSeesHydratedContact asserts every
// json field on the Actor resource in a single pass for a `user`-type
// contact, using the admin caller (chatUserClient, which bypasses the
// core-service team_users:read gate inside hydrateContacts) so avatar_url
// actually resolves. Closes remediation criterion #2 for `handle` and
// `avatar_url`: `handle` is asserted null (prod bug suspect #1 —
// contactActorFromProto never sets it, and hydrateContacts fetches the
// email but never assigns it to Handle) rather than being silently
// dropped from coverage.
func TestCovMessagingContacts_AllFields_AdminSeesHydratedContact(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)

	list, status, err := user.GetList(covMessagingContactsPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status)

	var found map[string]any
	for _, raw := range list.Data {
		row := parseJSON(raw)
		if covContactActorID(row) == SeedAccountUser2ID {
			found = row
			break
		}
	}
	require.NotNil(t, found, "the admin caller's directory must include SeedAccountUser2ID")

	assertObjectField(t, found, "actor")
	assert.Equal(t, "user", jsonField(found, "type"))
	assert.Equal(t, "Sarah Martinez", jsonField(found, "name"))
	// Prod bug suspect #1: handle is documented as the user's email for
	// `user` actors but is always null on this endpoint today.
	assertNilField(t, found, "handle")
	// hydrateContacts resolves avatar_url via core-service for the admin
	// caller (IsAdmin() bypasses the team_users:read gate).
	assert.NotEmpty(t, jsonField(found, "avatar_url"), "avatar_url should resolve for an admin caller")
	// role is expandable — nil without ?include=role.
	assertNilField(t, found, "role")
}

// TestCovMessagingContacts_SelfInclusion closes prod bug suspect #4: the
// sibling file's header comment claims the caller sees themselves in their
// own directory, but no test ever asserted it in code.
func TestCovMessagingContacts_SelfInclusion(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)

	list, status, err := user.GetList(covMessagingContactsPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status)

	var self bool
	for _, raw := range list.Data {
		row := parseJSON(raw)
		if covContactActorID(row) == SeedAccountUserID {
			self = true
			break
		}
	}
	assert.True(t, self, "the caller (SeedAccountUserID) must appear in their own contacts directory")
}

// TestCovMessagingContacts_IncludeRolePopulated asserts the `role`
// expandable field, requested via ?include=role, resolves for an admin
// caller with id/object/name (and type) matching SeedSalesRepRoleID for
// SeedAccountUser2ID (Sarah Martinez, Sales Rep).
func TestCovMessagingContacts_IncludeRolePopulated(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)

	list, status, err := user.GetList(covMessagingContactsPath, url.Values{"include": {"role"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)

	var found map[string]any
	for _, raw := range list.Data {
		row := parseJSON(raw)
		if covContactActorID(row) == SeedAccountUser2ID {
			found = row
			break
		}
	}
	require.NotNil(t, found, "SeedAccountUser2ID must be present with ?include=role")

	role := jsonObject(found, "role")
	require.NotNil(t, role, "role should be populated with ?include=role")
	assert.Equal(t, SeedSalesRepRoleID, jsonField(role, "id"))
	assert.Equal(t, "role", jsonField(role, "object"))
	assert.Equal(t, "Sales Rep", jsonField(role, "name"))
}

// TestCovMessagingContacts_RoleNilWithoutInclude asserts the expandable
// `role` field is null when ?include=role is omitted, per
// e2e-test-patterns.md §6.
func TestCovMessagingContacts_RoleNilWithoutInclude(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)

	list, status, err := user.GetList(covMessagingContactsPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.NotEmpty(t, list.Data)

	for _, raw := range list.Data {
		row := parseJSON(raw)
		assertNilField(t, row, "role")
	}
}

// TestCovMessagingContacts_NonAdminHydrationDegradation documents prod bug
// suspect #3: a non-admin internal caller (chatUser2Client, Sales Rep role,
// which per shared/db/seed/0004_auth.sql only holds sales_orders +
// messaging permissions, not team_users) fails the core-service
// team_users:read gate inside BatchGetAccountUsersByIDs. Because
// hydrateContacts swallows that failure, the endpoint still returns 200
// with avatar_url and role silently null for every contact, even with
// ?include=role. This test pins the *current* (buggy) 200-with-nulls
// behavior — it is not asserting this is correct, only that a future
// change to it is visible as an intentional test update rather than a
// silent flip.
func TestCovMessagingContacts_NonAdminHydrationDegradation(t *testing.T) {
	t.Parallel()
	user2 := chatUser2Client(t)

	list, status, err := user2.GetList(covMessagingContactsPath, url.Values{"include": {"role"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.NotEmpty(t, list.Data)

	for _, raw := range list.Data {
		row := parseJSON(raw)
		assertNilField(t, row, "avatar_url")
		assertNilField(t, row, "role")
	}
}

// TestCovMessagingContacts_LimitAcceptedButIgnored pins prod bug suspect #2:
// limit is validated (min=1,max=1000) but functionally inert server-side
// (the sqlc query hardcodes LIMIT 100, no offset). limit=1 must not
// truncate the response to 1 row, and page_info must stay the zero value
// regardless — a future accidental half-wiring of real pagination should
// show up as a test failure here, not a silent pass.
func TestCovMessagingContacts_LimitAcceptedButIgnored(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covMessagingContactsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	assert.Greater(t, len(list.Data), 1, "limit=1 should NOT truncate the result today (prod bug suspect: limit is validated but ignored server-side)")

	assert.Equal(t, "list", list.Object)
	assert.False(t, list.PageInfo.HasNextPage, "page_info.has_next_page is always false for this endpoint, even when limit is smaller than the result set")
	assert.Nil(t, list.PageInfo.NextPageURL)
	assert.False(t, list.PageInfo.HasPrevPage)
	assert.Nil(t, list.PageInfo.PreviousPageURL)
}

// TestCovMessagingContacts_LimitValidRange asserts in-range limit values
// (including the documented bounds 1 and 1000) are accepted with 200 and
// never crash — the result count is intentionally not asserted to equal
// the requested limit (see TestCovMessagingContacts_LimitAcceptedButIgnored).
func TestCovMessagingContacts_LimitValidRange(t *testing.T) {
	t.Parallel()

	for _, limit := range []string{"1", "50", "1000"} {
		limit := limit
		t.Run(limit, func(t *testing.T) {
			t.Parallel()
			list, status, err := apiClient.GetList(covMessagingContactsPath, url.Values{"limit": {limit}})
			require.NoError(t, err)
			require.Equal(t, 200, status)
			assert.Equal(t, "list", list.Object)
		})
	}
}

// TestCovMessagingContacts_LimitValidation asserts out-of-range limit
// values are rejected with 400 invalid_format, never a 5xx.
func TestCovMessagingContacts_LimitValidation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		limit string
	}{
		{"zero", "0"},
		{"negative", "-1"},
		{"tooLarge", "1001"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status, body, err := apiClient.GetListRaw(covMessagingContactsPath, url.Values{"limit": {tc.limit}})
			require.NoError(t, err)
			requireStatus(t, 400, status, body)
			errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
			assertErrorParam(t, errObj, "Limit")
		})
	}
}

// TestCovMessagingContacts_LimitNonNumeric asserts a non-numeric limit is
// rejected with the raw query-parsing error (distinct from the struct
// validation "invalid_format" errors above).
func TestCovMessagingContacts_LimitNonNumeric(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(covMessagingContactsPath, url.Values{"limit": {"abc"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "limit")
}

// TestCovMessagingContacts_QueryTooLong asserts q > 500 chars is rejected
// with 400 invalid_format.
func TestCovMessagingContacts_QueryTooLong(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(covMessagingContactsPath, url.Values{"q": {strings.Repeat("x", 501)}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "Query")
}

// TestCovMessagingContacts_CursorAcceptedButInert asserts an arbitrary,
// unvalidated cursor value is accepted (200, never rejected) and does not
// change the result set, since cursor carries no meaning on the wire for
// this endpoint (proto ListContactsRequest has no cursor field at all).
func TestCovMessagingContacts_CursorAcceptedButInert(t *testing.T) {
	t.Parallel()

	baseline, status, err := apiClient.GetList(covMessagingContactsPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status)

	withCursor, status, err := apiClient.GetList(covMessagingContactsPath, url.Values{"cursor": {"totally-arbitrary-opaque-value"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)

	require.Equal(t, len(baseline.Data), len(withCursor.Data), "an arbitrary cursor must not change the result set")
	for i := range baseline.Data {
		baseRow := parseJSON(baseline.Data[i])
		curRow := parseJSON(withCursor.Data[i])
		assert.Equal(t, covContactActorID(baseRow), covContactActorID(curRow), "row order/identity must be unaffected by cursor at index %d", i)
	}
}

// TestCovMessagingContacts_IncludeBogusRejected asserts an unrecognized
// ?include value is rejected with 400 parameter_invalid, since Actor's only
// IncludesFor entry is "role".
func TestCovMessagingContacts_IncludeBogusRejected(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(covMessagingContactsPath, url.Values{"include": {"bogus_field"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "include[]")
}

// TestCovMessagingContacts_UnknownQueryParamRejected asserts an undeclared
// query parameter is rejected — this endpoint is excluded from the generic
// spec-driven sweep that would normally catch a regression here.
func TestCovMessagingContacts_UnknownQueryParamRejected(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(covMessagingContactsPath, url.Values{bogusE2EQueryParam: {"1"}})
	require.NoError(t, err)
	assertUnknownQueryParamRejected(t, covMessagingContactsPath, status, body)
}

// TestCovMessagingContacts_CustomerSeesOnlySupport_PinnedIDAndName extends
// the sibling test's type-only check by pinning the exact id ("support" —
// a literal constant, not a generated prefix_xxxxx id, per prod bug
// suspect #5) and name ("Customer Service"), plus the remaining
// unasserted Actor fields (handle, avatar_url, role — all null for the
// shared support contact).
func TestCovMessagingContacts_CustomerSeesOnlySupport_PinnedIDAndName(t *testing.T) {
	t.Parallel()
	customer := getCustomerPortalClient()

	list, status, err := customer.GetList(covMessagingContactsPath, url.Values{"include": {"role"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, list.Data, 1, "a customer must see exactly one (support) contact")

	contact := parseJSON(list.Data[0])
	assert.Equal(t, "support", jsonField(contact, "id"), "the support contact id is a stable literal constant, not a generated prefix_xxxxx id")
	assertObjectField(t, contact, "actor")
	assert.Equal(t, "group", jsonField(contact, "type"))
	assert.Equal(t, "Customer Service", jsonField(contact, "name"))
	assertNilField(t, contact, "handle")
	assertNilField(t, contact, "avatar_url")
	assertNilField(t, contact, "role")
}

// TestCovMessagingContacts_EmptyBearerTokenUnauthorized verifies a request
// with an empty bearer token ("Authorization: Bearer ") is rejected with
// 401 invalid_credentials.
func TestCovMessagingContacts_EmptyBearerTokenUnauthorized(t *testing.T) {
	t.Parallel()

	anonClient := apiClient.WithBearerToken("", SeedAccountID)
	status, body, err := anonClient.GetListRaw(covMessagingContactsPath, nil)
	require.NoError(t, err)
	requireStatus(t, 401, status, body)
	requireErrorResponse(t, body, "invalid_credentials", "invalid_request_error")
}

// TestCovMessagingContacts_InvalidBearerTokenUnauthorized verifies a
// syntactically-invalid (non-JWT, non-API-key) bearer token is rejected
// with 401 invalid_credentials.
func TestCovMessagingContacts_InvalidBearerTokenUnauthorized(t *testing.T) {
	t.Parallel()

	badClient := apiClient.WithBearerToken("garbage_totally_invalid_token", SeedAccountID)
	status, body, err := badClient.GetListRaw(covMessagingContactsPath, nil)
	require.NoError(t, err)
	requireStatus(t, 401, status, body)
	requireErrorResponse(t, body, "invalid_credentials", "invalid_request_error")
}

// TestCovMessagingContacts_MissingAccountHeaderRejected verifies that a
// user-session caller (not an API key, which is already permanently scoped
// to an account) with an empty Augno-Account header is rejected. Live-curl
// verified: the gateway's RequiredPermissions gate (messaging:read) fails
// before the request ever reaches notification-service's own
// !IsTargetAccountSet() check, so the observed status is 403
// insufficient_permissions rather than 401 — this test pins the actually
// observed behavior rather than the speculative 401 from the task doc.
func TestCovMessagingContacts_MissingAccountHeaderRejected(t *testing.T) {
	t.Parallel()

	user := loginAsUser(t, seedUserEmail, seedUserPassword, "")
	status, body, err := user.GetListRaw(covMessagingContactsPath, nil)
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	errObj := requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
	msg, _ := errObj["message"].(string)
	assert.Contains(t, msg, "messaging:read")
}
