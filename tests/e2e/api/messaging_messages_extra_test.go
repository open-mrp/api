//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Gap coverage for message send: threaded replies (reply_to_message_id),
// resource links (link_resource_type/id), and non-participant send rejection.

func TestChat_ReplyToThreadsMessages(t *testing.T) {
	user := chatUserClient(t)
	convID := jsonField(createDM(t, user, SeedAccountUser2ID), "id")

	parent := sendMessage(t, user, convID, uniqueName("parent"), newIdempotencyKey())
	parentID := jsonField(parent, "id")

	// Send a reply and request the reply_to expansion.
	resp, err := user.PostFull(withQuery(conversationsPath+"/"+convID+"/messages", url.Values{"include": {"reply_to"}}), map[string]any{
		"body":                uniqueName("reply"),
		"client_message_id":   newIdempotencyKey(),
		"reply_to_message_id": parentID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	reply := parseJSON(resp.Body)
	replyTo := jsonObject(reply, "reply_to")
	require.NotNil(t, replyTo, "a threaded reply carries the reply_to message when ?include=reply_to")
	assert.Equal(t, parentID, jsonField(replyTo, "id"), "reply_to points at the parent message")
}

func TestChat_LinkResourceOnMessage(t *testing.T) {
	user := chatUserClient(t)
	convID := jsonField(createDM(t, user, SeedAccountUser2ID), "id")

	resp, err := user.PostFull(withQuery(conversationsPath+"/"+convID+"/messages", url.Values{"include": {"resource"}}), map[string]any{
		"body":               uniqueName("see order"),
		"client_message_id":  newIdempotencyKey(),
		"link_resource_type": "sales_order",
		"link_resource_id":   SeedSalesOrderID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	msg := parseJSON(resp.Body)
	resource := jsonObject(msg, "resource")
	require.NotNil(t, resource, "a linked message carries the resource when ?include=resource")
	assert.Equal(t, SeedSalesOrderID, jsonField(resource, "id"), "the linked resource id round-trips")
}

func TestChat_NonParticipantCannotSend(t *testing.T) {
	user := chatUserClient(t)
	convID := jsonField(createDM(t, user, SeedAccountUser2ID), "id")

	// The customer API key is not a participant → send must be refused without leaking existence.
	customer := apiClient.WithBearerToken(SeedCustomerAPIKey, SeedAccountID)
	status, body, err := customer.Post(conversationsPath+"/"+convID+"/messages",
		map[string]any{"body": "intrusion", "client_message_id": newIdempotencyKey()}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Contains(t, []int{403, 404}, status,
		"a non-participant must not send to the conversation, got %d: %s", status, string(body))
}
