package repository

import (
	"strings"
	"testing"
	"time"

	"github.com/open-mrp/api/shared/pagination"
	"github.com/stretchr/testify/assert"
)

func TestTerritoryListPageQuery_EmitsOnlySuppliedPredicates(t *testing.T) {
	query, args := territoryListPageQuery("ac_1", nil, nil, 11)

	assert.Equal(t, "SELECT t.id FROM territory t\n"+
		"JOIN account_user au ON au.id = t.sales_rep_id\n"+
		"JOIN `user` u ON u.id = au.user_id\n"+
		"WHERE t.account_id = ?\n"+
		"ORDER BY t.created_at DESC, t.id DESC\n"+
		"LIMIT ?", query)
	assert.Equal(t, []any{"ac_1", int32(11)}, args)
	assert.NotContains(t, query, " OR ", "an unfiltered list carries no optional guards")

	empty := ""
	query, _ = territoryListPageQuery("ac_1", &empty, nil, 11)
	assert.NotContains(t, query, "LIKE", "an empty search is no search")
}

func TestTerritoryListPageQuery_TextSearchScopesTheProductLineToTheAccount(t *testing.T) {
	q := "100%_east"
	query, args := territoryListPageQuery("ac_1", &q, nil, 11)

	assert.Contains(t, query, "LEFT JOIN product_line pl ON pl.id = t.product_line_id AND (pl.account_id = t.account_id OR pl.account_id IS NULL)",
		"another tenant's product line name must never match")
	assert.Contains(t, query, "AND (t.state LIKE ? OR u.name LIKE ? OR u.email LIKE ? OR pl.name LIKE ?)\n")
	assert.NotContains(t, query, "zipcode", "a term that is not a ZIP code adds no ZIP code match")
	like := `%100\%\_east%`
	assert.Equal(t, []any{"ac_1", like, like, like, like, int32(11)}, args)
}

func TestTerritoryListPageQuery_ZipcodeSearchAlsoMatchesCoveringTerritories(t *testing.T) {
	q := "36500"
	query, args := territoryListPageQuery("ac_1", &q, nil, 11)

	assert.Contains(t, query, "AND (t.state LIKE ? OR u.name LIKE ? OR u.email LIKE ? OR pl.name LIKE ?"+
		" OR (t.start_zipcode <= ? AND t.end_zipcode >= ?) OR (t.start_zipcode = ? AND t.end_zipcode IS NULL))",
		"the ZIP code match is OR-ed with the text match, not AND-ed")
	assert.Equal(t, []any{"ac_1", "%36500%", "%36500%", "%36500%", "%36500%", int32(36500), int32(36500), int32(36500), int32(11)}, args)
	assert.Equal(t, strings.Count(query, "?"), len(args))
}

func TestTerritoryZipcodeQuery(t *testing.T) {
	for q, want := range map[string]int32{"36500": 36500, "00501": 501, "99999": 99999} {
		got, ok := territoryZipcodeQuery(q)
		assert.True(t, ok, q)
		assert.Equal(t, want, got, q)
	}
	for _, q := range []string{"500", "100000", "0", "000", "NY", "365a0", "-501", "4294967797"} {
		_, ok := territoryZipcodeQuery(q)
		assert.False(t, ok, q)
	}
}

func TestTerritoryListPageQuery_Cursors(t *testing.T) {
	at := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)

	query, args := territoryListPageQuery("ac_1", nil, &pagination.StringCursor{OccurredAt: at, ID: "tr_9", Direction: pagination.DirectionForward}, 11)
	assert.Contains(t, query, "AND (t.created_at < ? OR (t.created_at = ? AND t.id < ?))\nORDER BY t.created_at DESC, t.id DESC")
	assert.Equal(t, []any{"ac_1", at, at, "tr_9", int32(11)}, args)

	query, _ = territoryListPageQuery("ac_1", nil, &pagination.StringCursor{OccurredAt: at, ID: "tr_9", Direction: pagination.DirectionBackward}, 11)
	assert.Contains(t, query, "AND (t.created_at > ? OR (t.created_at = ? AND t.id > ?))\nORDER BY t.created_at ASC, t.id ASC")
}
