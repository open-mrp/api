//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase-6b coverage: an account admin places a conversation under legal hold (blocking redaction),
// releases it, then redacts the conversation — clearing every message body while keeping the rows as
// an audit shell.

func TestRedaction_LegalHoldBlocksThenAllowsRedaction(t *testing.T) {
	admin := chatUserClient(t)

	// A DM with some content to redact.
	dm := createDM(t, admin, SeedAccountUser2ID)
	convID := jsonField(dm, "id")
	body := uniqueName("sensitive personal data")
	sendMessage(t, admin, convID, body, newIdempotencyKey())

	holdPath := conversationsPath + "/" + convID + "/actions/set-legal-hold"
	redactPath := conversationsPath + "/" + convID + "/actions/redact"

	// Place the conversation under legal hold.
	resp, err := admin.PostFull(holdPath, map[string]any{"legal_hold": "held"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	assert.Equal(t, "held", jsonField(parseJSON(resp.Body), "legal_hold"))

	// Redaction is refused while held.
	resp, err = admin.PostFull(redactPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, resp.StatusCode, resp.Body)

	// Release the hold.
	resp, err = admin.PostFull(holdPath, map[string]any{"legal_hold": "released"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	assert.Equal(t, "released", jsonField(parseJSON(resp.Body), "legal_hold"))

	// Now redaction succeeds and the conversation (audit shell) is returned.
	resp, err = admin.PostFull(redactPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	redacted := parseJSON(resp.Body)
	assert.Equal(t, convID, jsonField(redacted, "id"), "the conversation row survives redaction as an audit shell")

	// Every message body is now cleared, but the message rows remain.
	msgs, _, err := admin.GetList(conversationsPath+"/"+convID+"/messages", url.Values{"limit": {"50"}})
	require.NoError(t, err)
	require.NotEmpty(t, msgs.Data, "the message rows are kept as an audit shell")
	for _, raw := range msgs.Data {
		m := parseJSON(raw)
		assert.Nil(t, m["body"], "the message body is stripped by redaction")
	}
}

func TestRedaction_RequiresInternalActor(t *testing.T) {
	customer := getCustomerPortalClient()

	// Provision a support conversation as the customer.
	resp, err := customer.PostFull(supportPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	convID := jsonField(parseJSON(resp.Body), "id")

	// A customer cannot place their support conversation under legal hold or redact it.
	resp, err = customer.PostFull(conversationsPath+"/"+convID+"/actions/set-legal-hold", map[string]any{"legal_hold": "held"}, newIdempotencyKey())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, resp.StatusCode, 400, "legal hold is not available to customer accounts")

	resp, err = customer.PostFull(conversationsPath+"/"+convID+"/actions/redact", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, resp.StatusCode, 400, "redaction is not available to customer accounts")
}
