//go:build plans

package repository

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/pagination"
)

func shipmentPlanDims() []planDim[domain.ListShipmentsParams] {
	str := func(s string) *string { return &s }
	day := func(t time.Time) *string { return str(t.Format(time.DateOnly)) }
	recent := planFulCreatedAt(planFulOrders - 1)
	old := planFulCreatedAt(planFulOrders / 4)
	var tail []string
	for c := 100; c < 160; c++ {
		tail = append(tail, planFulCustomerID(c))
	}
	return []planDim[domain.ListShipmentsParams]{
		{"status", []planValue[domain.ListShipmentsParams]{
			{"shipped", func(p *domain.ListShipmentsParams) { p.Status = str("shipped") }},
			{"packed", func(p *domain.ListShipmentsParams) { p.Status = str("packed") }},
		}},
		{"customer", []planValue[domain.ListShipmentsParams]{
			{"large", func(p *domain.ListShipmentsParams) { p.CustomerIDs = []string{planFulCustomerID(0)} }},
			{"rare", func(p *domain.ListShipmentsParams) { p.CustomerIDs = []string{planFulCustomerID(planFulRareCustomer)} }},
			{"three", func(p *domain.ListShipmentsParams) {
				p.CustomerIDs = []string{planFulCustomerID(1), planFulCustomerID(300), planFulCustomerID(planFulRareCustomer)}
			}},
			{"sixty", func(p *domain.ListShipmentsParams) { p.CustomerIDs = tail }},
		}},
		{"group", []planValue[domain.ListShipmentsParams]{
			{"big", func(p *domain.ListShipmentsParams) { p.CustomerGroupIDs = []string{"ag_planful_big"} }},
			{"once", func(p *domain.ListShipmentsParams) { p.CustomerGroupIDs = []string{"ag_planful_once"} }},
			{"small", func(p *domain.ListShipmentsParams) { p.CustomerGroupIDs = []string{"ag_planful_small"} }},
		}},
		{"sales_rep", []planValue[domain.ListShipmentsParams]{
			{"dense", func(p *domain.ListShipmentsParams) { p.SalesRepIDs = []string{"acus_planful_dense"} }},
			{"once", func(p *domain.ListShipmentsParams) { p.SalesRepIDs = []string{"acus_planful_once"} }},
			{"rare", func(p *domain.ListShipmentsParams) { p.SalesRepIDs = []string{"acus_planful_rare"} }},
		}},
		{"item", []planValue[domain.ListShipmentsParams]{
			{"dense", func(p *domain.ListShipmentsParams) { p.ItemIDs = []string{planFulItemID(0)} }},
			{"rare", func(p *domain.ListShipmentsParams) { p.ItemIDs = []string{planFulItemID(planFulRareProduct)} }},
			{"zero", func(p *domain.ListShipmentsParams) { p.ItemIDs = []string{planFulItemID(planFulZeroProduct)} }},
		}},
		{"product_line", []planValue[domain.ListShipmentsParams]{
			{"dense", func(p *domain.ListShipmentsParams) { p.ProductLineIDs = []string{planFulLineDense} }},
			{"rare", func(p *domain.ListShipmentsParams) { p.ProductLineIDs = []string{planFulLineRare} }},
			{"zero", func(p *domain.ListShipmentsParams) { p.ProductLineIDs = []string{planFulLineZero} }},
		}},
		{"created", []planValue[domain.ListShipmentsParams]{
			{"last30d", func(p *domain.ListShipmentsParams) { p.StartDate = day(recent.Add(-30 * 24 * time.Hour)) }},
			{"old30d", func(p *domain.ListShipmentsParams) { p.StartDate, p.EndDate = day(old), day(old.Add(30*24*time.Hour)) }},
		}},
	}
}

// shipmentPlanCases is every filter pair on the first page and on a page deep in the account in both
// directions.
func shipmentPlanCases() []planCase[domain.ListShipmentsParams] {
	mid := planFulCreatedAt(planFulOrders / 2)
	cursor := func(dir pagination.Direction) func(*domain.ListShipmentsParams) {
		return func(p *domain.ListShipmentsParams) {
			c := pagination.EncodeStringCursor(pagination.StringCursor{OccurredAt: mid, ID: "sh_planful_~", Direction: dir})
			p.Cursor = &c
		}
	}
	return planCases(
		domain.ListShipmentsParams{AccountID: planFulAccount, Limit: 25},
		qualifiedPlanDims(shipmentPlanDims()),
		[]planValue[domain.ListShipmentsParams]{
			{"first", func(*domain.ListShipmentsParams) {}},
			{"deep-next", cursor(pagination.DirectionForward)},
			{"deep-prev", cursor(pagination.DirectionBackward)},
		},
	)
}

