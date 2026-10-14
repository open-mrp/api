//go:build plans

package repository

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	dbpkg "github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/pagination"
)

func customerPlanDims() []planDim[domain.ListCustomersParams] {
	type P = domain.ListCustomersParams
	str := func(s string) *string { return &s }
	yes, no := true, false
	recent := planCatOrigin.Add(planCatSpan)
	old := planCatOrigin.Add(planCatSpan / 4)
	ids := func(set func(*P, []string), vals ...string) func(*P) { return func(p *P) { set(p, vals) } }
	return []planDim[P]{
		{"search", []planValue[P]{
			{"search=one", func(p *P) { p.Query = str("Customer 1234") }},
			{"search=every", func(p *P) { p.Query = str("Plan Customer") }},
		}},
		{"group", []planValue[P]{
			{"group=dense", ids(func(p *P, v []string) { p.CustomerGroupIDs = v }, planCatGroupID(0))},
			{"group=rare", ids(func(p *P, v []string) { p.CustomerGroupIDs = v }, planCatGroupID(planCatGroups-1))},
		}},
		{"pricing", []planValue[P]{
			{"pricing=dense", ids(func(p *P, v []string) { p.PricingGroupIDs = v }, planCatGroupID(12))},
			{"pricing=none", ids(func(p *P, v []string) { p.PricingGroupIDs = v }, planCatGroupID(0))},
		}},
		{"rep", []planValue[P]{
			{"rep=dense", ids(func(p *P, v []string) { p.SalesRepIDs = v }, planCatRepID(0))},
			{"rep=rare", ids(func(p *P, v []string) { p.SalesRepIDs = v }, planCatRepID(planCatReps-1))},
		}},
		{"status", []planValue[P]{
			{"status=normal", ids(func(p *P, v []string) { p.StatusCodes = v }, "normal")},
			{"status=hold_shipment", ids(func(p *P, v []string) { p.StatusCodes = v }, "hold_shipment")},
		}},
		{"shipping", []planValue[P]{
			{"shipping=dense", ids(func(p *P, v []string) { p.ShippingTermIDs = v }, "st_plancat_00")},
			{"shipping=rare", ids(func(p *P, v []string) { p.ShippingTermIDs = v }, "st_plancat_14")},
		}},
		{"payment", []planValue[P]{
			{"payment=dense", ids(func(p *P, v []string) { p.PaymentTermIDs = v }, "pt_plancat_00")},
			{"payment=rare", ids(func(p *P, v []string) { p.PaymentTermIDs = v }, "pt_plancat_14")},
		}},
		{"commission", []planValue[P]{
			{"commission=dense", ids(func(p *P, v []string) { p.CommissionPolicyCodes = v }, "commission_applied")},
			{"commission=rare", ids(func(p *P, v []string) { p.CommissionPolicyCodes = v }, "commission_exempt")},
		}},
		{"freight", []planValue[P]{
			{"freight=dense", ids(func(p *P, v []string) { p.FreightPolicyCodes = v }, "billed_freight")},
			{"freight=rare", ids(func(p *P, v []string) { p.FreightPolicyCodes = v }, "free_freight")},
		}},
		{"carrier", []planValue[P]{
			{"carrier=dense", ids(func(p *P, v []string) { p.CarrierIDs = v }, "carr_plancat_00")},
			{"carrier=none", ids(func(p *P, v []string) { p.CarrierIDs = v }, "carr_plancat_99")},
		}},
		{"level", []planValue[P]{
			{"level=dense", ids(func(p *P, v []string) { p.ServiceLevelIDs = v }, "caop_plancat_00")},
			{"level=none", ids(func(p *P, v []string) { p.ServiceLevelIDs = v }, "caop_plancat_99")},
		}},
		{"parent", []planValue[P]{
			{"parent=yes", func(p *P) { p.IsParentAccount = &yes }},
			{"parent=no", func(p *P) { p.IsParentAccount = &no }},
		}},
		{"place", []planValue[P]{
			{"place=state", func(p *P) { p.State = str("NC") }},
			{"place=city+postal", func(p *P) { p.City, p.PostalCode = str("Plan City 07"), str("20007") }},
		}},
		{"created", []planValue[P]{
			{"created=last30d", func(p *P) { at := recent.AddDate(0, 0, -30); p.StartDate = &at }},
			{"created=old30d", func(p *P) { s, e := old, old.AddDate(0, 0, 30); p.StartDate, p.EndDate = &s, &e }},
		}},
	}
}

func customerPlanCases() []planCase[domain.ListCustomersParams] {
	type P = domain.ListCustomersParams
	cursor := func(dir pagination.Direction) func(*P) {
		return func(p *P) {
			mid := planCatOrigin.Add(planCatSpan / 2)
			s := pagination.EncodeStringCursor(pagination.StringCursor{OccurredAt: mid, ID: "ac_plancat_~", Direction: dir})
			p.Cursor = &s
		}
	}
	return planCases(P{AccountID: planCatAccount, Limit: 25}, customerPlanDims(), []planValue[P]{
		{"first", func(*P) {}},
		{"deep-next", cursor(pagination.DirectionForward)},
		{"deep-prev", cursor(pagination.DirectionBackward)},
	})
}

