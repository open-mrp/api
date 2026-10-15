package repository

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/apikit/pagination"
)

func TestBuildAddressListQuery_EmitsOnlyGivenFilters(t *testing.T) {
	t.Parallel()

	q, args := buildAddressListQuery(domain.ListAddressesParams{AccountID: "ac_1"}, pagination.DirectionForward, nil, 26)
	assert.NotContains(t, q, "IS NULL")
	assert.NotContains(t, q, "LIKE")
	assert.NotContains(t, q, "is_drop_ship =")
	assert.Contains(t, q, "ORDER BY a.created_at DESC, a.id DESC LIMIT ?")
	assert.Equal(t, []any{"ac_1", int32(26)}, args)

	search, dropShip := "dock", true
	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	q, args = buildAddressListQuery(domain.ListAddressesParams{AccountID: "ac_1", Query: &search, DropShip: &dropShip},
		pagination.DirectionBackward, &pagination.StringCursor{OccurredAt: at, ID: "ad_9"}, 11)
	assert.Equal(t, len(addressSearchColumns), strings.Count(q, " LIKE ?"))
	assert.Contains(t, q, "a.is_drop_ship = ?")
	assert.Contains(t, q, "(a.created_at > ? OR (a.created_at = ? AND a.id > ?))")
	assert.Contains(t, q, "ORDER BY a.created_at ASC, a.id ASC")
	assert.Len(t, args, 1+len(addressSearchColumns)+1+3+1)
	assert.Equal(t, "%dock%", args[1])
}
