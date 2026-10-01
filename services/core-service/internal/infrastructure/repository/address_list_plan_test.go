//go:build plans

package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/pagination"
)

func addressPlanCases() []planCase[domain.ListAddressesParams] {
	type P = domain.ListAddressesParams
	str := func(s string) *string { return &s }
	yes, no := true, false
	mid := planCatOrigin.Add(planCatSpan / 2)
	cursor := func(dir pagination.Direction) func(*P) {
		return func(p *P) {
			s := pagination.EncodeStringCursor(pagination.StringCursor{OccurredAt: mid, ID: "ad_plancat_~", Direction: dir})
			p.Cursor = &s
		}
	}
	return planCases(P{AccountID: planCatCustomerID(0), Limit: 25}, []planDim[P]{
		{"search", []planValue[P]{
			{"search=one", func(p *P) { p.Query = str("Ship To 0123") }},
			{"search=every", func(p *P) { p.Query = str("Plan") }},
		}},
		{"dropship", []planValue[P]{
			{"dropship=yes", func(p *P) { p.DropShip = &yes }},
			{"dropship=no", func(p *P) { p.DropShip = &no }},
		}},
	}, []planValue[P]{
		{"first", func(*P) {}},
		{"deep-next", cursor(pagination.DirectionForward)},
		{"deep-prev", cursor(pagination.DirectionBackward)},
	})
}

// addressPlanFloor is how many addresses the account has. Addresses carry no account (account_address
// links them), so no key yields one account's addresses in list order, or pins a drop-ship or search
// filter within them: the bar is reading only the account's addresses.
func addressPlanFloor(t *testing.T, sqlDB *sql.DB, p domain.ListAddressesParams) float64 {
	t.Helper()
	return planCount(t, sqlDB, "SELECT COUNT(*) FROM account_address WHERE account_id = ?", p.AccountID)
}

// TestAddressList_ReadsAboutAPage holds every filter combination ListAddresses accepts to reading only
// the account's addresses it may return (listPlanSuite, joinScoped).
func TestAddressList_ReadsAboutAPage(t *testing.T) {
	ensureCatalogCorpus(t)
	listPlanSuite[domain.ListAddressesParams]{
		table: "address", joinScoped: true,
		from: "FROM account_address aa", alias: "a",
		cases: addressPlanCases(),
		limit: func(p domain.ListAddressesParams) int32 { return p.Limit },
		list: func(ctx context.Context, q *sqlc.Queries, p domain.ListAddressesParams) error {
			if _, apiErr := NewAddressRepo(q).List(ctx, p); apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor:       addressPlanFloor,
		statsTables: []string{"account_address"},
		reads:       []planRead[domain.ListAddressesParams]{{alias: "aa", fanout: 1}, {alias: "g", fanout: 1}},
	}.run(t)
}