// shipmentBuyerSet is the customers a request may list, or nil for any: what
// shipmentRepoImpl.buyerFilter resolves, read here independently of it.
func shipmentBuyerSet(t *testing.T, db *sql.DB, p domain.ListShipmentsParams) []string {
	t.Helper()
	if len(p.CustomerGroupIDs) == 0 && len(p.SalesRepIDs) == 0 {
		return p.CustomerIDs
	}
	where, args := []string{"owner_account_id = ?"}, []any{p.AccountID}
	for column, values := range map[string][]string{
		"account_group_id": p.CustomerGroupIDs, "default_sales_rep_id": p.SalesRepIDs, "counterparty_account_id": p.CustomerIDs,
	} {
		if len(values) > 0 {
			where, args = append(where, column+" IN ("+placeholders(len(values))+")"), append(args, stringArgs(values)...)
		}
	}
	rows, err := db.Query("SELECT DISTINCT counterparty_account_id FROM account_relation WHERE "+strings.Join(where, " AND "), args...)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	set := []string{}
	for rows.Next() {
		var id string
		require.NoError(t, rows.Scan(&id))
		set = append(set, id)
	}
	require.NoError(t, rows.Err())
	return set
}

// shipmentUnorderedFloor is how many shipments the request's narrowest unordered filter matches, or 0
// when it has none. No key yields these in list order: a set of customers (ranges of the buyer key)
// and items or product lines (child tables). The best any plan can do is read one filter's matches.
func shipmentUnorderedFloor(t *testing.T, db *sql.DB, p domain.ListShipmentsParams) float64 {
	t.Helper()
	var floors []float64
	count := func(where string, args ...any) {
		var n float64
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM shipment s WHERE s.account_id = ? AND "+where,
			append([]any{p.AccountID}, args...)...).Scan(&n))
		floors = append(floors, n)
	}
	if buyers := shipmentBuyerSet(t, db, p); len(buyers) > 1 {
		count("s.buyer_account_id IN ("+placeholders(len(buyers))+")", stringArgs(buyers)...)
	}
	if len(p.ItemIDs) > 0 {
		count("s.id IN ("+shipmentsWithItems(len(p.ItemIDs))+")", stringArgs(p.ItemIDs)...)
	}
	if len(p.ProductLineIDs) > 0 {
		count("s.id IN ("+shipmentsWithProductLines(len(p.ProductLineIDs))+")", stringArgs(p.ProductLineIDs)...)
	}
	if len(floors) == 0 {
		return 0
	}
	floor := floors[0]
	for _, f := range floors[1:] {
		floor = min(floor, f)
	}
	return max(floor, 1)
}

// TestShipmentList_ReadsAboutAPage holds every filter combination ListShipments accepts to reading
// about a page of shipments (listPlanSuite).
func TestShipmentList_ReadsAboutAPage(t *testing.T) {
	ensureFulfillmentCorpus(t)
	db := planDB(t)
	listPlanSuite[domain.ListShipmentsParams]{
		table: "shipment", scopeColumn: "account_id",
		from: "FROM shipment s", alias: "s",
		statement: func(query string) bool { return strings.HasPrefix(query, "SELECT STRAIGHT_JOIN s.id") },
		cases:     shipmentPlanCases(),
		limit:     func(p domain.ListShipmentsParams) int32 { return p.Limit },
		list: func(ctx context.Context, q *sqlc.Queries, p domain.ListShipmentsParams) error {
			if _, apiErr := NewShipmentRepo(q).List(ctx, p); apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: shipmentUnorderedFloor,
		empty: func(p domain.ListShipmentsParams) bool {
			buyers := shipmentBuyerSet(t, db, p)
			return buyers != nil && len(buyers) == 0
		},
	}.run(t)
}
