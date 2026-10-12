//go:build plans

package repository

import (
	"context"
	"database/sql"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/core-service/internal/scheduling"
)

// fillDeliveryFacts computes delivery_order_fact for every order in the database and marks the facts
// ready, as a completed reconcile pass would. The cleanup empties them again, so the other delivery
// suites keep measuring the live query.
func fillDeliveryFacts(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	repo := NewDeliveryFactRepo(sqlc.New(db))
	_, err := db.Exec("DELETE FROM delivery_order_fact")
	require.NoError(t, err)
	after := ""
	for {
		ids, apiErr := repo.ListSalesOrderIDsAfter(ctx, after, 500)
		require.Nil(t, apiErr)
		for start := 0; start < len(ids); start += 100 {
			facts, apiErr := repo.ComputeFacts(ctx, ids[start:min(start+100, len(ids))])
			require.Nil(t, apiErr)
			require.Nil(t, repo.WriteFacts(ctx, facts, nil))
		}
		if len(ids) < 500 {
			break
		}
		after = ids[len(ids)-1]
	}
	now := time.Now().UTC()
	require.Nil(t, repo.SaveSync(ctx, domain.DeliveryFactSync{PassStartedAt: &now, LastCompletedAt: &now}))
	_, err = db.Exec("ANALYZE TABLE delivery_order_fact")
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM sales_sync WHERE name = ?", deliveryFactSyncName)
		require.NoError(t, err)
		_, err = db.Exec("DELETE FROM delivery_order_fact")
		require.NoError(t, err)
	})
}

// TestDeliveryFacts_MatchLiveQuery holds every plan-tested delivery request to returning, from
// delivery_order_fact, exactly what the live query returns: the same orders in the same order, the same
// quantities, customers, groups and product lines.
func TestDeliveryFacts_MatchLiveQuery(t *testing.T) {
	ensureDeliveryPerformanceCorpus(t)
	db := planDB(t)
	fillDeliveryFacts(t, db)
	repo := &productionScheduleInputRepoImpl{queries: sqlc.New(db)}
	ctx := context.Background()

	sortLines := func(outcomes []scheduling.DeliveryOutcome) {
		for _, o := range outcomes {
			slices.SortFunc(o.ProductLines, func(a, b scheduling.ProductLineRef) int { return strings.Compare(a.ID, b.ID) })
		}
	}
	matched := 0
	for _, c := range deliveryPerfPlanCases() {
		t.Run(c.name, func(t *testing.T) {
			args, apiErr := repo.deliveryFilterArgs(ctx, planAnaAccount, c.params.filters)
			require.Nil(t, apiErr)
			if args.none {
				return
			}
			live, apiErr := repo.listDeliveryOutcomesLive(ctx, planAnaAccount, c.params.start, c.params.end, args)
			require.Nil(t, apiErr)
			facts, err := repo.listDeliveryOutcomesFromFacts(ctx, planAnaAccount, c.params.start, c.params.end, args)
			require.NoError(t, err)
			sortLines(live)
			sortLines(facts)
			if len(live) == 0 {
				require.Empty(t, facts)
				return
			}
			require.Equal(t, live, facts)
			matched++
		})
	}
	require.Positive(t, matched, "no case returned any order")

	// The golden results recorded from the live query hold through the facts too.
	TestDeliveryPerformance_ResultsUnchanged(t)
}

// TestDeliveryFacts_ReadTheWindow holds the fact read to the window's rows: each of its two statements
// reads the account's range of the clustered key between the window's dates and nothing more.
func TestDeliveryFacts_ReadTheWindow(t *testing.T) {
	ensureDeliveryPerformanceCorpus(t)
	db := planDB(t)
	fillDeliveryFacts(t, db)
	edb := &explainingDB{db: db, explainRows: true}
	repo := NewProductionScheduleInputRepo(sqlc.New(edb))
	ctx := context.Background()
	for _, c := range deliveryPerfPlanCases() {
		t.Run(c.name, func(t *testing.T) {
			edb.statements = nil
			_, apiErr := repo.ListDeliveryOutcomes(ctx, planAnaAccount, c.params.start, c.params.end, c.params.filters)
			require.Nil(t, apiErr)
			var window float64
			require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM delivery_order_fact
				WHERE owner_account_id = ? AND ship_by_date >= ? AND ship_by_date <= ?`, planAnaAccount, c.params.start, c.params.end).Scan(&window))
			var read float64
			for _, stmt := range edb.statements {
				if strings.Contains(stmt.query, "FROM delivery_order_fact f") {
					a := tableAccess(stmt.plan, "f")
					read += a.rows
					require.NotContains(t, stmt.plan, "Table scan on f", "the fact read must be a range of its key\n%s", stmt.plan)
				}
			}
			require.LessOrEqual(t, read, 2*window+aggregateSlack, "the fact read covers more than the window's rows")
		})
	}
}
