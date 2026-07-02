//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Gap coverage for blocks (self-block, duplicate, unblock-nonexistent) and
// reports (duplicate, foreign message_id).

func TestChatBlocks_SelfBlockRejected(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t) // dane = SeedAccountUserID
	status, body, err := user.Post(blocksPath, map[string]any{"blocked_account_user_id": SeedAccountUserID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

func TestChatBlocks_DuplicateBlockIsIdempotent(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)
	// Block the second admin (Mike), not user2: blocking globally severs the
	// blocker↔target DM, and the shared dane↔user2 DM is exercised by many parallel
	// chat tests. Mike's DM with dane is not used elsewhere, so this stays isolated.
	target := SeedAdmin2AccountUserID
	t.Cleanup(func() { _, _ = user.DeleteFull(blocksPath + "/" + target) })

	first, err := user.PostFull(blocksPath, map[string]any{"blocked_account_user_id": target}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Contains(t, []int{200, 201}, first.StatusCode, "first block succeeds: %s", string(first.Body))

	second, err := user.PostFull(blocksPath, map[string]any{"blocked_account_user_id": target}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Contains(t, []int{200, 201, 409}, second.StatusCode,
		"a duplicate block must not 5xx, got %d: %s", second.StatusCode, string(second.Body))
}

func TestChatBlocks_UnblockNonexistentIsIdempotent(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)
	// Unblocking a user who was never blocked is a no-op success, not an error.
	status, body, err := user.Delete(blocksPath + "/acus_neverblocked00")
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

func TestMessagingReports_DuplicateAllowed(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)
	convID := jsonField(createDM(t, user, SeedAccountUser2ID), "id")

	first, err := user.PostFull(reportPath(convID), map[string]any{"reason": "spam"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, first.StatusCode, first.Body)

	// A second, independent report (distinct idempotency key) is accepted, not deduped — each files a
	// separate abuse record server-side. The endpoint returns the reported conversation both times.
	second, err := user.PostFull(reportPath(convID), map[string]any{"reason": "harassment"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, second.StatusCode, second.Body)
	assert.Equal(t, convID, jsonField(parseJSON(second.Body), "id"))
}
