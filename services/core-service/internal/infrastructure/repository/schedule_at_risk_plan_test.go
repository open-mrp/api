//go:build plans

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// ListAtRiskOrders reads a schedule version's diagnostics and its campaign-to-order links
// (ListLineOrders). It is not paged: a version's links are its whole answer. What it owes is reading
// only that version's links, however many other versions the account keeps.
const (
	planScheduleSmall      = "psch_plansales_small"
	planScheduleLarge      = "psch_plansales_large"
	planScheduleSmallLinks = 40
	planScheduleLargeLinks = 4_000
)

func ensureScheduleLinkCorpus(t *testing.T) {
	t.Helper()
	ensureSalesCorpus(t)
	db := planDB(t)
	var have int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM production_schedule_line_order WHERE account_id = ?", planSalesAccount).Scan(&have))
	if have == planScheduleSmallLinks+planScheduleLargeLinks {
		return
	}
	for _, table := range []string{"production_schedule_line_order", "production_schedule_line"} {
		_, err := db.Exec("DELETE FROM "+table+" WHERE account_id = ?", planSalesAccount)
		require.NoError(t, err)
	}
	ins := &planInserter{t: t, db: db}
	week := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)
	for _, s := range []struct {
		id    string
		links int
	}{{planScheduleSmall, planScheduleSmallLinks}, {planScheduleLarge, planScheduleLargeLinks}} {
		for k := range s.links {
			lineID := fmt.Sprintf("pschl_plansales_%s_%05d", s.id[len(s.id)-5:], k)
			ins.add("production_schedule_line", "id, account_id, production_schedule_id, week_index, week_start_date, machine_id, item_id, planned_quantity, status_code, source_code",
				lineID, planSalesAccount, s.id, k%12, week, "mach_plansales", planSalesItemID(k%planSalesItems), 100, "planned", "solver")
			order := k * 7 % planSalesOrders
			ins.add("production_schedule_line_order", "id, account_id, production_schedule_id, production_schedule_line_id, sales_order_id, sales_order_line_id, allocated_quantity",
				fmt.Sprintf("pslo_plansales_%s_%05d", s.id[len(s.id)-5:], k), planSalesAccount, s.id, lineID, planSalesOrderID(order), planSalesLineID(order, 0), 10)
		}
	}
	ins.flush()
	_, err := db.Exec("ANALYZE TABLE production_schedule_line_order, production_schedule_line")
	require.NoError(t, err)
}

// TestScheduleAtRiskOrders_ReadsOnlyItsVersion holds ListLineOrders to reading one version's links.
func TestScheduleAtRiskOrders_ReadsOnlyItsVersion(t *testing.T) {
	ensureScheduleLinkCorpus(t)
	edb := &explainingDB{db: planDB(t)}
	repo := NewProductionScheduleRepo(sqlc.New(edb))
	for id, links := range map[string]int{planScheduleSmall: planScheduleSmallLinks, planScheduleLarge: planScheduleLargeLinks} {
		edb.statements = nil
		got, apiErr := repo.ListLineOrders(context.Background(), planSalesAccount, id)
		require.Nil(t, apiErr)
		require.Len(t, got, links)
		require.Len(t, edb.statements, 1)
		read := tableAccess(edb.statements[0].plan, "lo")
		assert.LessOrEqual(t, read.rows, float64(links), "%s read %v rows of production_schedule_line_order via %v\n%s",
			id, read.rows, read.indexes, edb.statements[0].plan)
	}
}
