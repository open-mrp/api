//go:build e2e

package api_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file closes e2e coverage gaps for the messaging_groups group
// (/v1/messaging/groups) identified during the e2e coverage audit, layered on
// top of the primary suite in messaging_groups_test.go (which covers a basic
// CRUD lifecycle plus the cross-feature conversation-seeding/snapshot tests).
// It adds: exhaustive field assertions (including the nested Actor sub-object
// and the `members` List envelope), response-shape/id-prefix checks, a
// dedicated list test plus unsupported query-param rejection, `?include=`
// rejection across multiple routes (this resource has zero wired expandable
// fields), create + add-member idempotency, the full request-body validation
// matrix, and every 404/410 not-found/already-deleted path for all 7
// operations. All requests use a logged-in user client (chatUserClient) since
// the messaging-group endpoints require an account-member user identity, not
// a bare API key.

// covMessagingGroupsFindMember returns the full member object (id, object,
// actor) within a roster's `members` list whose actor id matches actorID, or
// nil if not found. Unlike groupMemberActorIDs/groupMemberID (messaging_groups_test.go),
// this returns the whole member map so every field on it can be asserted.
func covMessagingGroupsFindMember(t *testing.T, group map[string]any, actorID string) map[string]any {
	t.Helper()
	data, _ := listData(group, "members")
	for _, raw := range data {
		m, _ := raw.(map[string]any)
		actor, _ := m["actor"].(map[string]any)
		if actor != nil && jsonField(actor, "id") == actorID {
			return m
		}
	}
	return nil
}

// covMessagingGroupsAssertTimestampNotZero asserts a timestamp field is both
// a valid RFC3339 value AND not the Go zero-value time ("0001-01-01T00:00:00Z"),
// which is a valid-but-wrong RFC3339 string that assertValidTimestamp alone
// would not catch. See TestCovMessagingGroups_CreateTimestampsNotZero.
func covMessagingGroupsAssertTimestampNotZero(t *testing.T, value, fieldName string) {
	t.Helper()
	assertValidTimestamp(t, value, fieldName)
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339Nano, value)
	}
	require.NoError(t, err)
	assert.True(t, parsed.Year() > 2000, "%s should be a real timestamp, not the Go zero value (got %q)", fieldName, value)
}

// ──────────────────────────────────────────────
// allFields + responseShape
// ──────────────────────────────────────────────

