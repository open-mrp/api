package repository

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/pagination"
)

func groupsOf(ids ...string) []domain.SalesBreakdownGroup {
	out := make([]domain.SalesBreakdownGroup, len(ids))
	for i, k := range ids {
		out[i] = domain.SalesBreakdownGroup{Key: k}
	}
	return out
}

func groupKeys(gs []domain.SalesBreakdownGroup) []string {
	out := make([]string, len(gs))
	for i, g := range gs {
		out[i] = g.Key
	}
	return out
}

func decode(t *testing.T, s *string) pagination.ValueCursor {
	t.Helper()
	require.NotNil(t, s)
	c, err := pagination.DecodeValueCursor(*s)
	require.NoError(t, err)
	return c
}

func TestBreakdownPageFirstPage(t *testing.T) {
	pagination.Init([]byte("breakdown-page-test"))
	got, info := breakdownPage(groupsOf("a", "b", "c"), []string{"30", "20", "10"}, 2, nil)

	require.Equal(t, []string{"a", "b"}, groupKeys(got))
	require.True(t, info.HasNextPage)
	require.False(t, info.HasPrevPage)
	require.Equal(t, pagination.ValueCursor{Value: "20", ID: "b", Direction: pagination.DirectionForward}, decode(t, info.NextCursor))
	require.Nil(t, info.PrevCursor)
}

func TestBreakdownPageLastForwardPage(t *testing.T) {
	pagination.Init([]byte("breakdown-page-test"))
	cur := &pagination.ValueCursor{Value: "20", ID: "b", Direction: pagination.DirectionForward}
	got, info := breakdownPage(groupsOf("c"), []string{"10"}, 2, cur)

	require.Equal(t, []string{"c"}, groupKeys(got))
	require.False(t, info.HasNextPage)
	require.True(t, info.HasPrevPage)
	require.Equal(t, pagination.ValueCursor{Value: "10", ID: "c", Direction: pagination.DirectionBackward}, decode(t, info.PrevCursor))
}

func TestBreakdownPageBackwardPageIsTurnedAround(t *testing.T) {
	pagination.Init([]byte("breakdown-page-test"))
	// Read in reverse ranking from before "c": b, then a, then one more beyond the limit.
	cur := &pagination.ValueCursor{Value: "10", ID: "c", Direction: pagination.DirectionBackward}
	got, info := breakdownPage(groupsOf("b", "a", "z"), []string{"20", "30", "40"}, 2, cur)

	require.Equal(t, []string{"a", "b"}, groupKeys(got))
	require.True(t, info.HasNextPage)
	require.True(t, info.HasPrevPage, "a row beyond the limit lies before this page")
	require.Equal(t, pagination.ValueCursor{Value: "20", ID: "b", Direction: pagination.DirectionForward}, decode(t, info.NextCursor))
	require.Equal(t, pagination.ValueCursor{Value: "30", ID: "a", Direction: pagination.DirectionBackward}, decode(t, info.PrevCursor))
}

func TestBreakdownPageEmpty(t *testing.T) {
	got, info := breakdownPage(nil, nil, 2, nil)
	require.Empty(t, got)
	require.Equal(t, pagination.PageInfo{}, info)
}
