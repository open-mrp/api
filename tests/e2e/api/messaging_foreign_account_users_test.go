//go:build e2e

package api_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// An account user of another account is, to every messaging write that names one, an account user that does not exist: nobody outside the account can be messaged, seated in a conversation or roster, or blocked.
func TestMessagingMembers_AnotherAccountsUserIsUnknown(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	foreign := SeedTenantBAccountUserID
	conv := createGroupConversation(t, owner, uniqueName("e2e-foreign-team"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")
	roster := createMessagingGroup(t, owner, uniqueName("e2e-foreign-roster"), []string{SeedAccountUser2ID}, nil)
	t.Cleanup(func() { _, _ = owner.DeleteFull(messagingGroupsPath + "/" + jsonField(roster, "id")) })

	cases := []struct {
		name, path string
		body       map[string]any
		status     int
		code       string
		param      string
	}{
		{"a direct message", conversationsPath, map[string]any{"participants": []map[string]any{{"account_user_id": foreign}}}, http.StatusUnprocessableEntity, "invalid_format", "participants"},
		{"a group member", conversationsPath, map[string]any{"title": uniqueName("e2e-foreign"), "participants": []map[string]any{{"account_user_id": SeedAccountUser2ID}, {"account_user_id": foreign}}}, http.StatusUnprocessableEntity, "invalid_format", "participants"},
		{"a group member with a role", conversationsPath, map[string]any{"title": uniqueName("e2e-foreign"), "participants": []map[string]any{{"account_user_id": foreign, "role": "admin"}}}, http.StatusUnprocessableEntity, "invalid_format", "participants[0].account_user_id"},
		{"a participant added later", conversationsPath + "/" + convID + "/participants", map[string]any{"account_user_id": foreign}, http.StatusBadRequest, "parameter_invalid", "account_user_id"},
		{"a roster member", messagingGroupsPath, map[string]any{"name": uniqueName("e2e-foreign-roster"), "member_account_user_ids": []string{foreign}}, http.StatusBadRequest, "parameter_invalid", "member_account_user_ids"},
		{"a member added to a roster", messagingGroupsPath + "/" + jsonField(roster, "id") + "/members", map[string]any{"member_type": "user", "account_user_id": foreign}, http.StatusBadRequest, "parameter_invalid", "account_user_id"},
		{"a blocked user", blocksPath, map[string]any{"blocked_account_user_id": foreign}, http.StatusBadRequest, "parameter_invalid", "blocked_account_user_id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body, err := owner.Post(tc.path, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, tc.status, status, body)
			assertErrorParam(t, requireErrorResponse(t, body, tc.code, "invalid_request_error"), tc.param)
		})
	}

	resp, err := owner.GetFull(conversationsPath+"/"+convID, participantIncludeQuery)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, resp.StatusCode, resp.Body)
	pid, _, _ := participantInfo(t, parseJSON(resp.Body), foreign)
	assert.Empty(t, pid, "the other account's user was not seated")
}