// TestCovMessagingGroups_CreateAndUpdateAllFields asserts every MessagingGroup
// and MessagingGroupMember json field (including the nested Actor and the
// `members` List envelope's object/page_info), then renames the roster and
// verifies members/created_at are preserved while updated_at advances. It
// also exercises a direct remove-member happy path (add-member's happy path
// is covered separately by TestCovMessagingGroups_AddMemberIdempotentNoop).
func TestCovMessagingGroups_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)

	name := uniqueName("cov-mg-allf")
	group := createMessagingGroup(t, owner, name, []string{SeedAccountUser2ID}, []string{SeedAgentConfigID})
	id := jsonField(group, "id")
	require.NotEmpty(t, id)
	defer owner.Delete(messagingGroupsPath + "/" + id)

	assertIDFormat(t, id, "cvgp")
	assertObjectField(t, group, "messaging_group")
	assert.Equal(t, name, jsonField(group, "name"))
	assertValidTimestamp(t, jsonField(group, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(group, "updated_at"), "updated_at")

	membersEnv := jsonObject(group, "members")
	require.NotNil(t, membersEnv, "members should always be present — it is not expandable-gated")
	assertObjectField(t, membersEnv, "list")
	pageInfo := jsonObject(membersEnv, "page_info")
	require.NotNil(t, pageInfo)
	assert.Equal(t, "false", jsonField(pageInfo, "has_next_page"))
	assert.Equal(t, "false", jsonField(pageInfo, "has_prev_page"))
	assertNilField(t, pageInfo, "next_page_url")
	assertNilField(t, pageInfo, "previous_page_url")

	memberData, ok := listData(group, "members")
	require.True(t, ok)
	require.Len(t, memberData, 2)

	userMember := covMessagingGroupsFindMember(t, group, SeedAccountUser2ID)
	require.NotNil(t, userMember, "the seeded user member should be present")
	assertIDFormat(t, jsonField(userMember, "id"), "cvgppt")
	assertObjectField(t, userMember, "messaging_group_member")
	userActor := jsonObject(userMember, "actor")
	require.NotNil(t, userActor)
	assert.Equal(t, SeedAccountUser2ID, jsonField(userActor, "id"))
	assertObjectField(t, userActor, "actor")
	assert.Equal(t, "user", jsonField(userActor, "type"))
	assert.Equal(t, "Sarah Martinez", jsonField(userActor, "name"))
	assert.Equal(t, "user2@openmrp.ai", jsonField(userActor, "handle"))
	assert.NotEmpty(t, jsonField(userActor, "avatar_url"))
	assertNilField(t, userActor, "role")

	agentMember := covMessagingGroupsFindMember(t, group, SeedAgentConfigID)
	require.NotNil(t, agentMember, "the seeded agent member should be present")
	assertIDFormat(t, jsonField(agentMember, "id"), "cvgppt")
	assertObjectField(t, agentMember, "messaging_group_member")
	agentActor := jsonObject(agentMember, "actor")
	require.NotNil(t, agentActor)
	assert.Equal(t, SeedAgentConfigID, jsonField(agentActor, "id"))
	assertObjectField(t, agentActor, "actor")
	assert.Equal(t, "agent", jsonField(agentActor, "type"))
	assertNilField(t, agentActor, "name")
	assertNilField(t, agentActor, "handle")
	assertNilField(t, agentActor, "avatar_url")
	assertNilField(t, agentActor, "role")

	// Fetch a fresh GET to obtain the canonical timestamps — the create response's created_at/
	// updated_at are affected by a known bug (see TestCovMessagingGroups_CreateTimestampsNotZero),
	// so the preservation comparisons below are anchored on GET, not the raw create body.
	getStatus, getBody, err := owner.GetListRaw(messagingGroupsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	fetched := parseJSON(getBody)
	canonicalCreatedAt := jsonField(fetched, "created_at")
	canonicalUpdatedAt := jsonField(fetched, "updated_at")
	assertValidTimestamp(t, canonicalCreatedAt, "created_at")
	assertValidTimestamp(t, canonicalUpdatedAt, "updated_at")

	newName := uniqueName("cov-mg-allf-upd")
	patch, err := owner.PatchFull(messagingGroupsPath+"/"+id, map[string]any{"name": newName}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patch.StatusCode, patch.Body)
	updated := parseJSON(patch.Body)

	assert.Equal(t, id, jsonField(updated, "id"), "id must not change")
	assertObjectField(t, updated, "messaging_group")
	assert.Equal(t, newName, jsonField(updated, "name"))
	assert.Equal(t, canonicalCreatedAt, jsonField(updated, "created_at"), "created_at should be preserved across an update")
	assertValidTimestamp(t, jsonField(updated, "updated_at"), "updated_at")
	assert.NotEqual(t, canonicalUpdatedAt, jsonField(updated, "updated_at"), "updated_at should advance on a rename")

	updatedMembers, ok := listData(updated, "members")
	require.True(t, ok)
	assert.Len(t, updatedMembers, 2, "members should be preserved by a rename")

	// Direct remove-member happy path.
	removeResp, err := owner.DeleteFull(messagingGroupsPath + "/" + id + "/members/" + jsonField(userMember, "id"))
	require.NoError(t, err)
	requireStatus(t, 200, removeResp.StatusCode, removeResp.Body)
	afterRemove := parseJSON(removeResp.Body)
	afterData, ok := listData(afterRemove, "members")
	require.True(t, ok)
	assert.Len(t, afterData, 1, "removing one member should leave the other")
	assert.Nil(t, covMessagingGroupsFindMember(t, afterRemove, SeedAccountUser2ID), "the removed member should no longer be present")
	assert.NotNil(t, covMessagingGroupsFindMember(t, afterRemove, SeedAgentConfigID), "the untouched member should still be present")
}

// TestCovMessagingGroups_ResponseShape verifies id/member-id prefix formats
// and timestamp validity on a minimal create.
func TestCovMessagingGroups_ResponseShape(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	group := createMessagingGroup(t, owner, uniqueName("cov-mg-shape"), []string{SeedAccountUser2ID}, nil)
	id := jsonField(group, "id")
	require.NotEmpty(t, id)
	defer owner.Delete(messagingGroupsPath + "/" + id)

	assertIDFormat(t, id, "cvgp")
	assertObjectField(t, group, "messaging_group")
	assertValidTimestamp(t, jsonField(group, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(group, "updated_at"), "updated_at")

	memberID := groupMemberID(t, group, SeedAccountUser2ID)
	require.NotEmpty(t, memberID)
	assertIDFormat(t, memberID, "cvgppt")
}

// TestCovMessagingGroups_CreateTimestampsNotZero asserts the create response
// carries real created_at/updated_at values.
//
// BUG (confirmed): services/notification-service/internal/service/
// messaging_group_service.go CreateMessagingGroup builds
// `group := &domain.MessagingGroup{...}` in Go without ever setting
// CreatedAt/UpdatedAt on it, and messagingGroupRepoImpl.Create
// (services/notification-service/internal/infrastructure/repository/
// messaging_group_repository.go) only issues the INSERT — it never reads the
// DB-assigned timestamp defaults back into the passed-in domain object. The
// create response (and its idempotent-replay response, since the recovery-
// point cache stores this same struct) therefore always reports
// created_at/updated_at as the Go zero value ("0001-01-01T00:00:00Z"), even
// though a subsequent GET of the very same group returns the correct
// database timestamps. This test asserts the documented/correct behavior and
// is expected to fail (red) until Create hydrates the returned domain
// object's timestamps (e.g. by re-reading the row, or having the INSERT
// return them).
func TestCovMessagingGroups_CreateTimestampsNotZero(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	group := createMessagingGroup(t, owner, uniqueName("cov-mg-ts"), nil, nil)
	id := jsonField(group, "id")
	require.NotEmpty(t, id)
	defer owner.Delete(messagingGroupsPath + "/" + id)

	covMessagingGroupsAssertTimestampNotZero(t, jsonField(group, "created_at"), "created_at")
	covMessagingGroupsAssertTimestampNotZero(t, jsonField(group, "updated_at"), "updated_at")
}

// ──────────────────────────────────────────────
// omittedFields
// ──────────────────────────────────────────────

func TestCovMessagingGroups_OmittedFields(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)

	t.Run("CreateWithOnlyName", func(t *testing.T) {
		group := createMessagingGroup(t, owner, uniqueName("cov-mg-omit"), nil, nil)
		id := jsonField(group, "id")
		require.NotEmpty(t, id)
		defer owner.Delete(messagingGroupsPath + "/" + id)

		assertObjectField(t, group, "messaging_group")
		membersEnv := jsonObject(group, "members")
		require.NotNil(t, membersEnv)
		assertObjectField(t, membersEnv, "list")
		data, ok := listData(group, "members")
		require.True(t, ok)
		assert.Empty(t, data, "no members should be seeded when both id lists are omitted")
	})

	t.Run("CreateMissingName", func(t *testing.T) {
		resp, err := owner.PostFull(messagingGroupsPath, map[string]any{}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, resp.StatusCode, resp.Body)
		errObj := requireErrorResponse(t, resp.Body, "missing_field", "invalid_request_error")
		assertErrorParam(t, errObj, "name")
	})

	t.Run("CreateEmptyName", func(t *testing.T) {
		resp, err := owner.PostFull(messagingGroupsPath, map[string]any{"name": ""}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, resp.StatusCode, resp.Body)
		errObj := requireErrorResponse(t, resp.Body, "missing_field", "invalid_request_error")
		assertErrorParam(t, errObj, "name")
	})

	t.Run("CreateWhitespaceName", func(t *testing.T) {
		resp, err := owner.PostFull(messagingGroupsPath, map[string]any{"name": "   "}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, resp.StatusCode, resp.Body)
		errObj := requireErrorResponse(t, resp.Body, "parameter_missing", "invalid_request_error")
		assertErrorParam(t, errObj, "name")
	})

	t.Run("UpdateMissingName", func(t *testing.T) {
		group := createMessagingGroup(t, owner, uniqueName("cov-mg-updmiss"), nil, nil)
		id := jsonField(group, "id")
		require.NotEmpty(t, id)
		defer owner.Delete(messagingGroupsPath + "/" + id)

		// An empty PATCH body has no fields to update at all — this is a distinct code path
		// (validation_failed, no param) from sending an explicit empty/whitespace "name".
		resp, err := owner.PatchFull(messagingGroupsPath+"/"+id, map[string]any{}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, resp.StatusCode, resp.Body)
		requireErrorResponse(t, resp.Body, "validation_failed", "invalid_request_error")
	})

	t.Run("UpdateEmptyName", func(t *testing.T) {
		group := createMessagingGroup(t, owner, uniqueName("cov-mg-updempty"), nil, nil)
		id := jsonField(group, "id")
		require.NotEmpty(t, id)
		defer owner.Delete(messagingGroupsPath + "/" + id)

		resp, err := owner.PatchFull(messagingGroupsPath+"/"+id, map[string]any{"name": ""}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, resp.StatusCode, resp.Body)
		errObj := requireErrorResponse(t, resp.Body, "missing_field", "invalid_request_error")
		assertErrorParam(t, errObj, "name")
	})

	t.Run("UpdateWhitespaceName", func(t *testing.T) {
		group := createMessagingGroup(t, owner, uniqueName("cov-mg-updws"), nil, nil)
		id := jsonField(group, "id")
		require.NotEmpty(t, id)
		defer owner.Delete(messagingGroupsPath + "/" + id)

		resp, err := owner.PatchFull(messagingGroupsPath+"/"+id, map[string]any{"name": "   "}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, resp.StatusCode, resp.Body)
		errObj := requireErrorResponse(t, resp.Body, "parameter_missing", "invalid_request_error")
		assertErrorParam(t, errObj, "name")
	})

	t.Run("UpdatePreservesMembersAndCreatedAt", func(t *testing.T) {
		group := createMessagingGroup(t, owner, uniqueName("cov-mg-preserve"), []string{SeedAccountUser2ID}, []string{SeedAgentConfigID})
		id := jsonField(group, "id")
		require.NotEmpty(t, id)
		defer owner.Delete(messagingGroupsPath + "/" + id)

		// Anchor on a fresh GET, not the raw create response — see TestCovMessagingGroups_CreateTimestampsNotZero.
		getStatus, getBody, err := owner.GetListRaw(messagingGroupsPath+"/"+id, nil)
		require.NoError(t, err)
		requireStatus(t, 200, getStatus, getBody)
		origCreatedAt := jsonField(parseJSON(getBody), "created_at")
		require.NotEmpty(t, origCreatedAt)

		newName := uniqueName("cov-mg-preserve-upd")
		resp, err := owner.PatchFull(messagingGroupsPath+"/"+id, map[string]any{"name": newName}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, resp.StatusCode, resp.Body)
		updated := parseJSON(resp.Body)

		assert.Equal(t, newName, jsonField(updated, "name"))
		assert.Equal(t, origCreatedAt, jsonField(updated, "created_at"), "created_at should be preserved")
		data, ok := listData(updated, "members")
		require.True(t, ok)
		assert.Len(t, data, 2, "members should be untouched by a rename")
	})
}

// ──────────────────────────────────────────────
// list
// ──────────────────────────────────────────────

// TestCovMessagingGroups_List is a dedicated (non-embedded) list test: it
// confirms the list envelope shape and that a freshly created roster appears,
// then confirms every unsupported query key (ListMessagingGroupsRequest is an
// empty struct — no limit/cursor/q/sort are wired) is rejected with 400
// parameter_unknown naming the offending param, rather than being silently
// ignored or crashing the endpoint.
func TestCovMessagingGroups_List(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)

	group := createMessagingGroup(t, owner, uniqueName("cov-mg-list"), nil, nil)
	id := jsonField(group, "id")
	require.NotEmpty(t, id)
	defer owner.Delete(messagingGroupsPath + "/" + id)

	list, _, err := owner.GetList(messagingGroupsPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	var found bool
	for _, raw := range list.Data {
		if DataItemField(raw, "id") == id {
			found = true
			break
		}
	}
	assert.True(t, found, "the created roster should appear in the list")

	t.Run("UnsupportedQueryParamsRejected", func(t *testing.T) {
		for _, param := range []string{"limit", "cursor", "q", "sort"} {
			status, body, err := owner.GetListRaw(messagingGroupsPath, url.Values{param: {"1"}})
			require.NoError(t, err)
			requireStatus(t, 400, status, body)
			errObj := requireErrorResponse(t, body, "parameter_unknown", "invalid_request_error")
			assertErrorParam(t, errObj, param)
		}
	})
}

// ──────────────────────────────────────────────
// expandable (na, justified): ?include= must be rejected everywhere
// ──────────────────────────────────────────────

// TestCovMessagingGroups_IncludeUnsupported asserts `?include=` is rejected
// with 400 parameter_unknown (param="include") on every route tested — this
// resource has zero expandable top-level fields and none of the 7 endpoints
// wire an IncludeConfig, so an unrecognized query key falls through to the
// generic unknown-query-param check rather than the include-specific one.
func TestCovMessagingGroups_IncludeUnsupported(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)

	group := createMessagingGroup(t, owner, uniqueName("cov-mg-incl"), nil, nil)
	id := jsonField(group, "id")
	require.NotEmpty(t, id)
	defer owner.Delete(messagingGroupsPath + "/" + id)

	assertIncludeRejected := func(t *testing.T, status int, body []byte) {
		t.Helper()
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "parameter_unknown", "invalid_request_error")
		assertErrorParam(t, errObj, "include")
	}

	t.Run("Get", func(t *testing.T) {
		status, body, err := owner.GetListRaw(messagingGroupsPath+"/"+id, url.Values{"include": {"members"}})
		require.NoError(t, err)
		assertIncludeRejected(t, status, body)
	})

	t.Run("List", func(t *testing.T) {
		status, body, err := owner.GetListRaw(messagingGroupsPath, url.Values{"include": {"members"}})
		require.NoError(t, err)
		assertIncludeRejected(t, status, body)
	})

	t.Run("Update", func(t *testing.T) {
		resp, err := owner.PatchFull(withQuery(messagingGroupsPath+"/"+id, url.Values{"include": {"members"}}),
			map[string]any{"name": uniqueName("cov-mg-incl-upd")}, newIdempotencyKey())
		require.NoError(t, err)
		assertIncludeRejected(t, resp.StatusCode, resp.Body)
	})

	t.Run("AddMember", func(t *testing.T) {
		resp, err := owner.PostFull(withQuery(messagingGroupsPath+"/"+id+"/members", url.Values{"include": {"members"}}),
			map[string]any{"member_type": "user", "account_user_id": SeedAccountUser2ID}, newIdempotencyKey())
		require.NoError(t, err)
		assertIncludeRejected(t, resp.StatusCode, resp.Body)
	})

	t.Run("Delete", func(t *testing.T) {
		other := createMessagingGroup(t, owner, uniqueName("cov-mg-incl-del"), nil, nil)
		otherID := jsonField(other, "id")
		require.NotEmpty(t, otherID)
		defer owner.Delete(messagingGroupsPath + "/" + otherID)

		resp, err := owner.DeleteFull(withQuery(messagingGroupsPath+"/"+otherID, url.Values{"include": {"members"}}))
		require.NoError(t, err)
		assertIncludeRejected(t, resp.StatusCode, resp.Body)
	})
}

