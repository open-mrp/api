//go:build plans

package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	dbpkg "github.com/open-mrp/api/shared/db"
)

// TestShipmentList_SearchMatchesEveryField pins a search, whether its few matches are read up front or
// a common one is tested on each shipment walked, to the shipments whose number, note, bill of lading
// or tracking number, order number or PO number, or customer's name holds the term, in list order and
// across pages, alone and beside a filter.
func TestShipmentList_SearchMatchesEveryField(t *testing.T) {
	ensureFulfillmentCorpus(t)
	db := planDB(t)
	q := sqlc.New(db)
	shipped := "shipped"
	cases := map[string]struct {
		term   string
		status *string
	}{
		"number":           {term: planFulNumber(12345)},
		"number substring": {term: planFulNumber(12345)[2:]},
		"note":             {term: "dock 4"},
		"bill of lading":   {term: "BOL0000012"},
		"tracking":         {term: "1Z000000000001"},
		"order number":     {term: "SO" + planFulNumber(2000)},
		"po number":        {term: "PO-3000"},
		"customer name":    {term: "Customer 0012"},
		"every customer":   {term: "plan customer"},
		"short":            {term: "12"},
		"unindexed":        {term: "a"},
		"none":             {term: "no such shipment"},
		"with a status":    {term: "PO-3000", status: &shipped},
	}
	const pages, limit = 4, 25
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			like := "%" + dbpkg.EscapeLike(tc.term) + "%"
			where, args := "", []any{planFulAccount}
			if tc.status != nil {
				where, args = " AND s.shipment_status_code = ?", append(args, *tc.status)
			}
			args = append(args, like, like, like, like, like, like, like, pages*limit)
			want, err := selectStrings(context.Background(), db, `SELECT s.id FROM shipment s
				JOIN sales_order so ON so.id = s.sales_order_id JOIN account ba ON ba.id = so.buyer_account_id
				WHERE s.account_id = ?`+where+shipmentListable+`
				  AND (s.number LIKE ? OR s.note LIKE ? OR s.bill_of_lading LIKE ? OR s.master_tracking_number LIKE ?
				    OR so.number LIKE ? OR ba.name LIKE ? OR so.customer_po_number LIKE ?)
				ORDER BY s.created_at DESC, s.id DESC LIMIT ?`, args...)
			require.NoError(t, err)
			require.Equal(t, name == "none", len(want) == 0, "a case must match what it names")

			params := domain.ListShipmentsParams{AccountID: planFulAccount, Limit: limit, Query: &tc.term, Status: tc.status}
			var got []string
			for range pages {
				res, apiErr := NewShipmentRepo(q).List(context.Background(), params)
				require.Nil(t, apiErr)
				for _, s := range res.Shipments {
					got = append(got, s.ID)
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

// TestShipmentList_SearchReadsItsMatches holds a search that matches few shipments to reading them
// through the FULLTEXT keys, never every shipment of the account.
func TestShipmentList_SearchReadsItsMatches(t *testing.T) {
	ensureFulfillmentCorpus(t)
	db := planDB(t)
	for _, term := range []string{planFulNumber(12345), "BOL0000012", "SO" + planFulNumber(2000), "PO-3000013", "Customer 0012"} {
		t.Run(term, func(t *testing.T) {
			edb := &explainingDB{db: db}
			_, apiErr := NewShipmentRepo(sqlc.New(edb)).List(context.Background(), domain.ListShipmentsParams{
				AccountID: planFulAccount, Limit: 25, Query: &term,
			})
			require.Nil(t, apiErr)
			var search, page *explainedStatement
			for i := range edb.statements {
				switch st := &edb.statements[i]; {
				case strings.HasPrefix(st.query, "(SELECT s.id FROM shipment s WHERE"):
					search = st
				case strings.HasPrefix(st.query, "SELECT STRAIGHT_JOIN s.id "):
					page = st
				}
			}
			require.NotNil(t, search, "the matches were read up front")
			require.NotNil(t, page, "the page was read")
			require.NotRegexp(t, `on s using (shipment_account_created_idx|shipment_account_id_shipment_status)`, search.plan, "an arm walks the account\n%s", search.plan)
			require.Regexp(t, `FORCE INDEX \(PRIMARY\)`, page.query)
			require.NotContains(t, page.query, "LIKE", "the page tests the search again")
		})
	}
}
