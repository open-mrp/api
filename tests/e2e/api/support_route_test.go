//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Support-route management exercises the gateway → notification-service path that designates which
// group conversation handles a relationship's inbound support. The route's group-conversation
// participants become the deterministic recipients seated on a customer's support thread; the
// resolution/seating itself is covered by notification-service. This verifies the configuration
// surface end-to-end. Self-contained: the account-default route is cleared so the shared customer-
// support conversation in sibling tests stays unseeded.

const (
	supportRoutesSetPath   = "/v1/messaging/support-routes/actions/set"
	supportRoutesClearPath = "/v1/messaging/support-routes/actions/clear"
	supportRoutesGetPath   = "/v1/messaging/support-routes"
)

func clearDefaultSupportRoute(t *testing.T, c *Client) {
	t.Helper()
	// Best-effort: clears the account-level default so it never leaks into other tests' routing.
	c.Post(supportRoutesClearPath, map[string]any{}, newIdempotencyKey())
}

func TestSupportRoute_SetGetClear(t *testing.T) {
	owner := chatUserClient(t)
	t.Cleanup(func() { clearDefaultSupportRoute(t, owner) })

	// A group conversation is the routable support target.
	group := createGroupConversation(t, owner, uniqueName("support"), SeedAccountUser2ID)
	groupID := jsonField(group, "id")

	// Set the account-level default route to that group conversation.
	status, body, err := owner.Post(supportRoutesSetPath, map[string]any{
		"group_conversation_id": groupID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	set := parseJSON(body)
	assert.Equal(t, "support_route", jsonField(set, "object"))
	assert.Equal(t, groupID, jsonField(jsonObject(set, "group_conversation"), "id"))
	assert.Empty(t, jsonField(set, "relation_account"), "the account-level default has a null relation scope")

	// Read it back.
	getStatus, getBody, err := owner.GetListRaw(supportRoutesGetPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, groupID, jsonField(jsonObject(parseJSON(getBody), "group_conversation"), "id"))

	// Clear it; reading the route then 404s.
	clearStatus, clearBody, err := owner.Post(supportRoutesClearPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, clearStatus, clearBody)

	missingStatus, missingBody, err := owner.GetListRaw(supportRoutesGetPath, nil)
	require.NoError(t, err)
	requireStatus(t, 404, missingStatus, missingBody)
}

func TestSupportRoute_RejectsNonGroupTarget(t *testing.T) {
	owner := chatUserClient(t)
	t.Cleanup(func() { clearDefaultSupportRoute(t, owner) })

	// A DM is not a group conversation, so it can't be a support route target — validation, not 5xx.
	dm := createDM(t, owner, SeedAccountUser2ID)
	status, body, err := owner.Post(supportRoutesSetPath, map[string]any{
		"group_conversation_id": jsonField(dm, "id"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

func TestSupportRoute_RejectsUnknownConversation(t *testing.T) {
	owner := chatUserClient(t)
	t.Cleanup(func() { clearDefaultSupportRoute(t, owner) })

	// A conversation that doesn't exist in the account is a validation error, not a 5xx.
	status, body, err := owner.Post(supportRoutesSetPath, map[string]any{
		"group_conversation_id": "cv_doesnotexist0000000000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

func TestSupportRoute_RelationOverride(t *testing.T) {
	owner := chatUserClient(t)
	t.Cleanup(func() {
		clearDefaultSupportRoute(t, owner)
		owner.Post(supportRoutesClearPath, map[string]any{"relation_account_id": SeedCustomerAccountID}, newIdempotencyKey())
	})

	group := createGroupConversation(t, owner, uniqueName("support"), SeedAccountUser2ID)
	groupID := jsonField(group, "id")

	// A per-relation override is scoped to a specific customer account.
	status, body, err := owner.Post(supportRoutesSetPath, map[string]any{
		"relation_account_id":   SeedCustomerAccountID,
		"group_conversation_id": groupID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, SeedCustomerAccountID, jsonField(jsonObject(parseJSON(body), "relation_account"), "id"))

	// Read it back at that scope.
	getStatus, getBody, err := owner.GetListRaw(supportRoutesGetPath, url.Values{"relation_account_id": {SeedCustomerAccountID}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, groupID, jsonField(jsonObject(parseJSON(getBody), "group_conversation"), "id"))
}