// ──────────────────────────────────────────────
// idempotency
// ──────────────────────────────────────────────

// TestCovMessagingGroups_CreateIdempotent proves the service's bespoke
// recovery-point idempotency cache: replaying the same Idempotency-Key
// returns the exact cached MessagingGroup rather than minting a second one.
func TestCovMessagingGroups_CreateIdempotent(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	name := uniqueName("cov-mg-idem")
	idemKey := newIdempotencyKey()

	resp1, err := owner.PostFull(messagingGroupsPath, map[string]any{"name": name}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, resp1.StatusCode, resp1.Body)
	id1 := jsonField(parseJSON(resp1.Body), "id")
	require.NotEmpty(t, id1)
	defer owner.Delete(messagingGroupsPath + "/" + id1)

	resp2, err := owner.PostFull(messagingGroupsPath, map[string]any{"name": name}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, resp2.StatusCode, resp2.Body)
	id2 := jsonField(parseJSON(resp2.Body), "id")
	assert.Equal(t, id1, id2, "replaying the same Idempotency-Key should return the exact cached group, not create a new one")
}

// TestCovMessagingGroups_AddMemberIdempotentNoop covers the add-member
// endpoint's happy path (a genuine new membership, 201) and then asserts the
// documented duplicate-add behavior: re-adding the same (member_type,
// actor_id) pair is a benign no-op — 201, member count unchanged, same
// member id.
//
// BUG (confirmed): services/notification-service/internal/service/
// messaging_group_service.go AddMessagingGroupMember only swallows the
// duplicate-insert error when its code equals apierror.ErrorCodeResourceConflict
// ("resource_conflict"). But the actual duplicate-key error produced for a
// MySQL 1062 violation is apierror.ErrorCodeResourceExists ("resource_exists")
// — see shared/db/sql_errors.go MapSQLError, case 1062 ->
// apierror.NewResourceExistsError. Because the codes don't match, the swallow
// never triggers and a duplicate add-member call returns 409 resource_exists
// instead of the documented 201 no-op. Fix: compare against
// ErrorCodeResourceExists (or both codes) in that check. This test asserts
// the documented/correct behavior for the second call and is expected to
// fail (red) until fixed.
func TestCovMessagingGroups_AddMemberIdempotentNoop(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	group := createMessagingGroup(t, owner, uniqueName("cov-mg-dupadd"), nil, nil)
	id := jsonField(group, "id")
	require.NotEmpty(t, id)
	defer owner.Delete(messagingGroupsPath + "/" + id)

	membersPath := messagingGroupsPath + "/" + id + "/members"

	// First add: a genuine new membership — the add-member happy path.
	first, err := owner.PostFull(membersPath, map[string]any{"member_type": "user", "account_user_id": SeedAccountUser2ID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, first.StatusCode, first.Body)
	afterFirst := parseJSON(first.Body)
	firstData, ok := listData(afterFirst, "members")
	require.True(t, ok)
	require.Len(t, firstData, 1)
	firstMemberID := groupMemberID(t, afterFirst, SeedAccountUser2ID)
	require.NotEmpty(t, firstMemberID)

	// Second add: the SAME (member_type, actor_id) pair — documented as a benign no-op.
	second, err := owner.PostFull(membersPath, map[string]any{"member_type": "user", "account_user_id": SeedAccountUser2ID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, second.StatusCode, second.Body)
	afterSecond := parseJSON(second.Body)
	secondData, ok := listData(afterSecond, "members")
	require.True(t, ok)
	assert.Len(t, secondData, 1, "duplicate add-member should not create a second membership row")
	secondMemberID := groupMemberID(t, afterSecond, SeedAccountUser2ID)
	assert.Equal(t, firstMemberID, secondMemberID, "duplicate add-member should return the existing member id, not mint a new one")
}

// ──────────────────────────────────────────────
// validation
// ──────────────────────────────────────────────

func TestCovMessagingGroups_CreateValidation(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)

	t.Run("NonexistentMemberAccountUserID", func(t *testing.T) {
		resp, err := owner.PostFull(messagingGroupsPath, map[string]any{
			"name":                    uniqueName("cov-mg-badmember"),
			"member_account_user_ids": []string{"acus_doesnotexist000000"},
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, resp.StatusCode, resp.Body)
		errObj := requireErrorResponse(t, resp.Body, "parameter_invalid", "invalid_request_error")
		assertErrorParam(t, errObj, "member_account_user_ids")
	})

	t.Run("BogusAgentConfigIDAcceptedOnCreate", func(t *testing.T) {
		resp, err := owner.PostFull(messagingGroupsPath, map[string]any{
			"name":                    uniqueName("cov-mg-bogusagent"),
			"member_agent_config_ids": []string{"agcf_doesnotexist"},
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, resp.StatusCode, resp.Body)
		created := parseJSON(resp.Body)
		id := jsonField(created, "id")
		require.NotEmpty(t, id)
		defer owner.Delete(messagingGroupsPath + "/" + id)

		member := covMessagingGroupsFindMember(t, created, "agcf_doesnotexist")
		require.NotNil(t, member, "a nonexistent agent_config_id is accepted as-is (not existence-validated)")
		actor := jsonObject(member, "actor")
		assertNilField(t, actor, "name")
	})

	t.Run("DuplicateMemberIDsDeduped", func(t *testing.T) {
		resp, err := owner.PostFull(messagingGroupsPath, map[string]any{
			"name":                    uniqueName("cov-mg-dupids"),
			"member_account_user_ids": []string{SeedAccountUserID, SeedAccountUserID},
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, resp.StatusCode, resp.Body)
		created := parseJSON(resp.Body)
		id := jsonField(created, "id")
		require.NotEmpty(t, id)
		defer owner.Delete(messagingGroupsPath + "/" + id)

		data, ok := listData(created, "members")
		require.True(t, ok)
		assert.Len(t, data, 1, "duplicate ids in the same create request should be silently deduped")
	})
}

func TestCovMessagingGroups_AddMemberValidation(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	group := createMessagingGroup(t, owner, uniqueName("cov-mg-addval"), nil, nil)
	id := jsonField(group, "id")
	require.NotEmpty(t, id)
	defer owner.Delete(messagingGroupsPath + "/" + id)

	membersPath := messagingGroupsPath + "/" + id + "/members"

	t.Run("InvalidMemberType", func(t *testing.T) {
		resp, err := owner.PostFull(membersPath, map[string]any{"member_type": "robot", "account_user_id": SeedAccountUser2ID}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, resp.StatusCode, resp.Body)
		errObj := requireErrorResponse(t, resp.Body, "parameter_invalid", "invalid_request_error")
		assertErrorParam(t, errObj, "member_type")
	})

	t.Run("UserMissingAccountUserID", func(t *testing.T) {
		resp, err := owner.PostFull(membersPath, map[string]any{"member_type": "user"}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, resp.StatusCode, resp.Body)
		errObj := requireErrorResponse(t, resp.Body, "parameter_missing", "invalid_request_error")
		assertErrorParam(t, errObj, "account_user_id")
	})

	t.Run("AgentMissingAgentConfigID", func(t *testing.T) {
		resp, err := owner.PostFull(membersPath, map[string]any{"member_type": "agent"}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, resp.StatusCode, resp.Body)
		errObj := requireErrorResponse(t, resp.Body, "parameter_missing", "invalid_request_error")
		assertErrorParam(t, errObj, "agent_config_id")
	})

	t.Run("NonexistentAccountUserID", func(t *testing.T) {
		resp, err := owner.PostFull(membersPath, map[string]any{"member_type": "user", "account_user_id": "acus_doesnotexist000000"}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, resp.StatusCode, resp.Body)
		errObj := requireErrorResponse(t, resp.Body, "parameter_invalid", "invalid_request_error")
		assertErrorParam(t, errObj, "account_user_id")
	})

	t.Run("NullAccountUserID", func(t *testing.T) {
		resp, err := owner.PostFull(membersPath, map[string]any{"member_type": "user", "account_user_id": nil}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, resp.StatusCode, resp.Body)
		errObj := requireErrorResponse(t, resp.Body, "invalid_format", "invalid_request_error")
		assertErrorParam(t, errObj, "account_user_id")
	})

	t.Run("EmptyAccountUserID", func(t *testing.T) {
		resp, err := owner.PostFull(membersPath, map[string]any{"member_type": "user", "account_user_id": ""}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, resp.StatusCode, resp.Body)
		errObj := requireErrorResponse(t, resp.Body, "invalid_format", "invalid_request_error")
		assertErrorParam(t, errObj, "account_user_id")
	})

	t.Run("BogusAgentConfigIDAcceptedOnAddMember", func(t *testing.T) {
		resp, err := owner.PostFull(membersPath, map[string]any{"member_type": "agent", "agent_config_id": "agcf_doesnotexist2"}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, resp.StatusCode, resp.Body)
		updated := parseJSON(resp.Body)
		member := covMessagingGroupsFindMember(t, updated, "agcf_doesnotexist2")
		require.NotNil(t, member, "a nonexistent agent_config_id is accepted as-is on add-member too")
	})
}

// ──────────────────────────────────────────────
// 404 / 410: nonexistent group/member ids across all 7 operations
// ──────────────────────────────────────────────

func TestCovMessagingGroups_NotFound(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	const badGroupID = "cvgp_doesnotexist00000000"
	const badMemberID = "cvgppt_doesnotexist000000"

	t.Run("Get", func(t *testing.T) {
		status, body, err := owner.GetListRaw(messagingGroupsPath+"/"+badGroupID, nil)
		require.NoError(t, err)
		requireStatus(t, 404, status, body)
	})

	t.Run("Update", func(t *testing.T) {
		resp, err := owner.PatchFull(messagingGroupsPath+"/"+badGroupID, map[string]any{"name": "x"}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 404, resp.StatusCode, resp.Body)
	})

	t.Run("Delete", func(t *testing.T) {
		resp, err := owner.DeleteFull(messagingGroupsPath + "/" + badGroupID)
		require.NoError(t, err)
		requireStatus(t, 404, resp.StatusCode, resp.Body)
	})

	t.Run("AddMember", func(t *testing.T) {
		resp, err := owner.PostFull(messagingGroupsPath+"/"+badGroupID+"/members",
			map[string]any{"member_type": "user", "account_user_id": SeedAccountUser2ID}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 404, resp.StatusCode, resp.Body)
	})

	t.Run("RemoveMemberNonexistentGroup", func(t *testing.T) {
		resp, err := owner.DeleteFull(messagingGroupsPath + "/" + badGroupID + "/members/" + badMemberID)
		require.NoError(t, err)
		requireStatus(t, 404, resp.StatusCode, resp.Body)
	})

	t.Run("RemoveMemberNonexistentMemberID", func(t *testing.T) {
		group := createMessagingGroup(t, owner, uniqueName("cov-mg-404member"), nil, nil)
		id := jsonField(group, "id")
		require.NotEmpty(t, id)
		defer owner.Delete(messagingGroupsPath + "/" + id)

		resp, err := owner.DeleteFull(messagingGroupsPath + "/" + id + "/members/" + badMemberID)
		require.NoError(t, err)
		requireStatus(t, 404, resp.StatusCode, resp.Body)
	})

	t.Run("RemoveMemberCrossGroup", func(t *testing.T) {
		groupA := createMessagingGroup(t, owner, uniqueName("cov-mg-crossA"), nil, nil)
		idA := jsonField(groupA, "id")
		require.NotEmpty(t, idA)
		defer owner.Delete(messagingGroupsPath + "/" + idA)

		groupB := createMessagingGroup(t, owner, uniqueName("cov-mg-crossB"), []string{SeedAccountUser2ID}, nil)
		idB := jsonField(groupB, "id")
		require.NotEmpty(t, idB)
		defer owner.Delete(messagingGroupsPath + "/" + idB)

		memberIDInB := groupMemberID(t, groupB, SeedAccountUser2ID)
		require.NotEmpty(t, memberIDInB)

		// A member id that exists, but belongs to a different roster, must 404 on group A.
		resp, err := owner.DeleteFull(messagingGroupsPath + "/" + idA + "/members/" + memberIDInB)
		require.NoError(t, err)
		requireStatus(t, 404, resp.StatusCode, resp.Body)
	})

	t.Run("DeleteAlreadyDeletedReturns410", func(t *testing.T) {
		group := createMessagingGroup(t, owner, uniqueName("cov-mg-410"), nil, nil)
		id := jsonField(group, "id")
		require.NotEmpty(t, id)

		first, err := owner.DeleteFull(messagingGroupsPath + "/" + id)
		require.NoError(t, err)
		requireStatus(t, 200, first.StatusCode, first.Body)

		second, err := owner.DeleteFull(messagingGroupsPath + "/" + id)
		require.NoError(t, err)
		requireStatus(t, 410, second.StatusCode, second.Body)
		requireErrorResponse(t, second.Body, "resource_gone", "invalid_request_error")
	})
}
