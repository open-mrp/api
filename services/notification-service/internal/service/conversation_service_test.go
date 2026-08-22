package service

import (
	"testing"

	"github.com/open-mrp/api/shared/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDMKey_OrderIndependent(t *testing.T) {
	a := buildDMKey("acus_a", "acus_b")
	b := buildDMKey("acus_b", "acus_a")
	assert.Equal(t, a, b, "the DM key must be identical regardless of argument order")
	assert.Equal(t, "acus_a:acus_b", a, "the key is the sorted pair joined by a colon")
}

func TestUnreadFrom(t *testing.T) {
	// next_sequence is one past the last assigned sequence, so maxSeq = next-1.
	assert.Equal(t, int64(0), unreadFrom(1, 0), "an empty conversation has no unread")
	assert.Equal(t, int64(3), unreadFrom(4, 0), "3 messages, cursor at 0 → 3 unread")
	assert.Equal(t, int64(1), unreadFrom(4, 2), "cursor at 2 of 3 → 1 unread")
	assert.Equal(t, int64(0), unreadFrom(4, 3), "cursor caught up → 0 unread")
	assert.Equal(t, int64(0), unreadFrom(4, 9), "cursor ahead (shouldn't happen) → clamped to 0")
}

func TestSeqCursor_RoundTrip(t *testing.T) {
	c := encodeSeqCursor(42)
	got, apiErr := decodeSeqCursor(&c)
	require.Nil(t, apiErr)
	require.NotNil(t, got)
	assert.Equal(t, int64(42), *got)
}

func TestSeqCursor_NilAndMalformed(t *testing.T) {
	got, apiErr := decodeSeqCursor(nil)
	require.Nil(t, apiErr)
	assert.Nil(t, got)

	bad := "not-base64-seq!!"
	_, apiErr = decodeSeqCursor(&bad)
	require.NotNil(t, apiErr)
}

func TestRoleAllows(t *testing.T) {
	assert.True(t, roleAllows("owner", constants.ParticipantRoleOwner, constants.ParticipantRoleAdmin))
	assert.True(t, roleAllows("admin", constants.ParticipantRoleOwner, constants.ParticipantRoleAdmin))
	assert.False(t, roleAllows("member", constants.ParticipantRoleOwner, constants.ParticipantRoleAdmin))
	assert.False(t, roleAllows("viewer", constants.ParticipantRoleOwner, constants.ParticipantRoleAdmin))
	assert.True(t, roleAllows("viewer", constants.ParticipantRoleViewer))
}

func TestTruncatePreview(t *testing.T) {
	short := "hello"
	assert.Equal(t, short, truncatePreview(short))

	long := make([]byte, messagePreviewMaxLen+50)
	for i := range long {
		long[i] = 'x'
	}
	assert.Len(t, truncatePreview(string(long)), messagePreviewMaxLen)
}
