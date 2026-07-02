//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase-6a coverage: a customer (relation actor, no account_user) opens a support conversation,
// posts to it, staff lazy-join and reply anonymized, and the customer sees the reply with the real
// author stripped. Customers are confined to their own support conversation.

const supportPath = "/v1/messaging/support"
const supportAvailabilityPath = "/v1/messaging/support-availability"

// seedSupportRoute configures SeedAccountID's account-default support route to point at a freshly
// created staff group conversation, so a customer's ContactSupport resolves recipients and is allowed
// to provision a thread (support is refused when no route is configured). Clears the route on cleanup
// so it doesn't leak into tests that assume support is unconfigured.
func seedSupportRoute(t *testing.T) {
	t.Helper()
	owner := chatUserClient(t)
	group := createGroupConversation(t, owner, uniqueName("support"), SeedAccountUser2ID)
	groupID := jsonField(group, "id")
	status, body, err := owner.Post(supportRoutesSetPath, map[string]any{"group_conversation_id": groupID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	t.Cleanup(func() { owner.Post(supportRoutesClearPath, map[string]any{}, newIdempotencyKey()) })
}

func TestCustomerSupport_ProvisionDedupAndPost(t *testing.T) {
	seedSupportRoute(t)
	customer := getCustomerPortalClient()

	// First contact provisions the support conversation.
	resp, err := customer.PostFull(supportPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	conv := parseJSON(resp.Body)
	assert.Equal(t, "conversation", jsonField(conv, "object"))
	assert.Equal(t, "group", jsonField(conv, "type"))
	assert.Equal(t, "customer", jsonField(conv, "audience"))
	convID := jsonField(conv, "id")
	assertIDFormat(t, convID, "cv")

	// Repeat contact is deduped to the same conversation.
	resp2, err := customer.PostFull(supportPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp2.StatusCode, resp2.Body)
	assert.Equal(t, convID, jsonField(parseJSON(resp2.Body), "id"), "contact support is deduped per customer")

	// The customer can post to their support conversation.
	body := uniqueName("help please")
	sendResp, err := customer.PostFull(conversationsPath+"/"+convID+"/messages", map[string]any{
		"body":              body,
		"client_message_id": uniqueName("cmid"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, sendResp.StatusCode, sendResp.Body)

	// And read it back from their own conversation.
	msgs, _, err := customer.GetList(conversationsPath+"/"+convID+"/messages", url.Values{"limit": {"50"}})
	require.NoError(t, err)
	found := false
	for _, raw := range msgs.Data {
		if jsonField(parseJSON(raw), "body") == body {
			found = true
		}
	}
	assert.True(t, found, "the customer sees their own support message")
}

func TestCustomerSupport_Availability(t *testing.T) {
	customer := getCustomerPortalClient()
	owner := chatUserClient(t)

	// With no route configured, support is unavailable.
	owner.Post(supportRoutesClearPath, map[string]any{}, newIdempotencyKey()) // best-effort: ensure no leftover route
	resp, err := customer.GetFull(supportAvailabilityPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	assert.Equal(t, "support_availability", jsonField(parseJSON(resp.Body), "object"))
	assert.Equal(t, false, parseJSON(resp.Body)["available"], "unavailable when no route is configured")

	// Configuring a route makes support available.
	seedSupportRoute(t)
	resp2, err := customer.GetFull(supportAvailabilityPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, resp2.StatusCode, resp2.Body)
	assert.Equal(t, true, parseJSON(resp2.Body)["available"], "available once a route with a recipient is configured")
}

func TestCustomerSupport_CannotAccessOtherConversations(t *testing.T) {
	customer := getCustomerPortalClient()
	dane := chatUserClient(t)

	// An internal DM the customer is not part of.
	dm := createDM(t, dane, SeedAccountUser2ID)
	convID := jsonField(dm, "id")

	resp, err := customer.GetFull(conversationsPath+"/"+convID, nil)
	require.NoError(t, err)
	requireStatus(t, 404, resp.StatusCode, resp.Body) // existence not leaked to a non-participant customer
}

func TestCustomerSupport_StaffReplyAnonymizedToCustomer(t *testing.T) {
	seedSupportRoute(t)
	customer := getCustomerPortalClient()
	dane := chatUserClient(t)

	// Provision the support conversation as the customer.
	resp, err := customer.PostFull(supportPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	convID := jsonField(parseJSON(resp.Body), "id")

	// Staff (not yet a participant) reply via the customer-reply path — they lazy-join the support
	// conversation and the reply is always posted as the account's anonymizing "Customer Service"
	// persona (created on first reply). The staff member cannot choose the persona, and the generic
	// messages endpoint would post an internal note instead, which is the deliberate guardrail.
	replyBody := uniqueName("happy to help")
	staffResp, err := dane.PostFull(conversationsPath+"/"+convID+"/messages", map[string]any{
		"body":              replyBody,
		"client_message_id": uniqueName("cmid"),
		"audience":          "customer",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, staffResp.StatusCode, staffResp.Body)

	// The customer sees the staff reply attributed to the persona, with the real author stripped.
	msgs, _, err := customer.GetList(conversationsPath+"/"+convID+"/messages", url.Values{"limit": {"50"}, "include": {"sender", "author"}})
	require.NoError(t, err)
	var staffMsg map[string]any
	for _, raw := range msgs.Data {
		m := parseJSON(raw)
		if jsonField(m, "body") == replyBody {
			staffMsg = m
		}
	}
	require.NotNil(t, staffMsg, "the customer sees the staff reply")
	sender, _ := staffMsg["sender"].(map[string]any)
	require.NotNil(t, sender, "reply is attributed to the persona")
	assert.Equal(t, "group", jsonField(sender, "type"))
	assert.Nil(t, staffMsg["author"], "the real staff author is stripped for the customer viewer")
}
