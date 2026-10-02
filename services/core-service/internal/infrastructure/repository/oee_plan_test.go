//go:build plans

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// The OEE corpus is a plant's last four years of scans: tens of thousands of batches across six
// stations in three departments, most on two busy stations, each scan tied to the machines that ran
// it; and a downtime log of short stops most days, a few long ones, and a handful still open.
const (
	planOeeAccount       = "ac_planoee"
	planOeeBatches       = 60_000
	planOeeStations      = 6
	planOeeMachines      = 10
	planOeeDowntime      = 1_500
	planOeeCorpusVersion = "Plan OEE Plant v1"
)

var planOeeOrigin = time.Date(2022, 10, 1, 0, 0, 0, 0, time.UTC)

func planOeeScannedAt(i int) time.Time {
	return planOeeOrigin.Add(time.Duration(i) * (4 * 365 * 24 * time.Hour / planOeeBatches))
}

func planOeeStation(i int) int {
	if r := planHash(i, 31) % 100; r < 70 {
		return r % 2
	}
	return 2 + planHash(i, 32)%(planOeeStations-2)
}

var planOeeOnce sync.Once

func ensureOeeCorpus(t *testing.T) {
	t.Helper()
	db := planDB(t)
	planOeeOnce.Do(func() {
		var version string
		_ = db.QueryRow("SELECT name FROM account WHERE id = ?", planOeeAccount).Scan(&version)
		if version == planOeeCorpusVersion {
			return
		}
		t.Log("seeding the OEE plan corpus; it is kept for later runs")
		exec := func(query string, args ...any) {
			_, err := db.Exec(query, args...)
			require.NoError(t, err)
		}
		exec("DELETE FROM _batches_machines WHERE A LIKE 'ba\\_planoee\\_%'")
		exec("DELETE FROM batch WHERE account_id = ?", planOeeAccount)
		exec("DELETE FROM quantity WHERE id LIKE 'qy\\_planoee\\_%'")
		exec("DELETE FROM machine_downtime_event WHERE account_id = ?", planOeeAccount)
		exec("DELETE FROM scanning_station WHERE account_id = ?", planOeeAccount)
		exec("DELETE FROM department WHERE account_id = ?", planOeeAccount)
		exec(`INSERT INTO account (id, name, account_type_code, onboarding_status_code) VALUES (?, 'pending', 'company', 'active')
		      ON DUPLICATE KEY UPDATE name = 'pending'`, planOeeAccount)
		exec(`INSERT IGNORE INTO unit (id, name, abbreviation, unit_dimension_code, account_id,
		                               ratio_numerator, ratio_denominator, is_base_unit, created_at, updated_at)
		      VALUES ('un_planoee', 'plantest', 'un_planoee', 'quantity', NULL, 1, 1, 0, NOW(3), NOW(3))`)
		for d := range 3 {
			exec("INSERT INTO department (id, name, account_id) VALUES (?, ?, ?)", fmt.Sprintf("dp_planoee_%d", d), fmt.Sprintf("Plan Dept %d", d), planOeeAccount)
		}
		for s := range planOeeStations {
			exec(`INSERT INTO scanning_station (id, name, department_id, account_id, scanning_station_type_code) VALUES (?, ?, ?, ?, 'production')`,
				fmt.Sprintf("ss_planoee_%d", s), fmt.Sprintf("Plan Station %d", s), fmt.Sprintf("dp_planoee_%d", s%3), planOeeAccount)
		}

		const batch = 1_000
		for start := 0; start < planOeeBatches; start += batch {
			var qVals, bVals, mVals []string
			var qArgs, bArgs, mArgs []any
			for i := start; i < start+batch; i++ {
				at := planOeeScannedAt(i)
				id, qID := fmt.Sprintf("ba_planoee_%07d", i), fmt.Sprintf("qy_planoee_%07d", i)
				qVals = append(qVals, "(?, ?, 'un_planoee', ?, ?)")
				qArgs = append(qArgs, qID, 1+planHash(i, 33)%200, at, at)
				bVals = append(bVals, "(?, ?, ?, ?, ?, ?, ?)")
				bArgs = append(bArgs, id, planOeeAccount, fmt.Sprintf("it_planoee_%03d", planHash(i, 34)%300), qID,
					fmt.Sprintf("ss_planoee_%d", planOeeStation(i)), at, at)
				mVals = append(mVals, "(?, ?)")
				mArgs = append(mArgs, id, fmt.Sprintf("mc_planoee_%02d", planHash(i, 35)%planOeeMachines))
			}
			exec("INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES "+strings.Join(qVals, ","), qArgs...)
			exec("INSERT INTO batch (id, account_id, item_id, quantity_id, scanning_station_id, scanned_at, created_at) VALUES "+strings.Join(bVals, ","), bArgs...)
			exec("INSERT INTO _batches_machines (A, B) VALUES "+strings.Join(mVals, ","), mArgs...)
		}

		reasons := []string{"breakdown", "changeover", "minor_stop", "material_shortage", "no_operator"}
		var dVals []string
		var dArgs []any
		for i := range planOeeDowntime {
			started := planOeeOrigin.Add(time.Duration(i) * (4 * 365 * 24 * time.Hour / planOeeDowntime))
			var ended any = started.Add(time.Duration(10+planHash(i, 36)%240) * time.Minute)
			switch {
			case i >= planOeeDowntime-3:
				ended = nil
			case i%100 == 7:
				ended = started.Add(9 * 24 * time.Hour)
			}
			dVals = append(dVals, "(?, ?, ?, ?, ?, ?, ?, ?, 'us_planoee')")
			dArgs = append(dArgs, fmt.Sprintf("md_planoee_%05d", i), planOeeAccount, fmt.Sprintf("mc_planoee_%02d", i%planOeeMachines),
				fmt.Sprintf("dp_planoee_%d", i%3), reasons[i%len(reasons)], started, ended, started.Format("2006-01-02"))
		}
		exec(`INSERT INTO machine_downtime_event (id, account_id, machine_id, department_id, reason_code, started_at, ended_at, shift_date, reported_by_id)
		      VALUES `+strings.Join(dVals, ","), dArgs...)
		exec("UPDATE account SET name = ? WHERE id = ?", planOeeCorpusVersion, planOeeAccount)
		exec("ANALYZE TABLE batch, _batches_machines, machine_downtime_event")
	})
}

