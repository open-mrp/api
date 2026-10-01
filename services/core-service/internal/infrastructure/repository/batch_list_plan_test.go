//go:build plans

package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/pagination"
)

func batchesByStationPlanCases() []planCase[domain.ListBatchesByScanningStationParams] {
	str := func(s string) *string { return &s }
	mid := planBatchCreatedAt(planBatchRows / 2)
	cursor := func(dir pagination.Direction) func(*domain.ListBatchesByScanningStationParams) {
		return func(p *domain.ListBatchesByScanningStationParams) { p.Cursor = planCursorAt(mid, "btch_~", dir) }
	}
	return planCases(
		domain.ListBatchesByScanningStationParams{AccountID: planBatchAccount, ScanningStationID: planBatchStationID(0), Limit: 25},
		[]planDim[domain.ListBatchesByScanningStationParams]{
			{"station", []planValue[domain.ListBatchesByScanningStationParams]{
				{"quiet", func(p *domain.ListBatchesByScanningStationParams) {
					p.ScanningStationID = planBatchStationID(len(planBatchStationShares) - 1)
				}},
			}},
			{"search", []planValue[domain.ListBatchesByScanningStationParams]{
				{"one", func(p *domain.ListBatchesByScanningStationParams) { p.Query = str(planBatchRareSKU) }},
				{"every", func(p *domain.ListBatchesByScanningStationParams) { p.Query = str(planBatchDenseSKU) }},
			}},
		},
		[]planValue[domain.ListBatchesByScanningStationParams]{
			{"first", func(*domain.ListBatchesByScanningStationParams) {}},
			{"deep-next", cursor(pagination.DirectionForward)},
			{"deep-prev", cursor(pagination.DirectionBackward)},
		},
	)
}

// batchesByStationSearchFloor is how many of the station's scanned batches a SKU search matches, or 0
// without one: a search matches an arbitrary set of items, which no key yields in list order.
func batchesByStationSearchFloor(t *testing.T, db *sql.DB, p domain.ListBatchesByScanningStationParams) float64 {
	t.Helper()
	if p.Query == nil {
		return 0
	}
	var n float64
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM batch b JOIN item i ON i.id = b.item_id
		WHERE b.account_id = ? AND b.scanning_station_id = ? AND b.scanned_at IS NOT NULL AND i.sku LIKE ?`,
		p.AccountID, p.ScanningStationID, "%"+*p.Query+"%").Scan(&n))
	return n
}

// TestBatchesByStationList_ReadsAboutAPage holds every request ListBatchesByScanningStation accepts to
// reading about a page of batches (listPlanSuite).
func TestBatchesByStationList_ReadsAboutAPage(t *testing.T) {
	ensureBatchCorpus(t)
	listPlanSuite[domain.ListBatchesByScanningStationParams]{
		table: "batch", scopeColumn: "account_id",
		from: "FROM batch b", alias: "b",
		cases: batchesByStationPlanCases(),
		limit: func(p domain.ListBatchesByScanningStationParams) int32 { return p.Limit },
		list: func(ctx context.Context, q *sqlc.Queries, p domain.ListBatchesByScanningStationParams) error {
			if _, apiErr := NewBatchRepo(q).FindByScanningStation(ctx, p); apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: batchesByStationSearchFloor,
	}.run(t)
}
