package repository

import (
	"strings"
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/stretchr/testify/assert"
)

func TestSupplierListPageQuery_EmitsOnlySuppliedPredicates(t *testing.T) {
	query, args := supplierListPageQuery(domain.ListSuppliersParams{OwnerAccountID: "ac_1"}, nil, 11)

	assert.Equal(t, "SELECT ar.counterparty_account_id FROM account_relation ar FORCE INDEX (account_relation_owner_role_created_idx)\n"+
		"WHERE ar.owner_account_id = ?\n"+
		"AND ar.account_relation_role_code = 'supplier'\n"+
		"ORDER BY ar.created_at DESC, ar.counterparty_account_id DESC\n"+
		"LIMIT ?", query)
	assert.Equal(t, []any{"ac_1", int32(11)}, args)
	assert.NotContains(t, query, " OR ", "an unfiltered list carries no optional guards")
	assert.NotContains(t, query, "JOIN account a", "only a search reads the supplier's account")

	blank := "  ,  "
	query, _ = supplierListPageQuery(domain.ListSuppliersParams{OwnerAccountID: "ac_1", Query: &blank}, nil, 11)
	assert.NotContains(t, query, "LIKE", "a search with no words is no search")
}

func TestSupplierListPageQuery_EveryWordMustMatchOnItsOwn(t *testing.T) {
	q := "Acme,  sup-001 100%"
	query, args := supplierListPageQuery(domain.ListSuppliersParams{OwnerAccountID: "ac_1", Query: &q}, nil, 11)

	assert.Contains(t, query, "\nJOIN account a ON a.id = ar.counterparty_account_id\n")
	wordMatch := "(COALESCE(NULLIF(ar.alias, ''), a.name) LIKE ? OR ar.external_number LIKE ? OR ar.notes LIKE ?)"
	assert.Equal(t, 3, strings.Count(query, wordMatch), "one predicate per word, AND-ed together")
	assert.Equal(t, []any{
		"ac_1",
		"%Acme%", "%Acme%", "%Acme%",
		"%sup-001%", "%sup-001%", "%sup-001%",
		`%100%`, `%100%`, `%100%`,
		int32(11),
	}, args, "punctuation around a word is dropped, inside one it is kept")
	assert.Equal(t, strings.Count(query, "?"), len(args))
}

func TestSupplierListPageQuery_EscapesLikeWildcards(t *testing.T) {
	q := "50%_off"
	_, args := supplierListPageQuery(domain.ListSuppliersParams{OwnerAccountID: "ac_1", Query: &q}, nil, 11)
	assert.Equal(t, `%50\%\_off%`, args[1], "a wildcard in the search is matched literally")
}

func TestSupplierListPageQuery_FiltersAndBackwardCursor(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	at := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	cursor := &pagination.StringCursor{OccurredAt: at, ID: "ac_9", Direction: pagination.DirectionBackward}

	query, args := supplierListPageQuery(domain.ListSuppliersParams{
		OwnerAccountID: "ac_1",
		StartDate:      &start,
		EndDate:        &end,
		ItemIDs:        []string{"it_1", "it_2"},
	}, cursor, 6)

	assert.Contains(t, query, "AND ar.created_at >= ?\nAND ar.created_at <= ?\n")
	assert.Contains(t, query, "AND m.item_id IN (?,?))")
	assert.Contains(t, query, "sm.owner_account_id = ar.owner_account_id", "only the owner's links can match an item")
	assert.Contains(t, query, "AND (ar.created_at > ? OR (ar.created_at = ? AND ar.counterparty_account_id > ?))")
	assert.Contains(t, query, "ORDER BY ar.created_at ASC, ar.counterparty_account_id ASC")
	assert.Equal(t, []any{"ac_1", start, end, "it_1", "it_2", at, at, "ac_9", int32(6)}, args)
}

func TestSupplierSearchWords(t *testing.T) {
	q := func(s string) *string { return &s }

	assert.Nil(t, supplierSearchWords(nil))
	assert.Empty(t, supplierSearchWords(q("")))
	assert.Equal(t, []string{"Acme", "Inc"}, supplierSearchWords(q(" Acme, Inc. ")))
	assert.Equal(t, []string{"SUP-001"}, supplierSearchWords(q("(SUP-001)")))
	assert.Equal(t, []string{"yarn", "Co"}, supplierSearchWords(q("yarn YARN Co")), "a repeated word adds no predicate")
	assert.Equal(t, []string{"Müller"}, supplierSearchWords(q("Müller")))

	many := strings.Repeat("w ", 40) + strings.Join([]string{"a1", "a2", "a3", "a4", "a5", "a6", "a7", "a8", "a9", "a10", "a11", "a12", "a13", "a14", "a15", "a16", "a17"}, " ")
	assert.Len(t, supplierSearchWords(q(many)), supplierSearchWordLimit)
}
