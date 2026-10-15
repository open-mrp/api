package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/apikit/pagination"
)

func TestBuildProductLineListQuery_EmitsOnlySuppliedFilters(t *testing.T) {
	t.Parallel()

	query, args := buildProductLineListQuery(productLineListFilter{accountID: "ac_1", dir: pagination.DirectionForward, limit: 26})
	assert.Equal(t, "SELECT "+productLineListColumns+" FROM product_line pl WHERE (pl.account_id = ? OR pl.account_id IS NULL)"+
		" ORDER BY pl.created_at DESC, pl.id DESC LIMIT ?", query)
	assert.Equal(t, []any{"ac_1", int32(26)}, args)

	active := false
	at := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	query, args = buildProductLineListQuery(productLineListFilter{
		accountID: "ac_1", search: "%sock%", isActive: &active,
		dir: pagination.DirectionBackward, cursorCreatedAt: &at, cursorID: "pdln_1", limit: 11,
	})
	assert.Contains(t, query, "AND pl.name LIKE ? AND pl.is_active = ? AND (pl.created_at > ? OR (pl.created_at = ? AND pl.id > ?))")
	assert.Contains(t, query, "ORDER BY pl.created_at ASC, pl.id ASC")
	assert.Equal(t, []any{"ac_1", "%sock%", false, at, at, "pdln_1", int32(11)}, args)
}
