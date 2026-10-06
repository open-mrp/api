//go:build e2e

package api_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Creating a group with admins is one request under messaging:create: participants carries each member's starting role, applied as the owner's role change.

// chatRoleUser is a person on the seed account under a role of their own. Chat needs an account membership, which an API key does not have.
type chatRoleUser struct {
	client                       *Client
	accountUserID, roleID, email string
}

func newChatRoleUser(t *testing.T, perms ...string) chatRoleUser {
	t.Helper()
	role := createAndCleanup(t, "/v1/identity/roles", map[string]any{"name": uniqueName("e2e-chat-role"), "permissions": perms})
	_, email := covAuthPasswordsRegisterUser(t, "e2e-chat-role")
	status, body, err := apiClient.Post(accountUsersPath, map[string]any{"email": email, "role_id": jsonField(role, "id")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, status, body)
	accountUserID := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, accountUserID)
	t.Cleanup(func() { removeAccountUser(accountUserID) })
	return chatRoleUser{
		client:        loginAsUser(t, email, covAuthUsersPassword, SeedAccountID),
		accountUserID: accountUserID,
		roleID:        jsonField(role, "id"),
		email:         email,
	}
}

// customRoleChatUser is a person signed in to the seed account under a role holding exactly perms.
func customRoleChatUser(t *testing.T, perms ...string) (client *Client, accountUserID string) {
	t.Helper()
	u := newChatRoleUser(t, perms...)
	return u.client, u.accountUserID
}

func createConversationAs(t *testing.T, c *Client, body map[string]any) (int, []byte) {
	t.Helper()
	resp, err := c.PostFull(withQuery(conversationsPath, participantIncludeQuery), body, newIdempotencyKey())
	require.NoError(t, err)
	return resp.StatusCode, resp.Body
}

func TestConversationCreate_SeatsAdminsUnderMessagingCreateAlone(t *testing.T) {
	t.Parallel()
	creator, creatorID := customRoleChatUser(t, "messaging:create")

	status, body := createConversationAs(t, creator, map[string]any{
		"type":  "group",
		"title": uniqueName("e2e-roles-team"),
		"participants": []map[string]any{
			{"account_user_id": SeedAccountUser2ID, "role": "admin"},
			{"account_user_id": SeedAccountUserID, "role": "member"},
		},
	})
	requireStatus(t, http.StatusCreated, status, body)
	conv := parseJSON(body)
	convID := jsonField(conv, "id")

	_, creatorRole, _ := participantInfo(t, conv, creatorID)
	adminPID, adminRole, _ := participantInfo(t, conv, SeedAccountUser2ID)
	_, memberRole, _ := participantInfo(t, conv, SeedAccountUserID)
	assert.Equal(t, "owner", creatorRole, "the creator owns the group")
	assert.Equal(t, "admin", adminRole, "the member listed as admin starts as an admin")
	assert.Equal(t, "member", memberRole)

	// The role is announced and audited as setting it afterwards would have been.
	msgs, _, err := chatUserClient(t).GetList(conversationsPath+"/"+convID+"/messages", url.Values{"limit": {"50"}})
	require.NoError(t, err)
	var announced int
	for _, raw := range msgs.Data {
		m := parseJSON(raw)
		if jsonField(m, "kind") == "system_event" && jsonField(m, "body") == "was made an admin" {
			announced++
		}
	}
	assert.Equal(t, 1, announced, "one role change is announced in the thread")
	event := expectAuditEventWithChanges(t, adminPID, "conversation_participant", "update")
	change, ok := changeForField(jsonListData(event, "changes"), "role")
	require.True(t, ok, "the participant's update event records the role")
	assert.Equal(t, "member", jsonField(change, "old_value"))
	assert.Equal(t, "admin", jsonField(change, "new_value"))
}

func TestConversationCreate_ParticipantRoleRefusals(t *testing.T) {
	t.Parallel()
	creator, creatorID := customRoleChatUser(t, "messaging:create")
	updater, _ := customRoleChatUser(t, "messaging:update")
	group := func(participants ...map[string]any) map[string]any {
		return map[string]any{"type": "group", "title": uniqueName("e2e-roles-ref"), "participants": participants}
	}

	cases := []struct {
		name   string
		client *Client
		body   map[string]any
		status int
		code   string
		param  string
	}{
		{"an unknown role", creator, group(map[string]any{"account_user_id": SeedAccountUser2ID, "role": "boss"}), http.StatusBadRequest, "parameter_invalid", "participants[0].role"},
		{"the creator given a role", creator, group(map[string]any{"account_user_id": creatorID, "role": "admin"}), http.StatusBadRequest, "parameter_invalid", "participants[0].account_user_id"},
		{"a member listed twice", creator, group(
			map[string]any{"account_user_id": SeedAccountUser2ID, "role": "admin"},
			map[string]any{"account_user_id": SeedAccountUser2ID, "role": "viewer"},
		), http.StatusBadRequest, "parameter_invalid", "participants[1].account_user_id"},
		{"an account user that does not exist", creator, group(map[string]any{"account_user_id": "acus_01e2edoesnotexist0", "role": "admin"}), http.StatusBadRequest, "parameter_invalid", "participants[0].account_user_id"},
		{"no role", creator, group(map[string]any{"account_user_id": SeedAccountUser2ID}), http.StatusBadRequest, "", "participants[0].role"},
		{"roles on a direct message", creator, map[string]any{
			"type": "direct_message", "participant_account_user_ids": []string{SeedAccountUser2ID},
			"participants": []map[string]any{{"account_user_id": SeedAccountUser2ID, "role": "admin"}},
		}, http.StatusBadRequest, "parameter_invalid", "participants"},
	}
	for _, tc := range cases {
		status, body := createConversationAs(t, tc.client, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			requireStatus(t, tc.status, status, body)
			errObj, _ := parseJSON(body)["error"].(map[string]any)
			require.NotNil(t, errObj, string(body))
			if tc.code != "" {
				assert.Equal(t, tc.code, errObj["code"])
			}
			assertErrorParam(t, errObj, tc.param)
		})
	}

	t.Run("a role holding messaging:update alone", func(t *testing.T) {
		status, body := createConversationAs(t, updater, group(map[string]any{"account_user_id": SeedAccountUser2ID, "role": "admin"}))
		requirePermissionRefused(t, status, body, "messaging:create")
	})
}
