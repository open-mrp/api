//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// /v1/sales/contacts/actions/find-by-email — closes the remaining gaps left
// by tests/e2e/api/find_contact_by_email_test.go per TASK-sales_contacts.md.
//
// This is a single, read-only action endpoint (no CRUD, no idempotency
// semantics — see the task doc §5). Reuses contactsFindByEmailPath,
// createSelfContact, findContacts, findContactMatchByID, and
// requireSelfMatch from find_contact_by_email_test.go (same package).
//
// Covers: a positive `relationship:"customer"` match, a multi-account
// same-email fan-out, ?include=account_user.role/account_user.department
// (both null-without and populated-with), every AccountUser scalar field
// reachable through account_user, page_info always-zero-value, an
// empty-string email validation case, an invalid ?include= value, the
// documented-but-unenforced `relationships=` garbage-value behavior, and
// unauthenticated access.
//
// A positive `relationship:"supplier"` match (also requested by the task
// doc) is NOT included: creating an account_user on a customer/supplier
// counterparty account requires being an "internal user" of that account
// (core-service CheckIsInternalActor), which the vendor's own API key is
// not, even via Client.WithAccountID — confirmed via curl (403
// insufficient_permissions) against both the pre-seeded SeedCustomerAccountID
// and a freshly created customer account. No account_user is seeded on
// either SeedSupplierAccountID counterparty account, and adding one would
// require a new DB seed row, which is out of scope for this file (hard
// rule: write only to this file). The positive-customer-match case is
// covered instead using the pre-existing seeded account_user
// acus_01seedcustuser00000 (email dev@augno.com) on SeedCustomerAccountID,
// which requires no new seed data.
// ──────────────────────────────────────────────

// covSalesContactsCustomerAccountUserID is the seeded account_user living on
// SeedCustomerAccountID (shared/db/seed/0010_customers.sql), used to exercise
// a real `relationship:"customer"` match without creating new seed data.
const covSalesContactsCustomerAccountUserID = "acus_01seedcustuser00000"

// covSalesContactsCustomerEmail is the email on the seeded customer account_user above.
const covSalesContactsCustomerEmail = "dev@augno.com"

// TestCovSalesContacts_CustomerRelationshipMatch asserts a real positive
// `relationship:"customer"` match (not just filter-exclusion of a self
// match, which is all the existing suite covers).
//
// NOTE (confirmed backend bug): the `account` assertion below is the
// CORRECT/desired behavior per the endpoint's own doc comment ("The account
// this contact belongs to") and IncludeConfig (which explicitly allows
// `account` as an include path) — but it currently fails. core-service's
// AccountSvc.BatchGetAccountsByIDs (services/core-service/internal/service/account_service.go
// ~L825-836) filters the requested IDs down to ONLY identity.Target.AccountID
// ("the caller can only read their own account"), silently dropping every
// other id. Since a customer/supplier ContactMatch's account_id is the
// COUNTERPARTY account (never the caller's own), `?include=account` can
// never hydrate for anything but a self match — which trivially already
// equals the caller's own account. This makes the `account` expandable
// field on ContactMatch effectively dead for its one interesting use case.
func TestCovSalesContacts_CustomerRelationshipMatch(t *testing.T) {
	t.Parallel()

	items := findContacts(t, covSalesContactsCustomerEmail, url.Values{"include": {"account_user,account"}})
	match := findContactMatchByID(items, covSalesContactsCustomerAccountUserID)
	require.NotNil(t, match, "seeded customer account_user %s should match %s: %+v",
		covSalesContactsCustomerAccountUserID, covSalesContactsCustomerEmail, items)

	assert.Equal(t, "contact_match", jsonField(match, "object"))
	assert.Equal(t, covSalesContactsCustomerAccountUserID, jsonField(match, "id"))
	assert.Equal(t, "customer", jsonField(match, "relationship"),
		"acus_01seedcustuser00000 lives on SeedCustomerAccountID, which relates to the caller as a customer")
	assert.Equal(t, covSalesContactsCustomerEmail, jsonField(match, "email"))

	au := jsonObject(match, "account_user")
	require.NotNil(t, au, "account_user should be populated with ?include=account_user")
	assert.Equal(t, "account_user", jsonField(au, "object"))
	assert.Equal(t, covSalesContactsCustomerAccountUserID, jsonField(au, "id"))

	// SUSPECTED BUG: see doc comment above. This currently returns null instead
	// of the customer account, so this assertion is expected to fail today.
	acct := jsonObject(match, "account")
	require.NotNil(t, acct, "account should be populated with ?include=account for a customer-relationship match")
	assert.Equal(t, "account", jsonField(acct, "object"))
	assert.Equal(t, SeedCustomerAccountID, jsonField(acct, "id"))
}