// oeePlanRequest is one OEE read: a window, optionally scoped to the plan's machines.
type oeePlanRequest struct {
	start, end time.Time
	machines   []string
}

func oeePlanCases() []planCase[oeePlanRequest] {
	last := planOeeScannedAt(planOeeBatches - 1)
	window := func(start, end time.Time) func(*oeePlanRequest) {
		return func(p *oeePlanRequest) { p.start, p.end = start, end }
	}
	return planCases(oeePlanRequest{}, []planDim[oeePlanRequest]{
		{"machines", []planValue[oeePlanRequest]{
			{"machines=most", func(p *oeePlanRequest) {
				for m := range planOeeMachines - 2 {
					p.machines = append(p.machines, fmt.Sprintf("mc_planoee_%02d", m))
				}
			}},
			{"machines=one", func(p *oeePlanRequest) { p.machines = []string{"mc_planoee_09"} }},
		}},
	}, []planValue[oeePlanRequest]{
		{"week", window(last.AddDate(0, 0, -7), last)},
		{"quarter", window(last.AddDate(0, -3, 0), last)},
		{"1y", window(last.AddDate(-1, 0, 0), last)},
		{"old-month", window(time.Date(2023, 5, 1, 0, 0, 0, 0, time.UTC), time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC))},
	})
}

func (p oeePlanRequest) window() domain.GetOeeWindowParams {
	return domain.GetOeeWindowParams{AccountID: planOeeAccount, StartDate: p.start, EndDate: p.end, MachineIDs: p.machines}
}

// oeePlanFloor is an OEE read's scope: the window's scans for each output read (the machine scope is a
// join table without the scan time, so no key on the scans can pin it), and the downtime events an
// index bounded on one side can reach. An overlap with the window bounds an interval on both ends, and a
// B-tree only on one: events started before the window's end, or ended (or still open) after its start,
// whichever is fewer.
func oeePlanFloor(t *testing.T, db *sql.DB, p oeePlanRequest) map[string]float64 {
	t.Helper()
	var scans, startedBefore, endedAfter float64
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM batch WHERE account_id = ? AND scanned_at >= ? AND scanned_at <= ?", planOeeAccount, p.start, p.end).Scan(&scans))
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM machine_downtime_event WHERE account_id = ? AND started_at <= ?", planOeeAccount, p.end).Scan(&startedBefore))
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM machine_downtime_event WHERE account_id = ? AND (ended_at >= ? OR ended_at IS NULL)", planOeeAccount, p.start).Scan(&endedAfter))
	return map[string]float64{"b": 2 * scans, "e": math.Min(startedBefore, endedAfter)}
}

// TestOee_ReadsItsScope holds the OEE reads (the department output, weekly for the trend, and the
// downtime intervals both use) to reading only the window's scans and the downtime it can reach
// (aggregatePlanSuite).
func TestOee_ReadsItsScope(t *testing.T) {
	ensureOeeCorpus(t)
	aggregatePlanSuite[oeePlanRequest]{
		tables: []aggregateTable{
			{table: "batch", scopeColumn: "account_id", from: "FROM batch b", alias: "b"},
			{table: "machine_downtime_event", scopeColumn: "account_id", from: "FROM machine_downtime_event e", alias: "e"},
		},
		cases: oeePlanCases(),
		report: func(ctx context.Context, q *sqlc.Queries, p oeePlanRequest) error {
			repo := NewAnalyticsRepo(q)
			if _, apiErr := repo.GetOeeDepartmentData(ctx, p.window()); apiErr != nil {
				return apiErr
			}
			if _, apiErr := repo.GetOeeTrendDepartmentDataByWeek(ctx, p.window()); apiErr != nil {
				return apiErr
			}
			if _, apiErr := repo.GetOeeDowntimeIntervals(ctx, p.window()); apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: oeePlanFloor,
	}.run(t)
}

// TestOee_ResultsUnchanged pins what every plan-tested OEE read returns.
func TestOee_ResultsUnchanged(t *testing.T) {
	ensureOeeCorpus(t)
	repo := NewAnalyticsRepo(sqlc.New(planDB(t)))
	ctx := context.Background()
	got := map[string]string{}
	for _, c := range oeePlanCases() {
		dept, apiErr := repo.GetOeeDepartmentData(ctx, c.params.window())
		require.Nil(t, apiErr)
		trend, apiErr := repo.GetOeeTrendDepartmentDataByWeek(ctx, c.params.window())
		require.Nil(t, apiErr)
		downtime, apiErr := repo.GetOeeDowntimeIntervals(ctx, c.params.window())
		require.Nil(t, apiErr)
		// Open events end "now", which moves; their start pins them.
		for i := range downtime {
			if downtime[i].EndedAt.After(time.Now().Add(-time.Hour)) {
				downtime[i].EndedAt = time.Time{}
			}
		}
		got[c.name] = planDigest(t, struct{ Dept, Trend, Downtime any }{unorderedRows(t, dept), unorderedRows(t, trend), downtime})
	}
	checkPlanResults(t, "oee.json", got)
}
