//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests exercise abuse reporting (§12.10): an active participant may file a minimal report
// against a conversation; a non-participant cannot (the conversation's existence is never leaked).

func reportPath(conversationID string) string {
	return conversationsPath + "/" + conversationID + "/actions/report"
}

func TestMessagingReports_ParticipantCanReport(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)
	dm := createDM(t, user, SeedAccountUser2ID)
	conversationID := jsonField(dm, "id")

	// Reporting files an abuse record server-side and returns the reported conversation.
	resp, err := user.PostFull(reportPath(conversationID), map[string]any{
		"reason": "spam",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)

	report := parseJSON(resp.Body)
	assert.Equal(t, "conversation", jsonField(report, "object"))
	assert.Equal(t, conversationID, jsonField(report, "id"))
}

func TestMessagingReports_ReportSpecificMessage(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)
	dm := createDM(t, user, SeedAccountUser2ID)
	conversationID := jsonField(dm, "id")
	msg := sendMessage(t, user, conversationID, "offensive message", newIdempotencyKey())
	messageID := jsonField(msg, "id")

	// A report may target a specific message via message_id; the reported conversation is returned.
	resp, err := user.PostFull(reportPath(conversationID), map[string]any{
		"reason":     "harassment",
		"message_id": messageID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)

	report := parseJSON(resp.Body)
	assert.Equal(t, "conversation", jsonField(report, "object"))
	assert.Equal(t, conversationID, jsonField(report, "id"))
}

func TestMessagingReports_NonParticipantGets404(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	dm := createDM(t, owner, SeedAccountUser2ID)
	conversationID := jsonField(dm, "id")

	// The customer is not a participant of this internal DM; existence must not be leaked.
	customer := getCustomerPortalClient()
	resp, err := customer.PostFull(reportPath(conversationID), map[string]any{
		"reason": "spam",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, resp.StatusCode, resp.Body)
}