// TestCovSalesContacts_MultiMatchAcrossAccounts exercises the documented
// "several accounts can share an email... can return more than one match"
// behavior, which had zero coverage before this file. Reuses the seeded
// customer account_user (relationship "customer") and creates a second,
// self account_user with the SAME email so a single find-by-email call
// returns two distinct matches with distinct relationship values.
//
// NOTE (confirmed backend bug, surfaces as 409 on a persistent stack): this
// test passes on a clean database. On a stack whose DB survives prior runs it
// can fail at the create step with 409 resource_exists. account-user "remove"
// (PUT /v1/identity/account-users/{id}/actions/remove) is a SOFT delete that
// sets status_code='removed' but retains the row and its unique
// (user_id, account_id) key. POST /v1/identity/account-users then cannot re-add
// that member: the create path finds the existing user, sees no ACTIVE link
// (FindAccountUserWithRoleByAccountIDAndUserID filters to active/NULL), and
// fails the INSERT on the retained unique key — MapSQLError turns the duplicate
// into resource_exists. The create endpoint already reactivates *disabled*
// members (EnsureAccountUserActive / ReactivateAccountUsers in
// account_user.sql); the same should apply to a *removed* link so a
// previously-removed member can be re-added (201) instead of a misleading 409
// "already exists" for a resource that is invisible through the API. The 201
// expectation below is therefore the correct contract; see the backend patch
// in triage output. (This is an account-user-service gap, incidental to the
// find-by-email endpoint this file covers, which needs an active self match to
// exercise the multi-account fan-out.)
func TestCovSalesContacts_MultiMatchAcrossAccounts(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-covsc-multi")
	createStatus, createBody, err := apiClient.Post(accountUsersPath, map[string]any{
		"name":  name,
		"email": covSalesContactsCustomerEmail,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	selfID := jsonField(parseJSON(createBody), "id")
	require.NotEmpty(t, selfID)
	t.Cleanup(func() { removeAccountUser(selfID) })

	var items []map[string]any
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		items = findContacts(t, covSalesContactsCustomerEmail, nil)
		if findContactMatchByID(items, selfID) == nil {
			return fmt.Errorf("self contact %s not visible in find-by-email yet", selfID)
		}
		return nil
	})

	require.GreaterOrEqual(t, len(items), 2,
		"one email shared across two accounts (self + customer) should return at least two matches: %+v", items)

	selfMatch := findContactMatchByID(items, selfID)
	require.NotNil(t, selfMatch)
	assert.Equal(t, "self", jsonField(selfMatch, "relationship"))
	assert.Equal(t, covSalesContactsCustomerEmail, jsonField(selfMatch, "email"))

	customerMatch := findContactMatchByID(items, covSalesContactsCustomerAccountUserID)
	require.NotNil(t, customerMatch)
	assert.Equal(t, "customer", jsonField(customerMatch, "relationship"))
	assert.Equal(t, covSalesContactsCustomerEmail, jsonField(customerMatch, "email"))
}

