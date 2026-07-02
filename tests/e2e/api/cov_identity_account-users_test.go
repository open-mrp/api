//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file closes the concrete gaps identified in the identity_account-users
// e2e coverage review: update-idempotency, clearing role_id/department_id to
// null via PATCH, the duplicate-email-same-account 409 conflict, isolating
// the "admin users cannot be locked" rule from self-lock, 404 on
// action-on-nonexistent-id, same-account preferences accept(create)/reject
// (update), invalid notification_type_code, weak-password / malformed
// email / malformed username validation, invalid role_id/department_id FK
// behavior, invalid list-filter enum values, and cursor-pagination advance.
//
// Cross-account (external-target) preferences on create/update — called out
// as untested in the task brief — turned out to be **unreachable** through
// the live API, not merely untested: see
// TestCovIdentityAccountUsers_ListCrossAccountCustomerReadBlockedByHydrationBug
// below and the accompanying notes. That single read-only test pins the
// confirmed root cause without leaving orphaned data; a polluting
// create-cross-account variant was deliberately not added (see notes
// returned by the harness) since every cross-account create leaks an
// unrecoverable account_user + user row (confirmed via direct DB
// inspection) that the HTTP-only e2e client has no way to clean up.

// ──────────────────────────────────────────────
// Update idempotency
// ──────────────────────────────────────────────

