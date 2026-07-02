//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// External customer-service case coverage: message-level visibility (the central safety guarantee
// that an internal note never reaches the customer), staff customer-replies, case triage/assignment,
// the support inbox, and the draft-first reply flow.

// messageBodies returns the set of message bodies in a list response.
func messageBodies(list *ListResponse) map[string]bool {
	out := map[string]bool{}
	for _, raw := range list.Data {
		if b := jsonField(parseJSON(raw), "body"); b != "" {
			out[b] = true
		}
	}
	return out
}

// openCustomerCase seeds a route, has the customer open their support case, and returns its id.
func openCustomerCase(t *testing.T, customer *Client) string {
	t.Helper()
	resp, err := customer.PostFull(supportPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	conv := parseJSON(resp.Body)
	assert.Equal(t, "customer", jsonField(conv, "audience"), "a support case is a customer-facing case")
	return jsonField(conv, "id")
}

// An internal note posted on an external case is never serialized into a customer payload, while a
// customer reply is. Staff see everything; the customer sees only customer/system messages.
func TestExternalCase_InternalNoteHiddenFromCustomer(t *testing.T) {
	seedSupportRoute(t)
	customer := getCustomerPortalClient()
	dane := chatUserClient(t)

	convID := openCustomerCase(t, customer)

	// The customer asks a question (their inbound message is customer-visible).
	customerMsg := uniqueName("how much do I owe you")
	cr, err := customer.PostFull(conversationsPath+"/"+convID+"/messages", map[string]any{
		"body":              customerMsg,
		"client_message_id": uniqueName("cmid"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, cr.StatusCode, cr.Body)
	assert.Equal(t, "external", jsonField(parseJSON(cr.Body), "visibility"))

	// Staff post an INTERNAL NOTE via the normal messages endpoint — forced internal on an external case.
	note := uniqueName("internal only check the invoice first")
	nr, err := dane.PostFull(conversationsPath+"/"+convID+"/messages", map[string]any{
		"body":              note,
		"client_message_id": uniqueName("cmid"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, nr.StatusCode, nr.Body)
	assert.Equal(t, "internal", jsonField(parseJSON(nr.Body), "visibility"), "a note on an external case is internal")

	// Staff send a customer reply via the messages endpoint with audience=customer.
	reply := uniqueName("you owe one hundred dollars")
	rr, err := dane.PostFull(conversationsPath+"/"+convID+"/messages", map[string]any{
		"body":              reply,
		"client_message_id": uniqueName("cmid"),
		"audience":          "customer",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, rr.StatusCode, rr.Body)
	assert.Equal(t, "external", jsonField(parseJSON(rr.Body), "visibility"))

	// The customer sees their own message and the reply — but NEVER the internal note.
	cmsgs, _, err := customer.GetList(conversationsPath+"/"+convID+"/messages", url.Values{"limit": {"100"}})
	require.NoError(t, err)
	customerView := messageBodies(cmsgs)
	assert.True(t, customerView[customerMsg], "the customer sees their own message")
	assert.True(t, customerView[reply], "the customer sees the staff reply")
	assert.False(t, customerView[note], "SAFETY: the internal note must never reach the customer")

	// Staff see all three.
	smsgs, _, err := dane.GetList(conversationsPath+"/"+convID+"/messages", url.Values{"limit": {"100"}})
	require.NoError(t, err)
	staffView := messageBodies(smsgs)
	assert.True(t, staffView[customerMsg])
	assert.True(t, staffView[reply])
	assert.True(t, staffView[note], "staff see the internal note")
}

// customerUnread returns the customer's unread count for their support conversation (ContactSupport
// dedups to a single conversation per customer, so the list has exactly that one row).
func customerUnread(t *testing.T, customer *Client, convID string) float64 {
	t.Helper()
	list, _, err := customer.GetList(conversationsPath, url.Values{"limit": {"50"}})
	require.NoError(t, err)
	for _, raw := range list.Data {
		c := parseJSON(raw)
		if jsonField(c, "id") == convID {
			u, _ := c["unread"].(float64)
			return u
		}
	}
	t.Fatalf("support conversation %s not found in the customer's list", convID)
	return 0
}

// Internal notes do not bump the customer's unread count: posting team-only notes leaves the
// customer's unread unchanged (only customer-visible messages count for the customer).
func TestExternalCase_CustomerUnreadIgnoresInternalNotes(t *testing.T) {
	seedSupportRoute(t)
	customer := getCustomerPortalClient()
	dane := chatUserClient(t)

	convID := openCustomerCase(t, customer)

	before := customerUnread(t, customer, convID)

	// Staff post several internal notes.
	for i := 0; i < 3; i++ {
		nr, err := dane.PostFull(conversationsPath+"/"+convID+"/messages", map[string]any{
			"body":              uniqueName("internal note"),
			"client_message_id": uniqueName("cmid"),
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, nr.StatusCode, nr.Body)
	}

	after := customerUnread(t, customer, convID)
	assert.Equal(t, before, after, "internal notes must not bump the customer's unread count")
}

// Case triage: set status, assign, and find the case in the support inbox by status.
func TestExternalCase_TriageAndInbox(t *testing.T) {
	seedSupportRoute(t)
	customer := getCustomerPortalClient()
	dane := chatUserClient(t)

	convID := openCustomerCase(t, customer)

	// Move the case to a triage lane.
	sr, err := dane.PostFull(conversationsPath+"/"+convID+"/actions/set-status", map[string]any{
		"workflow_status": "waiting_internal",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, sr.StatusCode, sr.Body)
	assert.Equal(t, "waiting_internal", jsonField(parseJSON(sr.Body), "workflow_status"))

	// Assign it to a user. assignee is expandable, so request it via ?include=.
	ar, err := dane.PostFull(conversationsPath+"/"+convID+"/actions/assign?include=assignee", map[string]any{
		"assignee_resource_type": "account_user",
		"assignee_resource_id":   SeedAccountUserID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, ar.StatusCode, ar.Body)
	assignee, _ := parseJSON(ar.Body)["assignee"].(map[string]any)
	require.NotNil(t, assignee, "the case is assigned")
	assert.Equal(t, SeedAccountUserID, jsonField(assignee, "id"))

	// The case appears in the support inbox filtered to its lane. The inbox is the conversations list
	// scoped by workflow_status (any support-inbox filter selects the external-case inbox).
	inbox, _, err := dane.GetList(conversationsPath, url.Values{"workflow_status": {"waiting_internal"}, "limit": {"100"}})
	require.NoError(t, err)
	found := false
	for _, raw := range inbox.Data {
		if jsonField(parseJSON(raw), "id") == convID {
			found = true
		}
	}
	assert.True(t, found, "the case shows in the support inbox for its status")
}

// Draft-first reply: an agent/user proposes a draft, then a human approves and sends it, which
// materializes exactly one customer-visible message.
func TestExternalCase_DraftApproveSend(t *testing.T) {
	seedSupportRoute(t)
	customer := getCustomerPortalClient()
	dane := chatUserClient(t)

	convID := openCustomerCase(t, customer)

	// Propose a customer-reply draft (not sent).
	draftBody := uniqueName("your order ships Friday")
	dr, err := dane.PostFull(conversationsPath+"/"+convID+"/messages", map[string]any{
		"mode":    "draft",
		"channel": "message",
		"body":    draftBody,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, dr.StatusCode, dr.Body)
	draft := parseJSON(dr.Body)
	assert.Equal(t, "draft", jsonField(draft, "status"))
	draftID := jsonField(draft, "id")

	// The draft is not yet visible to the customer.
	before, _, err := customer.GetList(conversationsPath+"/"+convID+"/messages", url.Values{"limit": {"100"}})
	require.NoError(t, err)
	assert.False(t, messageBodies(before)[draftBody], "an unsent draft is never visible to the customer")

	// Approve and send. The draft is promoted to a sent customer-visible message in place.
	asr, err := dane.PostFull("/v1/messaging/messages/"+draftID+"/actions/approve-send", map[string]any{
		"client_message_id": uniqueName("cmid"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, asr.StatusCode, asr.Body)
	sent := parseJSON(asr.Body)
	assert.Equal(t, "sent", jsonField(sent, "status"))
	assert.Equal(t, draftID, jsonField(sent, "id"), "approve-send promotes the draft in place")

	// The customer now sees the reply exactly once.
	after, _, err := customer.GetList(conversationsPath+"/"+convID+"/messages", url.Values{"limit": {"100"}})
	require.NoError(t, err)
	count := 0
	for _, raw := range after.Data {
		if jsonField(parseJSON(raw), "body") == draftBody {
			count++
		}
	}
	assert.Equal(t, 1, count, "approve-send materializes exactly one customer-visible message")
}

// Deterministic auto-status: case activity advances the triage lane on its own, with no manual
// set-status — a customer message → waiting on team, a staff reply → waiting on customer, a pending
// draft → needs approval, a rejected draft → back to the team. A still-untriaged case stays "new".
func TestExternalCase_AutoStatusFromActivity(t *testing.T) {
	seedSupportRoute(t)
	customer := getCustomerPortalClient()
	dane := chatUserClient(t)

	convID := openCustomerCase(t, customer)

	// True when the case currently sits in the given inbox lane.
	inLane := func(status string) bool {
		inbox, _, err := dane.GetList(conversationsPath, url.Values{"workflow_status": {status}, "limit": {"100"}})
		require.NoError(t, err)
		for _, raw := range inbox.Data {
			if jsonField(parseJSON(raw), "id") == convID {
				return true
			}
		}
		return false
	}
	customerSays := func(body string) {
		r, err := customer.PostFull(conversationsPath+"/"+convID+"/messages", map[string]any{
			"body":              body,
			"client_message_id": uniqueName("cmid"),
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, r.StatusCode, r.Body)
	}

	// Note: ContactSupport dedups to one support conversation per customer, so this case may already
	// carry a lane from an earlier test. Every assertion below follows an action that deterministically
	// sets the lane, so the starting state doesn't matter.

	// The customer's message hands the case to the team.
	customerSays(uniqueName("first question"))
	assert.True(t, inLane("waiting_internal"), "a customer message moves the case to waiting on team")

	// A staff reply hands the ball to the customer.
	rr, err := dane.PostFull(conversationsPath+"/"+convID+"/messages", map[string]any{
		"body":              uniqueName("thanks for reaching out"),
		"client_message_id": uniqueName("cmid"),
		"audience":          "customer",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, rr.StatusCode, rr.Body)
	assert.True(t, inLane("waiting_external"), "a staff reply moves the case to waiting on customer")

	// The customer writes back on an engaged case → the team owes a response again.
	customerSays(uniqueName("a follow-up question"))
	assert.True(t, inLane("waiting_internal"), "a customer reply moves an engaged case to waiting on team")

	// Proposing a draft flags the case for human approval.
	dr, err := dane.PostFull(conversationsPath+"/"+convID+"/messages", map[string]any{
		"mode":    "draft",
		"channel": "message",
		"body":    uniqueName("draft answer"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, dr.StatusCode, dr.Body)
	draftID := jsonField(parseJSON(dr.Body), "id")
	assert.True(t, inLane("needs_approval"), "a pending draft moves the case to needs approval")

	// Rejecting the only draft returns the case to the team's queue.
	rj, err := dane.PostFull("/v1/messaging/messages/"+draftID+"/actions/reject", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, rj.StatusCode, rj.Body)
	assert.True(t, inLane("waiting_internal"), "rejecting the only draft returns the case to waiting on team")
}

// Linking business records to a conversation and listing conversations by record.
func TestExternalCase_RecordLinks(t *testing.T) {
	seedSupportRoute(t)
	customer := getCustomerPortalClient()
	dane := chatUserClient(t)

	convID := openCustomerCase(t, customer)

	// Link a sales order (any id; the link is a plain reference).
	orderID := "so_" + uniqueName("rec")
	lr, err := dane.PostFull(conversationsPath+"/"+convID+"/links", map[string]any{
		"resource_type": "sales_order",
		"resource_id":   orderID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, lr.StatusCode, lr.Body)
	assert.Equal(t, "conversation_link", jsonField(parseJSON(lr.Body), "object"))

	// The link shows in the conversation's link list.
	links, _, err := dane.GetList(conversationsPath+"/"+convID+"/links", nil)
	require.NoError(t, err)
	require.NotEmpty(t, links.Data)

	// The conversation is discoverable from the record via the topic-resource anchor filter.
	byRecord, _, err := dane.GetList(conversationsPath, url.Values{
		"topic_resource_type": {"sales_order"}, "topic_resource_id": {orderID}, "limit": {"50"},
	})
	require.NoError(t, err)
	found := false
	for _, raw := range byRecord.Data {
		if jsonField(parseJSON(raw), "id") == convID {
			found = true
		}
	}
	assert.True(t, found, "the case is discoverable from the linked record")

	// Unlinking by link id removes it.
	linkID := jsonField(parseJSON(lr.Body), "id")
	del, err := dane.DeleteFull(conversationsPath + "/" + convID + "/links/" + linkID)
	require.NoError(t, err)
	requireStatus(t, 200, del.StatusCode, del.Body)

	after, _, err := dane.GetList(conversationsPath+"/"+convID+"/links", nil)
	require.NoError(t, err)
	assert.Empty(t, after.Data, "the link is gone after unlinking")
}