// customerPlanFloor is how many of the account's customer relations a request must read when a
// filter cannot be served in list order: a substring search examines every relation the indexed
// filters leave, and a price group (a child table) is held to the relations it matches.
func customerPlanFloor(t *testing.T, sqlDB *sql.DB, p domain.ListCustomersParams) float64 {
	t.Helper()
	searching := p.Query != nil && *p.Query != ""
	if !searching && len(p.PricingGroupIDs) == 0 {
		return 0
	}
	where := []string{"ar.owner_account_id = ?", "ar.account_relation_role_code = 'customer'"}
	args := []any{p.AccountID}
	in := func(col string, vals []string) {
		if len(vals) > 0 {
			where = append(where, col+" IN ("+placeholders(len(vals))+")")
			args = append(args, stringArgs(vals)...)
		}
	}
	in("ar.account_group_id", p.CustomerGroupIDs)
	in("ar.default_sales_rep_id", p.SalesRepIDs)
	in("ar.account_status_code", p.StatusCodes)
	if p.StartDate != nil {
		where, args = append(where, "ar.created_at >= ?"), append(args, *p.StartDate)
	}
	if p.EndDate != nil {
		where, args = append(where, "ar.created_at <= ?"), append(args, *p.EndDate)
	}
	if !searching {
		where = append(where, "EXISTS (SELECT 1 FROM account_relation_price_group arpg WHERE arpg.account_relation_id = ar.id AND arpg.account_group_id IN ("+placeholders(len(p.PricingGroupIDs))+"))")
		args = append(args, stringArgs(p.PricingGroupIDs)...)
	}
	return planCount(t, sqlDB, "SELECT COUNT(*) FROM account_relation ar WHERE "+strings.Join(where, " AND "), args...)
}

// TestCustomerList_ReadsAboutAPage holds every filter combination ListCustomers accepts to reading
// about a page of customer relations (listPlanSuite).
func TestCustomerList_ReadsAboutAPage(t *testing.T) {
	ensureCatalogCorpus(t)
	listPlanSuite[domain.ListCustomersParams]{
		table: "account_relation", scopeColumn: "owner_account_id",
		from: "FROM account_relation ar", alias: "ar",
		cases: customerPlanCases(),
		limit: func(p domain.ListCustomersParams) int32 { return p.Limit },
		list: func(ctx context.Context, q *sqlc.Queries, p domain.ListCustomersParams) error {
			if _, apiErr := NewCustomerRepo(q).List(ctx, p); apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor:       customerPlanFloor,
		statsTables: []string{"account_address", "address"},
		reads: []planRead[domain.ListCustomersParams]{
			{alias: "a", fanout: 1}, {alias: "sab", fanout: 1}, {alias: "car", fanout: 1},
			// A place filter probes each customer's own addresses: five apiece here, about as many in production.
			{alias: "aa", fanout: 5}, {alias: "addr", fanout: 5}, {alias: "g", fanout: 5},
		},
	}.run(t)
}

// TestCustomerList_SearchMatchesEveryField pins a search, read from its matches or tested in the walk,
// to the customers whose name, alias, number, notes, or support email contains the term, in list order
// and across pages.
func TestCustomerList_SearchMatchesEveryField(t *testing.T) {
	ensureCatalogCorpus(t)
	db := planDB(t)
	ctx := context.Background()
	_, err := db.ExecContext(ctx, `INSERT INTO account_branding (id, owner_account_id, support_email) VALUES ('abr_plancat_search', ?, 'only-here@plan.test')`,
		planCatCustomerID(17))
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = db.ExecContext(ctx, `DELETE FROM account_branding WHERE id = 'abr_plancat_search'`) })

	str := func(s string) *string { return &s }
	cases := map[string]domain.ListCustomersParams{
		"name":           {Query: str("Customer 1234 Inc")},
		"alias":          {Query: str("Plan Customer 0042")},
		"number":         {Query: str("C0012")},
		"notes":          {Query: str("notes 03")},
		"support email":  {Query: str("only-here@")},
		"dense":          {Query: str("Incorporated")},
		"with filter":    {Query: str("C001"), StatusCodes: []string{"normal"}},
		"none wildcards": {Query: str("C0_1%")},
		"none":           {Query: str("no such customer")},
	}
	const limit = 25
	for name, params := range cases {
		t.Run(name, func(t *testing.T) {
			params.AccountID, params.Limit = planCatAccount, limit
			like := "%" + dbpkg.EscapeLike(*params.Query) + "%"
			where := `ar.owner_account_id = ? AND ar.account_relation_role_code = 'customer' AND (a.name LIKE ? OR ar.alias LIKE ?
				OR ar.external_number LIKE ? OR ar.notes LIKE ? OR ab.support_email LIKE ?)`
			args := []any{planCatAccount, like, like, like, like, like}
			if len(params.StatusCodes) > 0 {
				where += " AND ar.account_status_code IN (" + placeholders(len(params.StatusCodes)) + ")"
				args = append(args, stringArgs(params.StatusCodes)...)
			}
			want, err := selectStrings(ctx, db, `SELECT ar.counterparty_account_id FROM account_relation ar
JOIN account a ON a.id = ar.counterparty_account_id
LEFT JOIN account_branding ab ON ab.owner_account_id = ar.counterparty_account_id
WHERE `+where+" ORDER BY ar.created_at DESC, ar.counterparty_account_id DESC LIMIT ?", append(args, 4*limit)...)
			require.NoError(t, err)
			require.Equal(t, strings.HasPrefix(name, "none"), len(want) == 0, "a case must match what it names")

			repo := NewCustomerRepo(sqlc.New(db))
			var got []string
			for range 4 {
				res, apiErr := repo.List(ctx, params)
				require.Nil(t, apiErr)
				for _, c := range res.Items {
					got = append(got, c.ID)
				}
				if res.PageInfo.NextCursor == nil {
					break
				}
				params.Cursor = res.PageInfo.NextCursor
			}
			require.Equal(t, want, got)
		})
	}
}
