//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Gap-closing coverage for /v1/messaging/blocks (see
// TASK-messaging_blocks.md). Existing coverage (messaging_chat_groups_test.go
// TestChatGroup_BlockPreventsDM, messaging_chat_lifecycle_test.go
// TestChatGroup_BlockPreventsSendInExistingDM, and
// messaging_blocks_reports_extra_test.go self-block/duplicate-block/
// unblock-nonexistent) already covers: crudLifecycle end-to-end, the DM
// block-enforcement business rule (symmetric 403), and app-level create/
// delete idempotency. This file closes: allFields (every MessagingBlock
// json field asserted, blocked_user populated via ?include= and null
// without it), responseShape (id prefix + timestamp format), list envelope,
// per-field validation (missing/empty/wrong-type/nonexistent/cross-account
// target), invalid include, unsupported query params, and auth/permission
// failures (401 vs the 403 no-account-membership case). Pagination/search
// are intentionally NOT tested: ListBlocksRequest{} declares zero query
// fields and ListBlocks always returns an empty PageInfo{} — there is no
// pagination or search to exercise (na, not partial).
//
// blocksPath is declared in messaging_chat_groups_test.go; reused here, not
// redeclared.
//
// messaging_block's uniqueness is (blocker_account_user_id,
// blocked_account_user_id) and CreateBlock/DeleteBlock are unconditional
// upsert/delete on that pair (see messaging_block.sql.go), so any two
// parallel tests sharing a (blocker,target) pair can race on presence/
// absence assertions. messaging_blocks_reports_extra_test.go already owns
// the (dane, SeedAdmin2AccountUserID) pair for its duplicate-block test; to
// avoid racing with it (or with itself across this file's own parallel
// tests), each test below that needs a persistent create+list+delete cycle
// is given its own private (blocker,target) pair: admin2->dane,
// dane->child, user2->child. Tests that only assert status codes (no
// list-presence checks) safely reuse the dane->admin2 pair since a
// concurrent create/delete from the other file can't change their
// (idempotency-key-scoped or validation-only) outcome.
//
// covMessagingBlocksAdmin2Email is Mike Johnson's login (SeedAdmin2ID /
// SeedAdmin2AccountUserID in seed_test.go); no login helper for the second
// admin exists yet, so this file adds one locally rather than touching
// seed_test.go.
const covMessagingBlocksAdmin2Email = "mjohnson@openmrp.ai"

func covMessagingBlocksAdmin2Client(t *testing.T) *Client {
	t.Helper()
	return loginAsUser(t, covMessagingBlocksAdmin2Email, seedUserPassword, SeedAccountID)
}

