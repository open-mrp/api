//go:build plans

package repository

import (
	"context"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	dbpkg "github.com/open-mrp/api/shared/db"
)

// TestPickList_SearchMatchesEveryField pins a search, whether its few matches are paged in memory or a
// common one is tested on each pick walked, to the picks whose number or PO number starts with the term
// (anywhere in it under contains), or whose customer's name or number holds it, in list order and
// across pages.
func TestPickList_SearchMatchesEveryField(t *testing.T) {
	ensureFulfillmentCorpus(t)
	db := planDB(t)
	q := sqlc.New(db)
	contains := constants.PickSearchMatchContains
	cases := map[string]struct {
		term  string
		match constants.PickSearchMatch
	}{
		"number":                        {term: "12345"},
		"number range":                  {term: "123"},
		"number middle is not a prefix": {term: "4123"},
		"po number":                     {term: "PO-3000"},
		"every po":                      {term: "PO-3"},
		"customer name":                 {term: "Customer 0012"},
		"every customer":                {term: "plan customer"},
		"customer number":               {term: "C00012"},
		"short":                         {term: "12"},
		"none":                          {term: "no such pick"},
		"contains number":               {term: "4123", match: contains},
		"contains numbers":              {term: "234", match: contains},
		"contains po":                   {term: "3000", match: contains},
		"contains every customer":       {term: "Customer", match: contains},
	}
	const pages, limit = 4, 25
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			number, phrase := dbpkg.EscapeLike(tc.term)+"%", "%"+dbpkg.EscapeLike(tc.term)+"%"
			if tc.match == contains {
				number = phrase
			}
			where, args := "p.number LIKE ?", []any{planFulAccount, number}
			if utf8.RuneCountInString(tc.term) >= pickSubstringSearchMinRunes {
				where += ` OR EXISTS (SELECT 1 FROM sales_order so WHERE so.id = p.sales_order_id AND so.customer_po_number LIKE ?)
					OR EXISTS (SELECT 1 FROM account_relation r JOIN account b ON b.id = r.counterparty_account_id
						WHERE r.owner_account_id = p.account_id AND r.counterparty_account_id = p.buyer_account_id
						AND (b.name LIKE ? OR r.external_number LIKE ?))`
				args = append(args, number, phrase, phrase)
			}
			want, err := selectStrings(context.Background(), db, "SELECT p.id FROM pick p WHERE p.account_id = ? AND ("+where+
				") ORDER BY p.created_at DESC, p.id DESC LIMIT ?", append(args, pages*limit)...)
			require.NoError(t, err)
			require.Equal(t, name == "none" || name == "number middle is not a prefix", len(want) == 0, "a case must match what it names")

			params := domain.ListPicksParams{
				AccountID: planFulAccount, Limit: limit, Query: &tc.term, QueryMatch: tc.match, Sort: constants.PickSortCreatedAt,
			}
			var got []string
			for range pages {
				res, apiErr := NewPickRepo(q).List(context.Background(), params)
				require.Nil(t, apiErr)
				for _, pk := range res.Picks {
					got = append(got, pk.ID)
				}
				if res.PageInfo.NextCursor == nil {
					break
				}
				params.Cursor = res.PageInfo.NextCursor
			}
			if want == nil {
				want = []string{}
			}
			if got == nil {
				got = []string{}
			}
			require.Equal(t, want, got)
		})
	}
}
