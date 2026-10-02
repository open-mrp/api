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

// planRunBatchAliases are the batch-side tables the run list reads, under the aliases its statements use.
var planRunBatchAliases = []string{"b", "b2", "b3", "bq", "bm"}

func productionRunPlanCases() []planCase[domain.ListProductionRunsParams] {
	str := func(s string) *string { return &s }
	day := func(t time.Time) *string { return str(t.Format("2006-01-02")) }
	old := planBatchCreatedAt(planBatchRows / 4)
	mid := planBatchCreatedAt(planBatchRows / 2)
	cursor := func(dir pagination.Direction) func(*domain.ListProductionRunsParams) {
		return func(p *domain.ListProductionRunsParams) { p.Cursor = planCursorAt(mid, "pnrn_~", dir) }
	}
	return planCases(
		domain.ListProductionRunsParams{AccountID: planBatchAccount, Limit: 25},
		[]planDim[domain.ListProductionRunsParams]{
			{"status", []planValue[domain.ListProductionRunsParams]{
				{"open", func(p *domain.ListProductionRunsParams) { p.Status = str("open") }},
				{"closed", func(p *domain.ListProductionRunsParams) { p.Status = str("closed") }},
			}},
			{"item", []planValue[domain.ListProductionRunsParams]{
				{"item-common", func(p *domain.ListProductionRunsParams) { p.ItemIDs = []string{planBatchItemID(0)} }},
				{"item-rare", func(p *domain.ListProductionRunsParams) { p.ItemIDs = []string{planBatchItemID(planBatchItems - 1)} }},
			}},
			{"machine", []planValue[domain.ListProductionRunsParams]{
				{"machine-common", func(p *domain.ListProductionRunsParams) { p.MachineIDs = []string{planBatchMachineID(0)} }},
				{"machine-rare", func(p *domain.ListProductionRunsParams) {
					p.MachineIDs = []string{planBatchMachineID(planBatchRareMachine)}
				}},
			}},
			{"search", []planValue[domain.ListProductionRunsParams]{
				{"number", func(p *domain.ListProductionRunsParams) { p.Query = str("PR-00042") }},
				{"every", func(p *domain.ListProductionRunsParams) { p.Query = str("PR-") }},
				{"batch", func(p *domain.ListProductionRunsParams) { p.Query = str(planBatchID(1_000)) }},
			}},
			{"dates", []planValue[domain.ListProductionRunsParams]{
				{"old60d", func(p *domain.ListProductionRunsParams) {
					p.StartDate, p.EndDate = day(old), day(old.Add(60*24*time.Hour))
				}},
			}},
		},
		[]planValue[domain.ListProductionRunsParams]{
			{"first", func(*domain.ListProductionRunsParams) {}},
			{"deep-next", cursor(pagination.DirectionForward)},
			{"deep-prev", cursor(pagination.DirectionBackward)},
		},
	)
}

// runBatchFilterMatches is how many run batches a request's machine and batch-id filters match. No key
// leads with the run and either, so they are found from the batch side, and the bar for that is
// reading only their matches (as for an unordered filter).
func runBatchFilterMatches(t *testing.T, db *sql.DB, p domain.ListProductionRunsParams) float64 {
	t.Helper()
	var n, m float64
	if len(p.MachineIDs) > 0 {
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM _batches_machines bm JOIN batch b ON b.id = bm.A
			WHERE bm.B IN (`+placeholders(len(p.MachineIDs))+`) AND b.account_id = ? AND b.production_run_id IS NOT NULL`,
			append(stringArgs(p.MachineIDs), p.AccountID)...).Scan(&n))
	}
	if p.Query != nil {
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM batch WHERE account_id = ? AND id LIKE ? AND production_run_id IS NOT NULL`,
			p.AccountID, *p.Query+"%").Scan(&m))
	}
	return n + m
}

// TestProductionRunList_ReadsAboutAPage holds every filter combination ListProductionRuns accepts to
// reading about a page of runs (listPlanSuite), and to reading only the batches of the runs it returns
// (plus a probe per run its filters reject): each run's batch count and totals are of its own batches.
func TestProductionRunList_ReadsAboutAPage(t *testing.T) {
	ensureBatchCorpus(t)
	db := planDB(t)
	var page []string
	listPlanSuite[domain.ListProductionRunsParams]{
		table: "production_run", scopeColumn: "account_id",
		from: "FROM production_run pr", alias: "pr",
		cases: productionRunPlanCases(),
		limit: func(p domain.ListProductionRunsParams) int32 { return p.Limit },
		list: func(ctx context.Context, q *sqlc.Queries, p domain.ListProductionRunsParams) error {
			res, apiErr := NewProductionRunRepo(q).List(ctx, p)
			if apiErr != nil {
				return apiErr
			}
			page = page[:0]
			for _, run := range res.ProductionRuns {
				page = append(page, run.ID)
			}
			return nil
		},
		relatedTables: []string{"batch"},
		related: func(t *testing.T, stmts []explainedStatement, p domain.ListProductionRunsParams) {
			var pageBatches float64
			if len(page) > 0 {
				require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM batch WHERE account_id = ? AND production_run_id IN ("+
					placeholders(len(page))+")", append([]any{planBatchAccount}, stringArgs(page)...)...).Scan(&pageBatches))
			}
			budget := 2*pageBatches + 2*runBatchFilterMatches(t, db, p) + planRowBudget(p.Limit)
			for _, stmt := range stmts {
				var read float64
				var via []string
				for _, alias := range planRunBatchAliases {
					a := tableAccess(stmt.plan, alias)
					read += a.rows
					via = append(via, a.indexes...)
				}
				if read > budget {
					t.Errorf("read %.0f batch rows via %v; the page's runs hold %.0f\n%s\n%s",
						read, via, pageBatches, strings.TrimSpace(stmt.query), stmt.plan)
				}
			}
		},
	}.run(t)
}

// A machine common enough to walk the list key with an EXISTS probe returns the same pages, both ways,
// as reading its runs by id.
func TestProductionRunList_CommonMachineMatchesByID(t *testing.T) {
	ensureBatchCorpus(t)
	repo := NewProductionRunRepo(sqlc.New(planDB(t)))
	pages := func() [][]string {
		var out [][]string
		p := domain.ListProductionRunsParams{AccountID: planBatchAccount, Limit: 25, MachineIDs: []string{planBatchMachineID(0)}}
		for range 3 {
			res, apiErr := repo.List(context.Background(), p)
			require.Nil(t, apiErr)
			var ids []string
			for _, run := range res.ProductionRuns {
				ids = append(ids, run.ID)
			}
			out = append(out, ids)
			if res.PageInfo.NextCursor == nil {
				break
			}
			p.Cursor = res.PageInfo.NextCursor
		}
		// and back from the last page reached
		if p.Cursor != nil {
			res, apiErr := repo.List(context.Background(), p)
			require.Nil(t, apiErr)
			if res.PageInfo.PrevCursor != nil {
				p.Cursor = res.PageInfo.PrevCursor
				back, apiErr := repo.List(context.Background(), p)
				require.Nil(t, apiErr)
				var ids []string
				for _, run := range back.ProductionRuns {
					ids = append(ids, run.ID)
				}
				out = append(out, ids)
			}
		}
		return out
	}
	byID := pages()
	require.NotEmpty(t, byID[0])

	saved := productionRunCountedRatio
	productionRunCountedRatio = 0
	t.Cleanup(func() { productionRunCountedRatio = saved })
	require.Equal(t, byID, pages())
}
