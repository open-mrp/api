package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/apikit/pagination"
)

func TestEmbeddedNextCursor(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, time.October, 1, 12, 0, 0, 123456000, time.UTC)
	children := func(n int) []domain.LocationChild {
		out := make([]domain.LocationChild, n)
		for i := range out {
			out[i] = domain.LocationChild{ID: "loc_" + string(rune('a'+i)), CreatedAt: base.Add(-time.Duration(i) * time.Minute)}
		}
		return out
	}
	createdAt := func(c domain.LocationChild) time.Time { return c.CreatedAt }
	id := func(c domain.LocationChild) string { return c.ID }
	limit := domain.EmbeddedLocationChildren

	assert.Nil(t, embeddedNextCursor(children(limit-1), limit, createdAt, id), "a load that didn't fill up has nothing after it")
	assert.Nil(t, embeddedNextCursor(children(limit), 0, createdAt, id), "an unlimited load has nothing after it")

	items := children(limit)
	got := embeddedNextCursor(items, limit, createdAt, id)
	require.NotNil(t, got)
	c, err := pagination.DecodeStringCursor(*got)
	require.NoError(t, err)
	last := items[limit-2]
	assert.Equal(t, last.ID, c.ID, "the cursor follows the last embedded item, not the extra one")
	assert.True(t, last.CreatedAt.Equal(c.OccurredAt))
	assert.Equal(t, pagination.DirectionForward, c.Direction)
	assert.Nil(t, c.MatchTier, "the list continues on its unranked newest-first order")
}