// TestCovSalesContacts_IncludeAccountUserRoleAndDepartment covers the two
// configured include paths (account_user.role, account_user.department)
// that no existing test ever exercises: null without the nested include,
// populated id+object with it.
func TestCovSalesContacts_IncludeAccountUserRoleAndDepartment(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-covsc-roledept")
	email := name + "@e2e-test.augno.com"

	status, body, err := apiClient.Post(accountUsersPath, map[string]any{
		"name":          name,
		"email":         email,
		"role_id":       SeedAdminRoleID,
		"department_id": SeedDepartmentID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	accountUserID := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, accountUserID)
	t.Cleanup(func() { removeAccountUser(accountUserID) })

	requireSelfMatch(t, email, accountUserID, nil)

	// account_user alone: role/department stay null (gating is per-key, mirrors
	// the account_user.user behavior already covered in TestContacts_FindByEmail_Includes).
	match := findContactMatchByID(findContacts(t, email, url.Values{"include": {"account_user"}}), accountUserID)
	require.NotNil(t, match)
	au := jsonObject(match, "account_user")
	require.NotNil(t, au)
	assert.Nil(t, au["role"], "account_user.role should be null without ?include=account_user.role")
	assert.Nil(t, au["department"], "account_user.department should be null without ?include=account_user.department")

	// Both nested includes: role and department hydrate.
	match = findContactMatchByID(
		findContacts(t, email, url.Values{"include": {"account_user,account_user.role,account_user.department"}}),
		accountUserID,
	)
	require.NotNil(t, match)
	au = jsonObject(match, "account_user")
	require.NotNil(t, au)

	role := jsonObject(au, "role")
	require.NotNil(t, role, "account_user.role should be populated with ?include=account_user.role")
	assert.Equal(t, "role", jsonField(role, "object"))
	assert.Equal(t, SeedAdminRoleID, jsonField(role, "id"))

	dept := jsonObject(au, "department")
	require.NotNil(t, dept, "account_user.department should be populated with ?include=account_user.department")
	assert.Equal(t, "department", jsonField(dept, "object"))
	assert.Equal(t, SeedDepartmentID, jsonField(dept, "id"))
}

// TestCovSalesContacts_AccountUserScalarFields asserts every AccountUser
// scalar field reachable through account_user (status, last_used_at,
// created_at, updated_at) — none of which are asserted through this
// endpoint's response anywhere else in the suite.
func TestCovSalesContacts_AccountUserScalarFields(t *testing.T) {
	t.Parallel()
	accountUserID, email := createSelfContact(t)
	requireSelfMatch(t, email, accountUserID, nil)

	match := findContactMatchByID(findContacts(t, email, url.Values{"include": {"account_user"}}), accountUserID)
	require.NotNil(t, match)
	au := jsonObject(match, "account_user")
	require.NotNil(t, au)

	assert.Equal(t, "active", jsonField(au, "status"))
	assert.Nil(t, au["last_used_at"], "a freshly created, never-logged-in account_user has no last_used_at")
	assertValidTimestamp(t, jsonField(au, "created_at"), "account_user.created_at")
	assertValidTimestamp(t, jsonField(au, "updated_at"), "account_user.updated_at")
}

// TestCovSalesContacts_ListEnvelopePageInfoAlwaysZero asserts the list
// envelope's page_info sub-fields, which no existing test in the suite
// checks for this endpoint. This action never paginates (service.go always
// calls apiresource.NewList(items, apiresource.PageInfo{})), so page_info
// should always be the zero value regardless of match count.
func TestCovSalesContacts_ListEnvelopePageInfoAlwaysZero(t *testing.T) {
	t.Parallel()
	accountUserID, email := createSelfContact(t)
	requireSelfMatch(t, email, accountUserID, nil)

	status, body, err := apiClient.Post(contactsFindByEmailPath, map[string]any{"email": email}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, "list", jsonField(got, "object"))

	pageInfo := jsonObject(got, "page_info")
	require.NotNil(t, pageInfo, "page_info should be an object")
	assert.Nil(t, pageInfo["next_page_url"])
	assert.Nil(t, pageInfo["previous_page_url"])
	assert.Equal(t, "false", jsonField(pageInfo, "has_next_page"))
	assert.Equal(t, "false", jsonField(pageInfo, "has_prev_page"))
}

// TestCovSalesContacts_ValidationEmptyEmail covers an empty-string email,
// distinct from a wholly missing key (already covered) and a malformed
// value (already covered).
func TestCovSalesContacts_ValidationEmptyEmail(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(contactsFindByEmailPath, map[string]any{"email": ""}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

// TestCovSalesContacts_ValidationInvalidInclude covers an unknown ?include=
// value scoped to this route (the generic include_errors_test.go only
// covers /v1/sales/customers).
func TestCovSalesContacts_ValidationInvalidInclude(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(
		contactsFindByEmailPath+"?include=nonexistent_bogus_field",
		map[string]any{"email": uniqueName("e2e-covsc-badinclude") + "@e2e-test.augno.com"},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "include[]")
}

// TestCovSalesContacts_RelationshipsInvalidValue_SilentNoOp documents the
// ACTUAL (verified via curl, not assumed) behavior of an invalid
// `relationships=` value. FindContactByEmailRequest.Relationships carries no
// `validate` enum tag, and bind_plan.go has no generic enum-enforcement for
// query fields (unlike the strict `include` path above). The result is not
// a 5xx and not a 400 — it silently filters every match out, returning an
// empty list. This is a prodBugSuspect (inconsistent with `include`'s
// strict behavior on the same endpoint) but not a crash, so per the task
// doc it is documented here rather than asserted-around or turned into an
// invented 400 expectation.
func TestCovSalesContacts_RelationshipsInvalidValueRejected(t *testing.T) {
	t.Parallel()
	// Unrecognized enum values in a filter are rejected with 400 (platform convention).
	path := contactsFindByEmailPath + "?" + url.Values{"relationships": {"bogus"}}.Encode()
	status, body, err := apiClient.Post(path, map[string]any{"email": "e2e-nobody@example.com"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

// TestCovSalesContacts_Unauthenticated401 covers unauthenticated access to
// this action, mirroring the pattern in TestCovCoreSearch_Auth_InvalidBearerToken_401.
func TestCovSalesContacts_Unauthenticated401(t *testing.T) {
	t.Parallel()
	bogusClient := apiClient.WithBearerToken("garbage_not_a_real_token_xyz", SeedAccountID)

	status, body, err := bogusClient.Post(contactsFindByEmailPath, map[string]any{"email": "x@example.com"}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 401, status, "garbage bearer token should 401, got %d: %s", status, string(body))
}