func TestCovIdentityAccountUsers_UpdateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cov-au-updidem")
	email := name + "@e2e-test.augno.com"

	createStatus, createBody, err := apiClient.Post(accountUsersPath, map[string]any{
		"name":    name,
		"email":   email,
		"role_id": SeedSalesRepRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	require.NotEmpty(t, id)
	defer removeAccountUser(id)

	newName := uniqueName("e2e-cov-au-updidem2")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Patch(accountUsersPath+"/"+id, map[string]any{"name": newName}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	status2, body2, err := apiClient.Patch(accountUsersPath+"/"+id, map[string]any{"name": newName}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)

	assert.Equal(t, jsonField(parseJSON(body1), "id"), jsonField(parseJSON(body2), "id"))
	assert.Equal(t, jsonField(parseJSON(body1), "updated_at"), jsonField(parseJSON(body2), "updated_at"),
		"replaying the same idempotency key should return the identical cached response, not perform a second update")
}

// ──────────────────────────────────────────────
// Clearing role_id / department_id via PATCH null
// ──────────────────────────────────────────────

func TestCovIdentityAccountUsers_ClearRoleAndDepartment(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cov-au-clear")
	email := name + "@e2e-test.augno.com"

	createStatus, createBody, err := apiClient.Post(accountUsersPath, map[string]any{
		"name":          name,
		"email":         email,
		"role_id":       SeedAdminRoleID,
		"department_id": SeedDepartmentID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	require.NotEmpty(t, id)
	defer removeAccountUser(id)

	// Sanity: role/department are actually set before clearing.
	getStatus, getBody, err := apiClient.GetListRaw(accountUsersPath+"/"+id, url.Values{"include": {"role,department"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	require.NotNil(t, jsonObject(got, "role"), "role should be set before clearing")
	require.NotNil(t, jsonObject(got, "department"), "department should be set before clearing")

	// Clear both via explicit null.
	patchStatus, patchBody, err := apiClient.Patch(accountUsersPath+"/"+id, map[string]any{
		"role_id":       nil,
		"department_id": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	getStatus2, getBody2, err := apiClient.GetListRaw(accountUsersPath+"/"+id, url.Values{"include": {"role,department"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus2, getBody2)
	got2 := parseJSON(getBody2)
	assertNilField(t, got2, "role")
	assertNilField(t, got2, "department")
}

// ──────────────────────────────────────────────
// Duplicate-email same-account conflict
// ──────────────────────────────────────────────

func TestCovIdentityAccountUsers_CreateDuplicateEmailSameAccountConflict(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cov-au-dupemail")
	email := name + "@e2e-test.augno.com"

	createStatus, createBody, err := apiClient.Post(accountUsersPath, map[string]any{
		"name":    name,
		"email":   email,
		"role_id": SeedSalesRepRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	require.NotEmpty(t, id)
	defer removeAccountUser(id)

	// Re-adding the same email to the same account must conflict, not create
	// a second link or a second user.
	dupStatus, dupBody, err := apiClient.Post(accountUsersPath, map[string]any{
		"name":  uniqueName("e2e-cov-au-dupemail2"),
		"email": email,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, dupStatus, dupBody)

	errObj := requireErrorResponse(t, dupBody, "resource_conflict", "invalid_request_error")
	assert.Contains(t, errObj["message"], "already linked",
		"error message should explain the user is already linked to this account")
}

// ──────────────────────────────────────────────
// Admin-lock rule isolated from self-lock
// ──────────────────────────────────────────────

func TestCovIdentityAccountUsers_StatusLockAdminNotSelfFails(t *testing.T) {
	t.Parallel()
	// SeedAdmin2AccountUserID is an admin account_user that is NOT the acting
	// identity (unlike SeedAccountUserID, which is simultaneously self and
	// admin, conflating the two rules). This isolates "admin users cannot be
	// locked" from "you cannot lock your own account".
	status, body, err := apiClient.Put(accountUsersPath+"/"+SeedAdmin2AccountUserID+"/actions/disable", nil)
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assert.Contains(t, errObj["message"], "Admin", "error should name the admin-lock rule, got: %v", errObj["message"])
}

// ──────────────────────────────────────────────
// Actions on a nonexistent account_user id
// ──────────────────────────────────────────────

func TestCovIdentityAccountUsers_ActionsOnNonexistentIDReturn404(t *testing.T) {
	t.Parallel()
	const bogusID = "acus_000000000000"

	for _, action := range []string{"activate", "disable", "remove"} {
		status, body, err := apiClient.Put(accountUsersPath+"/"+bogusID+"/actions/"+action, nil)
		require.NoError(t, err)
		assert.Equal(t, 404, status, "%s on a nonexistent account_user id should 404, got %d: %s", action, status, body)
	}
}

// ──────────────────────────────────────────────
// Notification preferences — same-account behavior
// ──────────────────────────────────────────────

// TestCovIdentityAccountUsers_CreatePreferencesSameAccountSilentlyIgnored
// covers the documented, testable half of the create-preferences contract:
// "silently ignored (not rejected) when creating in your own account". The
// cross-account "applied" half is unreachable — see the top-of-file note and
// TestCovIdentityAccountUsers_ListCrossAccountCustomerReadBlockedByHydrationBug.
func TestCovIdentityAccountUsers_CreatePreferencesSameAccountSilentlyIgnored(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cov-au-prefcreate")
	email := name + "@e2e-test.augno.com"

	status, body, err := apiClient.Post(accountUsersPath, map[string]any{
		"name":  name,
		"email": email,
		"preferences": []map[string]any{
			{"notification_type": "invoice", "enabled": true},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	id := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, id)
	defer removeAccountUser(id)
}

func TestCovIdentityAccountUsers_UpdatePreferencesSameAccountRejected(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cov-au-prefupd")
	email := name + "@e2e-test.augno.com"

	createStatus, createBody, err := apiClient.Post(accountUsersPath, map[string]any{
		"name":    name,
		"email":   email,
		"role_id": SeedSalesRepRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	require.NotEmpty(t, id)
	defer removeAccountUser(id)

	status, body, err := apiClient.Patch(accountUsersPath+"/"+id, map[string]any{
		"preferences": []map[string]any{
			{"notification_type": "invoice", "enabled": true},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assert.Contains(t, errObj["message"], "external", "error should explain preferences are cross-account-only")
}

// TestCovIdentityAccountUsers_InvalidNotificationTypeCode confirms the
// service-layer notification_type_code check via the one reachable path: the
// validation runs (and fails, cleanly rolling back the transaction) before
// the create-vs-ignore branch even matters, because it is gated on
// IsExternalTarget() && len(preferences) > 0 — reachable during a
// cross-account create even though the *success* path is not observable
// through the gateway (see the hydration-bug note). Verified via direct DB
// inspection that an invalid-type-code cross-account create leaves no
// orphaned row (the error surfaces from inside the same transaction as the
// insert, unlike the hydration-bug case).
func TestCovIdentityAccountUsers_InvalidNotificationTypeCode(t *testing.T) {
	t.Parallel()
	custClient := apiClient.WithAccountID(SeedCustomerAccountID)
	name := uniqueName("e2e-cov-au-badpref")
	email := name + "@e2e-test.augno.com"

	status, body, err := custClient.Post(accountUsersPath, map[string]any{
		"name":  name,
		"email": email,
		"preferences": []map[string]any{
			{"notification_type": "bogus_notification_type", "enabled": true},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assert.Contains(t, errObj["message"], "notification type", "error should name the invalid notification type code")
}

// ──────────────────────────────────────────────
// Field-level validation: email / username / password
// ──────────────────────────────────────────────

func TestCovIdentityAccountUsers_CreateInvalidEmailFormatFails(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(accountUsersPath, map[string]any{
		"name":  uniqueName("e2e-cov-au-bademail"),
		"email": "not-a-valid-email",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "email")
}

func TestCovIdentityAccountUsers_CreateInvalidUsernameFormatFails(t *testing.T) {
	t.Parallel()
	// Username validator requires 3-255 chars, letters/digits/underscore/hyphen only.
	status, body, err := apiClient.Post(accountUsersPath, map[string]any{
		"username": "a!",
		"password": "ScannerPass123!",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "username")
}

func TestCovIdentityAccountUsers_CreateWeakPasswordFails(t *testing.T) {
	t.Parallel()
	username := uniqueName("e2e-cov-au-weakpw")

	status, body, err := apiClient.Post(accountUsersPath, map[string]any{
		"username": username,
		"password": "weak",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "password")
}

// ──────────────────────────────────────────────
// role_id / department_id FK validation
// ──────────────────────────────────────────────

func TestCovIdentityAccountUsers_CreateInvalidRoleIDFails(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(accountUsersPath, map[string]any{
		"name":    uniqueName("e2e-cov-au-badrole"),
		"email":   uniqueName("e2e-cov-au-badrole") + "@e2e-test.augno.com",
		"role_id": "rl_doesnotexist00000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

// TestCovIdentityAccountUsers_CreateInvalidDepartmentIDShouldFail documents a
// confirmed backend bug: unlike role_id (which correctly 404s for a
// nonexistent FK, see TestCovIdentityAccountUsers_CreateInvalidRoleIDFails),
// department_id is never existence-checked on create — a bogus
// department_id is silently persisted as a dangling foreign key (confirmed
// via direct DB inspection: the row is written with
// department_id='dp_doesnotexist00000' verbatim). The desired behavior,
// symmetric with role_id, is a 404. This assertion is therefore expected to
// fail against the current build (it observes 201) until the backend adds
// the same existence check department_id already lacks.
func TestCovIdentityAccountUsers_CreateInvalidDepartmentIDShouldFail(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-cov-au-baddept")
	resp, err := apiClient.PostFull(accountUsersPath, map[string]any{
		"name":          name,
		"email":         name + "@e2e-test.augno.com",
		"department_id": "dp_doesnotexist00000",
	}, newIdempotencyKey())
	require.NoError(t, err)

	// However the request lands, clean up any row it actually created before
	// asserting — the assertion below is expected to fail until the backend
	// bug is fixed (see doc comment), and cleanup must not depend on it.
	if resp.StatusCode == 201 {
		id := jsonField(parseJSON(resp.Body), "id")
		if id != "" {
			defer removeAccountUser(id)
		}
	}

	requireStatus(t, 404, resp.StatusCode, resp.Body)
}

// ──────────────────────────────────────────────
// List — invalid enum query values
// ──────────────────────────────────────────────

func TestCovIdentityAccountUsers_ListFilterInvalidRoleType(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountUsersPath, url.Values{"role_type": {"bogus_role"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

func TestCovIdentityAccountUsers_ListFilterInvalidRemovedScope(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountUsersPath, url.Values{"removed_scope": {"bogus_scope"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

// ──────────────────────────────────────────────
// List — cursor pagination advance
// ──────────────────────────────────────────────

func TestCovIdentityAccountUsers_ListCursorPaginationAdvances(t *testing.T) {
	t.Parallel()
	assertCursorPaginationAdvances(t, accountUsersPath, nil)
}

// ──────────────────────────────────────────────
// Confirmed backend bug: cross-account account-user hydration
// ──────────────────────────────────────────────

// TestCovIdentityAccountUsers_ListCrossAccountCustomerReadBlockedByHydrationBug
// pins a confirmed backend bug found while implementing the cross-account
// (?WithAccountID) preferences tests called for in the coverage task.
//
// account_user_service.checkAccountUserReadPermission (and
// checkAccountUserWritePermission) explicitly support internal actors
// managing a *customer* target account's users, gated on
// PermissionDomainCustomers — the same pattern addresses/orders/invoices
// use for cross-account access. ListAccountUsers/GetAccountUser/
// CreateAccountUser all honor this at the raw-RPC level.
//
// However, the api-gateway never returns those raw RPC results directly:
// every response (list, get, and — critically — create/update, whose
// response is re-fetched after the mutation) is hydrated through
// resourceloaders.LoadAccountUsers, which always calls the
// BatchGetAccountUsersByIDs RPC. That RPC enforces a *stricter*,
// unconditional identity.CheckIsInternalActor() gate (same-account
// membership only — see account_user_service.go's BatchGetAccountUsersByIDs)
// with no customer/supplier branch at all. The result: cross-account access
// to this endpoint group is completely blocked in practice, contradicting
// the read/write permission logic's own design and the CreateAccountUser
// preferences field's doc comment ("Only applies when creating a user in
// another account you manage (cross-account)").
//
// Confirmed via core-service logs + direct DB inspection during
// implementation: a cross-account POST to this endpoint returns HTTP 403
// ("You must be an internal user for this account to access this
// resource.") from the post-create hydration call, but CreateAccountUser
// itself already returned grpc_code=OK and committed the account_user (and,
// for a fresh email, a new user) row — an unrecoverable orphan, since every
// read path (Get/List) hits the same gate. This test uses List (read-only,
// no side effects) to pin the bug without leaking data; see the top-of-file
// note for why a create-based variant was deliberately not added.
func TestCovIdentityAccountUsers_ListCrossAccountCustomerReadBlockedByHydrationBug(t *testing.T) {
	t.Parallel()
	custClient := apiClient.WithAccountID(SeedCustomerAccountID)

	status, body, err := custClient.GetListRaw(accountUsersPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

// TestCovIdentityAccountUsers_UpdateCrossAccountBlocked pins the current,
// deliberate behavior of UpdateAccountUser: it calls
// identity.CheckIsInternalActor() unconditionally at the top (unlike
// Create/List/Get), so cross-account PATCH is always rejected regardless of
// account_relation ownership. Note this makes the "preferences applied when
// external" branch further down in UpdateAccountUser (and its accompanying
// request-field doc comment) unreachable dead code — flagged separately as
// a prodBugSuspect (doc/implementation mismatch), not asserted here since
// unlike the List/Get/Create case this restriction is applied consistently
// with role/department/user management elsewhere and may be intentional.
func TestCovIdentityAccountUsers_UpdateCrossAccountBlocked(t *testing.T) {
	t.Parallel()
	custClient := apiClient.WithAccountID(SeedCustomerAccountID)

	status, body, err := custClient.Patch(accountUsersPath+"/"+SeedAccountUserID, map[string]any{
		"name": "Cross Account Update Attempt",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
}
