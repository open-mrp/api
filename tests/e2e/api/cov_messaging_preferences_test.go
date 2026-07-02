//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Gap coverage for /v1/messaging/preferences: response fields never asserted
// anywhere else (email_enabled, push_enabled, created_at, updated_at,
// in_app_enabled=true), invalid-category validation, omitted-digest default,
// explicit-null-category equivalence to omission, PUT idempotency, and the
// API-key-actor (no account_user) 403-on-PUT / empty-list-on-GET paths.
//
// New tests use categories not claimed by messaging_preferences_test.go or
// messaging_preferences_extra_test.go (chat.message, order.updated) to avoid
// clobbering rows those parallel tests read immediately after writing:
// chat.mention, chat.added, agent.run_completed, agent.alert, system.broadcast.

func TestCovMessagingPreferences_AllResponseFieldsAsserted(t *testing.T) {
	t.Parallel()
	user := chatUser2Client(t)

	pref := upsertPreference(t, user, map[string]any{
		"category":       "agent.run_completed",
		"in_app_enabled": true,
		"email_enabled":  true,
		"push_enabled":   true,
		"digest":         "hourly",
	})

	assertIDFormat(t, jsonField(pref, "id"), "nfpf")
	assertObjectField(t, pref, "notification_preference")
	assert.Equal(t, "agent.run_completed", jsonField(pref, "category"))
	assert.Equal(t, "true", jsonField(pref, "in_app_enabled"), "in_app_enabled=true must round-trip in the response")
	assert.Equal(t, "true", jsonField(pref, "email_enabled"), "email_enabled must be asserted in a response")
	assert.Equal(t, "true", jsonField(pref, "push_enabled"), "push_enabled must be asserted in a response")
	assert.Equal(t, "hourly", jsonField(pref, "digest"))
	assertValidTimestamp(t, jsonField(pref, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(pref, "updated_at"), "updated_at")

	// Flip every bool to false on re-upsert (full-replace PUT) and confirm the
	// false values also round-trip distinctly from the prior true values.
	pref2 := upsertPreference(t, user, map[string]any{
		"category":       "agent.run_completed",
		"in_app_enabled": false,
		"email_enabled":  false,
		"push_enabled":   false,
		"digest":         "off",
	})
	assert.Equal(t, jsonField(pref, "id"), jsonField(pref2, "id"), "re-upsert of the same category replaces the same row")
	assert.Equal(t, "false", jsonField(pref2, "in_app_enabled"))
	assert.Equal(t, "false", jsonField(pref2, "email_enabled"))
	assert.Equal(t, "false", jsonField(pref2, "push_enabled"))
}

func TestCovMessagingPreferences_FullReplaceOmittedBoolsResetToFalse(t *testing.T) {
	t.Parallel()
	user := chatUser2Client(t)

	// Set all three booleans true.
	set := upsertPreference(t, user, map[string]any{
		"category":       "agent.alert",
		"in_app_enabled": true,
		"email_enabled":  true,
		"push_enabled":   true,
		"digest":         "daily",
	})
	assert.Equal(t, "true", jsonField(set, "in_app_enabled"))
	assert.Equal(t, "true", jsonField(set, "email_enabled"))
	assert.Equal(t, "true", jsonField(set, "push_enabled"))

	// Re-upsert the same category omitting email_enabled/push_enabled entirely: this is a
	// full-replace PUT (not a PATCH merge), so the omitted booleans reset to false.
	replaced := upsertPreference(t, user, map[string]any{
		"category":       "agent.alert",
		"in_app_enabled": true,
		"digest":         "daily",
	})
	assert.Equal(t, jsonField(set, "id"), jsonField(replaced, "id"))
	assert.Equal(t, "true", jsonField(replaced, "in_app_enabled"))
	assert.Equal(t, "false", jsonField(replaced, "email_enabled"), "full-replace PUT resets omitted email_enabled to false")
	assert.Equal(t, "false", jsonField(replaced, "push_enabled"), "full-replace PUT resets omitted push_enabled to false")
}

func TestCovMessagingPreferences_InvalidCategoryRejected(t *testing.T) {
	t.Parallel()
	user := chatUser2Client(t)

	resp, err := user.PutFull(preferencesPath, map[string]any{
		"category":       "not.a.real.category",
		"in_app_enabled": true,
		"digest":         "instant",
	})
	require.NoError(t, err)
	requireStatus(t, 400, resp.StatusCode, resp.Body)

	errObj := requireErrorResponse(t, resp.Body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "category")
}

func TestCovMessagingPreferences_OmittedDigestDefaultsToInstant(t *testing.T) {
	t.Parallel()
	user := chatUser2Client(t)

	pref := upsertPreference(t, user, map[string]any{
		"category":       "chat.added",
		"in_app_enabled": true,
	})
	assert.Equal(t, "chat.added", jsonField(pref, "category"))
	assert.Equal(t, "instant", jsonField(pref, "digest"), "omitting digest defaults server-side to instant")
}

func TestCovMessagingPreferences_ExplicitNullCategoryEqualsOmission(t *testing.T) {
	t.Parallel()
	user := chatUser2Client(t)

	// Omitted category -> global default row.
	omitted := upsertPreference(t, user, map[string]any{
		"in_app_enabled": true,
		"digest":         "instant",
	})
	assertNilField(t, omitted, "category")
	omittedID := jsonField(omitted, "id")

	// Explicit "category": null -> the same global default row.
	explicitNull := upsertPreference(t, user, map[string]any{
		"category":       nil,
		"in_app_enabled": true,
		"digest":         "instant",
	})
	assertNilField(t, explicitNull, "category")
	assert.Equal(t, omittedID, jsonField(explicitNull, "id"), "explicit null category resolves to the same global-default row as omission")

	// Explicit "category": "" (empty string) also collapses to the same global-default row.
	explicitEmpty := upsertPreference(t, user, map[string]any{
		"category":       "",
		"in_app_enabled": true,
		"digest":         "instant",
	})
	assertNilField(t, explicitEmpty, "category")
	assert.Equal(t, omittedID, jsonField(explicitEmpty, "id"), "explicit empty-string category collapses to the same global-default row")
}

func TestCovMessagingPreferences_PutIdempotentOnIdenticalPayload(t *testing.T) {
	t.Parallel()
	user := chatUser2Client(t)

	body := map[string]any{
		"category":       "system.broadcast",
		"in_app_enabled": true,
		"email_enabled":  true,
		"push_enabled":   false,
		"digest":         "daily",
	}

	first := upsertPreference(t, user, body)
	second := upsertPreference(t, user, body)

	// id and every non-timestamp field must be identical across the two identical writes.
	// updated_at is not asserted equal: this endpoint always re-writes the row on every PUT,
	// so updated_at legitimately advances between the two calls.
	assert.Equal(t, jsonField(first, "id"), jsonField(second, "id"), "id is stable across identical repeat PUTs")
	assert.Equal(t, jsonField(first, "category"), jsonField(second, "category"))
	assert.Equal(t, jsonField(first, "in_app_enabled"), jsonField(second, "in_app_enabled"))
	assert.Equal(t, jsonField(first, "email_enabled"), jsonField(second, "email_enabled"))
	assert.Equal(t, jsonField(first, "push_enabled"), jsonField(second, "push_enabled"))
	assert.Equal(t, jsonField(first, "digest"), jsonField(second, "digest"))
}

func TestCovMessagingPreferences_APIKeyActorForbiddenOnPut(t *testing.T) {
	t.Parallel()

	resp, err := apiClient.PutFull(preferencesPath, map[string]any{
		"category":       "chat.mention",
		"in_app_enabled": true,
		"digest":         "instant",
	})
	require.NoError(t, err)
	requireStatus(t, 403, resp.StatusCode, resp.Body)

	errObj := requireErrorResponse(t, resp.Body, "insufficient_permissions", "invalid_request_error")
	assert.Equal(t, "Notification preferences are only available to users with an account membership.", errObj["message"])
}

func TestCovMessagingPreferences_APIKeyActorEmptyListOnGet(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(preferencesPath, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.Equal(t, "list", list.Object)
	assertEmptyListData(t, list.Data, "an API-key actor with no account_user sees no notification preferences")
}

func TestCovMessagingPreferences_ListObjectIsList(t *testing.T) {
	t.Parallel()
	user := chatUser2Client(t)

	// Seed at least one row so the list is non-trivial, then confirm the envelope shape.
	upsertPreference(t, user, map[string]any{
		"category":       "chat.mention",
		"in_app_enabled": true,
		"digest":         "instant",
	})

	list, status, err := user.GetList(preferencesPath, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.Equal(t, "list", list.Object)
	assert.NotEmpty(t, list.Data, "the authenticated user's preference list is non-empty after an upsert")
}
