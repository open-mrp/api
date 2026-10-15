package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/apikit/pagination"
)

func TestBuildLocationListQuery_EmitsOnlySuppliedFilters(t *testing.T) {
	t.Parallel()

	query, args := buildLocationListQuery(locationListFilter{accountID: "ac_1", limit: 26})
	assert.Equal(t, "SELECT "+locationListColumns+"\nFROM storage_location sl\nLEFT JOIN storage_location p ON p.id = sl.parent_id"+
		"\nWHERE sl.account_id = ?\nORDER BY sl.created_at DESC, sl.id DESC\nLIMIT ?", query)
	assert.Equal(t, []any{"ac_1", int32(26)}, args)

	active := true
	at := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	query, args = buildLocationListQuery(locationListFilter{
		accountID: "ac_1", search: "%bin%", isActive: &active, parentIDs: []string{"sglc_1", "sglc_2"},
		cursor: &pagination.StringCursor{Direction: pagination.DirectionBackward, OccurredAt: at, ID: "sglc_9"}, limit: 11,
	})
	assert.Contains(t, query, "WHERE sl.account_id = ?\nAND sl.name LIKE ?\nAND sl.is_active = ?\nAND sl.parent_id IN (?,?)"+
		"\nAND (sl.created_at > ? OR (sl.created_at = ? AND sl.id > ?))")
	assert.Contains(t, query, "ORDER BY sl.created_at ASC, sl.id ASC")
	assert.Equal(t, []any{"ac_1", "%bin%", true, "sglc_1", "sglc_2", at, at, "sglc_9", int32(11)}, args)
}