// TestCovMessagingBlocks_CreateListDeleteAllFields uses a private pair
// (admin2/Mike blocking dane) so its create/list/expand/unblock/absence
// cycle can't race with messaging_blocks_reports_extra_test.go's
// (dane, admin2) duplicate-block test or with the shared dane<->user2 DM
// relied on by many other parallel chat tests. Sarah (SeedAccountUser2ID,
// Sales Rep) was tried first but lacks team:read, which the AccountUser
// sub-loader needs even for the un-nested blocked_user include (403
// insufficient_permissions) - both seeded admins (dane, admin2) have it.
func TestCovMessagingBlocks_CreateListDeleteAllFields(t *testing.T) {
	t.Parallel()
	user := covMessagingBlocksAdmin2Client(t)
	target := SeedAccountUserID
	t.Cleanup(func() { _, _ = user.DeleteFull(blocksPath + "/" + target) })

	createResp, err := user.PostFull(blocksPath, map[string]any{"blocked_account_user_id": target}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)
	created := parseJSON(createResp.Body)

	id := jsonField(created, "id")
	assertIDFormat(t, id, "mgbk")
	assertObjectField(t, created, "messaging_block")
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertNilField(t, created, "blocked_user")

	// GET list (no ?include=) should show the same row with blocked_user still null.
	list, status, err := user.GetList(blocksPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	assert.Equal(t, "list", list.Object)
	var foundNoInclude bool
	for _, raw := range list.Data {
		if DataItemField(raw, "id") == id {
			foundNoInclude = true
			assert.Equal(t, "", DataItemField(raw, "blocked_user"), "blocked_user should be absent/null without ?include=")
		}
	}
	assert.True(t, foundNoInclude, "created block should appear in the unfiltered list")

	// GET list with ?include=blocked_user populates the sub-object.
	listIncluded, status, err := user.GetList(blocksPath, url.Values{"include": {"blocked_user"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	var foundIncluded bool
	for _, raw := range listIncluded.Data {
		row := parseJSON(raw)
		if jsonField(row, "id") != id {
			continue
		}
		foundIncluded = true
		blockedUser := jsonObject(row, "blocked_user")
		require.NotNil(t, blockedUser, "blocked_user should be populated with ?include=blocked_user")
		assert.Equal(t, target, jsonField(blockedUser, "id"))
		assertObjectField(t, blockedUser, "account_user")
	}
	assert.True(t, foundIncluded, "created block should appear in the ?include=blocked_user list")

	// GET list with the nested include variants (blocked_user.user/.role/.department).
	listNested, status, err := user.GetList(blocksPath, url.Values{
		"include": {"blocked_user.user", "blocked_user.role", "blocked_user.department"},
	})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	var foundNested bool
	for _, raw := range listNested.Data {
		row := parseJSON(raw)
		if jsonField(row, "id") != id {
			continue
		}
		foundNested = true
		blockedUser := jsonObject(row, "blocked_user")
		require.NotNil(t, blockedUser, "blocked_user should be populated")

		nestedUser := jsonObject(blockedUser, "user")
		require.NotNil(t, nestedUser, "blocked_user.user should be populated with ?include=blocked_user.user")
		assert.NotEmpty(t, jsonField(nestedUser, "id"))
		assertObjectField(t, nestedUser, "user")

		nestedRole := jsonObject(blockedUser, "role")
		require.NotNil(t, nestedRole, "blocked_user.role should be populated with ?include=blocked_user.role")
		assert.NotEmpty(t, jsonField(nestedRole, "id"))
		assertObjectField(t, nestedRole, "role")

		nestedDept := jsonObject(blockedUser, "department")
		require.NotNil(t, nestedDept, "blocked_user.department should be populated with ?include=blocked_user.department")
		assertObjectField(t, nestedDept, "department")
	}
	assert.True(t, foundNested, "created block should appear in the nested-include list")

	// POST (create) itself also honors ?include=blocked_user on its own response, not just GET list.
	// Re-block (idempotent no-op on the already-blocked pair) with the include query param appended.
	includedCreateResp, err := user.PostFull(withQuery(blocksPath, url.Values{"include": {"blocked_user"}}),
		map[string]any{"blocked_account_user_id": target}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, includedCreateResp.StatusCode, includedCreateResp.Body)
	includedCreated := parseJSON(includedCreateResp.Body)
	includedBlockedUser := jsonObject(includedCreated, "blocked_user")
	require.NotNil(t, includedBlockedUser, "POST ?include=blocked_user should populate blocked_user on the create response")
	assert.Equal(t, target, jsonField(includedBlockedUser, "id"))
	assertObjectField(t, includedBlockedUser, "account_user")
	assert.Equal(t, id, jsonField(includedCreated, "id"), "re-blocking the same pair should return the original block id")

	// Unblock and confirm removal.
	unblockResp, err := user.DeleteFull(blocksPath + "/" + target)
	require.NoError(t, err)
	requireStatus(t, 200, unblockResp.StatusCode, unblockResp.Body)

	listAfter, status, err := user.GetList(blocksPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	for _, raw := range listAfter.Data {
		assert.NotEqual(t, id, DataItemField(raw, "id"), "unblocked row should no longer appear in the list")
	}
}

// TestCovMessagingBlocks_InvalidIncludeRejected asserts an unknown include
// token returns the standard generic parameter_invalid shape (matching the
// pattern in include_errors_test.go). No block state is mutated.
func TestCovMessagingBlocks_InvalidIncludeRejected(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)

	status, body, err := user.GetListRaw(blocksPath, url.Values{"include": {"nonsense"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

// TestCovMessagingBlocks_MissingRequiredField asserts omitting the sole
// required body field returns 400 missing_field with param naming the JSON
// field. No block state is mutated (validation fails before persistence).
func TestCovMessagingBlocks_MissingRequiredField(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)

	status, body, err := user.Post(blocksPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
	assertErrorParam(t, errObj, "blocked_account_user_id")
}

// TestCovMessagingBlocks_EmptyStringField asserts an empty-string value for
// the required field is treated the same as omitted (400 missing_field),
// matching the "required, plain string" convention (not Optional[T]).
func TestCovMessagingBlocks_EmptyStringField(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)

	status, body, err := user.Post(blocksPath, map[string]any{"blocked_account_user_id": ""}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
	assertErrorParam(t, errObj, "blocked_account_user_id")
}

// TestCovMessagingBlocks_WrongTypeField asserts a non-string value for
// blocked_account_user_id is rejected with 400 invalid_format.
func TestCovMessagingBlocks_WrongTypeField(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)

	status, body, err := user.Post(blocksPath, map[string]any{"blocked_account_user_id": 12345}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "blocked_account_user_id")
}

// Blocking a syntactically-valid but nonexistent account_user id is a 400 parameter_invalid, not a 404, and names the request field the caller sent.
func TestCovMessagingBlocks_NonexistentTargetRejected(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)

	status, body, err := user.Post(blocksPath, map[string]any{"blocked_account_user_id": "acus_doesnotexist0000"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "blocked_account_user_id")
}

// TestCovMessagingBlocks_CrossAccountTargetRecordedBehavior pins down actual
// behavior for Block() resolves the target account_user
// id via a query with no account_id filter, so a caller can plausibly
// create a messaging_block row referencing an account_user in a completely
// different account. SeedChildAccountUserID is an account_user scoped to
// SeedChildAccountID1 (a different account than SeedAccountID), and is a
// private pair (dane -> child) not touched by any other test. This test
// documents whichever branch is actually hit (success or a 4xx rejection) -
// it must not be a 5xx, and must not fabricate a populated blocked_user for
// a cross-account row. Observed on the live stack: the create succeeds
// (201) and the block row is created, but ?include=blocked_user on it
// resolves to null (the AccountUser resourceloader is account-scoped to
// the caller's account and silently finds no row) rather than erroring.
func TestCovMessagingBlocks_CrossAccountTargetRecordedBehavior(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)
	target := SeedChildAccountUserID
	t.Cleanup(func() { _, _ = user.DeleteFull(blocksPath + "/" + target) })

	resp, err := user.PostFull(blocksPath, map[string]any{"blocked_account_user_id": target}, newIdempotencyKey())
	require.NoError(t, err)
	require.NotEqual(t, 500, resp.StatusCode, "cross-account block target must not 5xx: %s", string(resp.Body))

	switch resp.StatusCode {
	case 201:
		// Currently-observed behavior: no account-scoping on target resolution, so the
		// cross-account block row is created successfully.
		created := parseJSON(resp.Body)
		assertIDFormat(t, jsonField(created, "id"), "mgbk")
		assertObjectField(t, created, "messaging_block")

		list, status, err := user.GetList(blocksPath, url.Values{"include": {"blocked_user"}})
		require.NoError(t, err)
		require.Equal(t, 200, status)
		for _, raw := range list.Data {
			if DataItemField(raw, "id") != jsonField(created, "id") {
				continue
			}
			row := parseJSON(raw)
			assert.Nil(t, row["blocked_user"],
				"cross-account blocked_user should resolve to null via the account-scoped AccountUser loader, not fabricate a populated sub-object")
		}
	case 400:
		// Alternative (safer) behavior, if some layer does reject cross-account targets.
		requireErrorResponse(t, resp.Body, "", "invalid_request_error")
	default:
		t.Fatalf("unexpected status %d for cross-account block target: %s", resp.StatusCode, string(resp.Body))
	}
}

// TestCovMessagingBlocks_UnsupportedQueryParamRejected pins down the
// undocumented behavior for query params ListBlocksRequest{} doesn't
// declare: per the generic unknown-query-param validation used elsewhere in
// this codebase, `limit` (and by extension `cursor`/`q`/`sort`) is rejected
// with 400 parameter_unknown rather than silently ignored, since the list
// endpoint has no pagination/search fields at all (na for those
// sub-features, not partial - there is nothing to page or search).
func TestCovMessagingBlocks_UnsupportedQueryParamRejected(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)

	status, body, err := user.GetListRaw(blocksPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_unknown", "invalid_request_error")
	assertErrorParam(t, errObj, "limit")
}

// TestCovMessagingBlocks_Unauthenticated asserts an empty bearer token (but
// valid OpenMRP-Version/OpenMRP-Account headers) is rejected with 401
// invalid_credentials on all three operations, matching the pattern used
// elsewhere (e.g. TestCovFinanceTransactionTypes_RequiresAuth). No block
// state is mutated (request never reaches the handler).
func TestCovMessagingBlocks_Unauthenticated(t *testing.T) {
	t.Parallel()
	unauth := apiClient.WithBearerToken("", SeedAccountID)

	status, body, err := unauth.GetListRaw(blocksPath, nil)
	require.NoError(t, err)
	requireStatus(t, 401, status, body)
	requireErrorResponse(t, body, "invalid_credentials", "invalid_request_error")

	status, body, err = unauth.Post(blocksPath, map[string]any{"blocked_account_user_id": SeedAdmin2AccountUserID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 401, status, body)
	requireErrorResponse(t, body, "invalid_credentials", "invalid_request_error")

	status, body, err = unauth.Delete(blocksPath + "/acus_neverblocked00")
	require.NoError(t, err)
	requireStatus(t, 401, status, body)
	requireErrorResponse(t, body, "invalid_credentials", "invalid_request_error")
}

// TestCovMessagingBlocks_NoAccountMembershipForbidden asserts that a raw
// API-key caller (no resolvable account_user / account-member session) gets
// 403 insufficient_permissions on all three operations, distinct from the
// DM block-enforcement 403 (already covered elsewhere) and from the 401
// no-credentials case above. `caller()` on the notification-service side
// requires an authenticated account-member session, not a bare API key. No
// block state is mutated (caller() fails before any persistence).
func TestCovMessagingBlocks_NoAccountMembershipForbidden(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(blocksPath, nil)
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")

	status, body, err = apiClient.Post(blocksPath, map[string]any{"blocked_account_user_id": SeedAdmin2AccountUserID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")

	status, body, err = apiClient.Delete(blocksPath + "/acus_neverblocked00")
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
}

// TestCovMessagingBlocks_IdempotentReplaySameKey asserts that replaying the
// exact same create request (same body) under the same Idempotency-Key
// returns the same block id and status, per the standard header-replay
// contract in e2e-test-patterns.md §7 (the existing duplicate-block test in
// messaging_blocks_reports_extra_test.go intentionally uses two *different*
// keys to prove the stronger app-level DB-upsert idempotency; this test
// adds the literal same-key replay case). Private pair (user2 -> child) to
// avoid racing with any other test's block/unblock cycle.
func TestCovMessagingBlocks_IdempotentReplaySameKey(t *testing.T) {
	t.Parallel()
	user := chatUser2Client(t)
	target := SeedChildAccountUserID
	t.Cleanup(func() { _, _ = user.DeleteFull(blocksPath + "/" + target) })

	key := newIdempotencyKey()
	first, err := user.PostFull(blocksPath, map[string]any{"blocked_account_user_id": target}, key)
	require.NoError(t, err)
	requireStatus(t, 201, first.StatusCode, first.Body)
	firstID := jsonField(parseJSON(first.Body), "id")

	second, err := user.PostFull(blocksPath, map[string]any{"blocked_account_user_id": target}, key)
	require.NoError(t, err)
	requireStatus(t, 201, second.StatusCode, second.Body)
	assert.Equal(t, firstID, jsonField(parseJSON(second.Body), "id"), "replaying the same idempotency key should return the same block id")
}

// TestCovMessagingBlocks_IdempotencyConflictDifferentBody asserts reusing
// the same Idempotency-Key with a different request body is rejected with
// 400 idempotency_error, matching TestIdempotency_CreateConflict's pattern
// for other create endpoints. The conflicting second request is rejected by
// the idempotency-key layer before it ever reaches block persistence
// (verified against the live stack: only target1 ends up as a persisted
// row), so reusing the dane->admin2 pair for target1 here is safe even
// though messaging_blocks_reports_extra_test.go's duplicate-block test also
// touches that pair - both tests only assert status codes/error bodies, not
// list presence, so they can't observe each other's interleaving.
func TestCovMessagingBlocks_IdempotencyConflictDifferentBody(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)
	target1 := SeedAdmin2AccountUserID
	t.Cleanup(func() { _, _ = user.DeleteFull(blocksPath + "/" + target1) })

	key := newIdempotencyKey()
	first, err := user.PostFull(blocksPath, map[string]any{"blocked_account_user_id": target1}, key)
	require.NoError(t, err)
	requireStatus(t, 201, first.StatusCode, first.Body)

	second, err := user.PostFull(blocksPath, map[string]any{"blocked_account_user_id": SeedChildAccountUserID}, key)
	require.NoError(t, err)
	requireStatus(t, 400, second.StatusCode, second.Body)
	requireErrorResponse(t, second.Body, "validation_failed", "idempotency_error")
}
