//go:build plans

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// The schedule history corpus is a plant's last four years of batches: a knitting department (the
// constraint) with most of the machines and three finishing departments after it, a third of batches
// scanned. A step names its department directly or through the station it is scanned at, and a few
// steps name a department other than their station's. Constraint batches are scanned on knitting
// machines; finishing ones mostly on no machine; and a few finishing scans are recorded against a
// knitting machine, which the constraint read must leave out.
const (
	planSchdAccount       = "ac_planschd"
	planSchdBatches       = 150_000
	planSchdDepartments   = 4
	planSchdStations      = 8
	planSchdSteps         = 600
	planSchdKnitMachines  = 30
	planSchdMachines      = 40
	planSchdItems         = 300
	planSchdRuns          = 200
	planSchdCorpusVersion = "Plan Schedule History v1"
)

var planSchdOrigin = time.Date(2022, 10, 1, 0, 0, 0, 0, time.UTC)

func planSchdCreatedAt(i int) time.Time {
	return planSchdOrigin.Add(time.Duration(i) * (4 * 365 * 24 * time.Hour / planSchdBatches))
}

func planSchdDept(d int) string    { return fmt.Sprintf("dp_planschd_%d", d) }
func planSchdStation(s int) string { return fmt.Sprintf("ss_planschd_%d", s) }
func planSchdStep(s int) string    { return fmt.Sprintf("ps_planschd_%04d", s) }
func planSchdMachine(m int) string { return fmt.Sprintf("mc_planschd_%02d", m) }
func planSchdItem(n int) string    { return fmt.Sprintf("it_planschd_%03d", n) }

// planSchdStepDept is the department a step belongs to: every fifth names one directly, the one after
// its station's; the rest take their station's.
func planSchdStepDept(s int) int {
	station := (s % planSchdStations) % planSchdDepartments
	if s%5 == 0 {
		return (station + 1) % planSchdDepartments
	}
	return station
}

var planSchdOnce sync.Once

func ensureScheduleHistoryCorpus(t *testing.T) {
	t.Helper()
	db := planDB(t)
	planSchdOnce.Do(func() {
		var version string
		_ = db.QueryRow("SELECT name FROM account WHERE id = ?", planSchdAccount).Scan(&version)
		if version == planSchdCorpusVersion {
			return
		}
		t.Logf("seeding the schedule history plan corpus (%d batches); it is kept for later runs", planSchdBatches)
		exec := func(query string, args ...any) {
			_, err := db.Exec(query, args...)
			require.NoError(t, err)
		}
		exec("DELETE FROM _batches_machines WHERE A LIKE 'ba\\_planschd\\_%'")
		exec("DELETE FROM batch WHERE account_id = ?", planSchdAccount)
		exec("DELETE FROM quantity WHERE id LIKE 'qy\\_planschd\\_%'")
		exec("DELETE FROM rate WHERE id LIKE 'rt\\_planschd\\_%'")
		for _, table := range []string{"production_run", "production_step", "machine", "item", "scanning_station", "department"} {
			exec("DELETE FROM `"+table+"` WHERE account_id = ?", planSchdAccount)
		}
		exec(`INSERT INTO account (id, name, account_type_code, onboarding_status_code) VALUES (?, 'pending', 'company', 'active')
		      ON DUPLICATE KEY UPDATE name = 'pending'`, planSchdAccount)
		exec(`INSERT IGNORE INTO unit (id, name, abbreviation, unit_dimension_code, account_id,
		                               ratio_numerator, ratio_denominator, is_base_unit, created_at, updated_at)
		      VALUES ('un_planschd', 'planschd', 'un_planschd', 'quantity', NULL, 1, 1, 0, NOW(3), NOW(3))`)
		for d := range planSchdDepartments {
			exec("INSERT INTO department (id, name, account_id) VALUES (?, ?, ?)", planSchdDept(d), fmt.Sprintf("Plan Dept %d", d), planSchdAccount)
		}
		for s := range planSchdStations {
			exec(`INSERT INTO scanning_station (id, name, department_id, account_id, scanning_station_type_code) VALUES (?, ?, ?, ?, 'production')`,
				planSchdStation(s), fmt.Sprintf("Plan Station %d", s), planSchdDept(s%planSchdDepartments), planSchdAccount)
		}

		var rVals, sVals []string
		var rArgs, sArgs []any
		rate := func(id string, value int) {
			rVals = append(rVals, "(?, ?, 'un_planschd', 'un_planschd')")
			rArgs = append(rArgs, id, value)
		}
		for s := range planSchdSteps {
			lt, lr, or := fmt.Sprintf("rt_planschd_lt_%04d", s), fmt.Sprintf("rt_planschd_lr_%04d", s), fmt.Sprintf("rt_planschd_or_%04d", s)
			rate(lt, 1+s%7)
			rate(lr, 20+s%5)
			rate(or, 5+s%3)
			var dept any
			if s%5 == 0 {
				dept = planSchdDept(planSchdStepDept(s))
			}
			sVals = append(sVals, "(?, ?, ?, ?, ?, ?, ?, ?)")
			sArgs = append(sArgs, planSchdStep(s), fmt.Sprintf("Plan Step %d", s), planSchdAccount, planSchdStation(s%planSchdStations), lr, lt, or, dept)
		}
		var iVals []string
		var iArgs []any
		for n := range planSchdItems {
			uc := fmt.Sprintf("rt_planschd_uc_%03d", n)
			rate(uc, 3+n%11)
			iVals = append(iVals, "(?, ?, ?, ?, ?, ?, 'product', 'ic_planschd')")
			iArgs = append(iArgs, planSchdItem(n), fmt.Sprintf("PLANSCHD-%03d", n), fmt.Sprintf("uv_planschd_%03d", n),
				fmt.Sprintf("br_planschd_%03d", n), uc, planSchdAccount)
		}
		exec("INSERT INTO rate (id, value, numerator_unit_id, denominator_unit_id) VALUES "+strings.Join(rVals, ","), rArgs...)
		exec(`INSERT INTO production_step (id, name, account_id, scanning_station_id, labor_rate_id, labor_time_id, overhead_rate_id, department_id)
		      VALUES `+strings.Join(sVals, ","), sArgs...)
		exec(`INSERT INTO item (id, sku, unit_value_id, burn_rate_id, unit_cost_id, account_id, item_type_code, item_category_id)
		      VALUES `+strings.Join(iVals, ","), iArgs...)
		for m := range planSchdMachines {
			dept := 0
			if m >= planSchdKnitMachines {
				dept = 1 + m%(planSchdDepartments-1)
			}
			exec(`INSERT INTO machine (id, account_id, name, serial_number, department_id) VALUES (?, ?, ?, ?, ?)`,
				planSchdMachine(m), planSchdAccount, fmt.Sprintf("Plan Machine %02d", m), fmt.Sprintf("SN-%02d", m), planSchdDept(dept))
		}
		for r := range planSchdRuns {
			created := planSchdOrigin.Add(time.Duration(r) * (4 * 365 * 24 * time.Hour / planSchdRuns))
			exec(`INSERT INTO production_run (id, responsible_user_id, number, account_id, created_at, updated_at) VALUES (?, 'acus_planschd', ?, ?, ?, ?)`,
				fmt.Sprintf("pr_planschd_%03d", r), fmt.Sprintf("PR-%03d", r), planSchdAccount, created, created)
		}

		const chunk = 1_000
		for start := 0; start < planSchdBatches; start += chunk {
			var qVals, bVals, mVals []string
			var qArgs, bArgs, mArgs []any
			for i := start; i < start+chunk; i++ {
				created := planSchdCreatedAt(i)
				step := planHash(i, 41) % planSchdSteps
				var scanned any
				if i%3 == 0 {
					scanned = created.Add(time.Hour)
				}
				id, qID := fmt.Sprintf("ba_planschd_%07d", i), fmt.Sprintf("qy_planschd_%07d", i)
				qVals = append(qVals, "(?, ?, 'un_planschd', ?, ?)")
				qArgs = append(qArgs, qID, 1+planHash(i, 42)%200, created, created)
				bVals = append(bVals, "(?, ?, ?, ?, ?, ?, ?, ?, ?)")
				bArgs = append(bArgs, id, planSchdAccount, planSchdItem(planHash(i, 43)%planSchdItems), qID,
					planSchdStep(step), fmt.Sprintf("pr_planschd_%03d", i*planSchdRuns/planSchdBatches), scanned, created, created)
				machine := -1
				switch r := planHash(i, 44) % 100; {
				case planSchdStepDept(step) == 0 && r < 95:
					machine = r % planSchdKnitMachines
				case planSchdStepDept(step) != 0 && r < 5:
					machine = r % planSchdKnitMachines
				case planSchdStepDept(step) != 0 && r < 35:
					machine = planSchdKnitMachines + r%(planSchdMachines-planSchdKnitMachines)
				}
				if machine >= 0 {
					mVals = append(mVals, "(?, ?)")
					mArgs = append(mArgs, id, planSchdMachine(machine))
				}
			}
			exec("INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES "+strings.Join(qVals, ","), qArgs...)
			exec(`INSERT INTO batch (id, account_id, item_id, quantity_id, production_step_id, production_run_id, scanned_at, created_at, updated_at)
			      VALUES `+strings.Join(bVals, ","), bArgs...)
			exec("INSERT INTO _batches_machines (A, B) VALUES "+strings.Join(mVals, ","), mArgs...)
		}
		exec("UPDATE account SET name = ? WHERE id = ?", planSchdCorpusVersion, planSchdAccount)
		exec("ANALYZE TABLE batch, _batches_machines, production_step, item")
	})
}

// The history reads as they were: the constraint's batches found through every batch its machines
// ever ran, the finishing stages' through the items' unwindowed key.
const (
	constraintHistoryReference = `SELECT b.id, bm.B FROM batch b
JOIN _batches_machines bm ON bm.A = b.id
JOIN item i ON i.id = b.item_id
LEFT JOIN production_step ps ON ps.id = b.production_step_id
LEFT JOIN scanning_station ss ON ss.id = ps.scanning_station_id
WHERE b.account_id = ? AND b.scanned_at IS NOT NULL AND b.scanned_at >= ? AND b.scanned_at <= ?
  AND bm.B IN (%s) AND COALESCE(ps.department_id, ss.department_id) = ?
ORDER BY bm.B, b.scanned_at, b.id`
	finishingHistoryReference = `SELECT b.id, COALESCE(bm.B, '') FROM batch b IGNORE INDEX (batch_item_scanned_idx)
LEFT JOIN _batches_machines bm ON bm.A = b.id
JOIN item i ON i.id = b.item_id
LEFT JOIN production_step ps ON ps.id = b.production_step_id
LEFT JOIN scanning_station ss ON ss.id = ps.scanning_station_id
WHERE b.account_id = ? AND b.scanned_at IS NOT NULL AND b.scanned_at >= ? AND b.scanned_at <= ?
  AND b.item_id IN (%s) AND COALESCE(ps.department_id, ss.department_id, '') != ?
ORDER BY b.item_id, b.scanned_at, b.id`
	runRateHistoryReference = `SELECT COALESCE(bm.B, '') FROM batch b IGNORE INDEX (batch_item_scanned_idx)
LEFT JOIN _batches_machines bm ON bm.A = b.id
JOIN production_step ps ON ps.id = b.production_step_id
JOIN rate labor_time ON labor_time.id = ps.labor_time_id
WHERE b.account_id = ? AND b.item_id = ? AND b.scanned_at IS NOT NULL
ORDER BY b.scanned_at DESC, b.id DESC
LIMIT ?`
)

func referenceRows(t *testing.T, db *sql.DB, query string, args ...any) []string {
	t.Helper()
	rows, err := db.Query(query, args...)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	cols, err := rows.Columns()
	require.NoError(t, err)
	var out []string
	for rows.Next() {
		vals := make([]string, len(cols))
		ptrs := make([]any, len(cols))
		for k := range vals {
			ptrs[k] = &vals[k]
		}
		require.NoError(t, rows.Scan(ptrs...))
		out = append(out, strings.Join(vals, "|"))
	}
	require.NoError(t, rows.Err())
	return out
}

// historyBatchReads is the batch rows the statements reading batch b read, and the rows they returned.
func historyBatchReads(edb *explainingDB) (read, returned float64, plans []string) {
	for _, stmt := range edb.statements {
		if !strings.Contains(stmt.query, "FROM batch b") {
			continue
		}
		read += tableAccess(stmt.plan, "b").rows
		returned += planReturned(stmt.plan)
		plans = append(plans, stmt.plan)
	}
	return read, returned, plans
}

// TestScheduleHistory_ReadsTheWindowOnly requires the schedule's batch history reads to return the
// batches the queries they replaced returned, in the same order, and to read no more batch rows than
// their window holds: the constraint department's batches, or the finished goods', in the window, never
// their whole history; and an item's run rate only the samples it returns.
func TestScheduleHistory_ReadsTheWindowOnly(t *testing.T) {
	ensureScheduleHistoryCorpus(t)
	db := planDB(t)
	ctx := context.Background()
	edb := &explainingDB{db: db}
	repo := &productionScheduleInputRepoImpl{queries: sqlc.New(edb)}

	end := planSchdCreatedAt(planSchdBatches - 1)
	windows := []struct {
		name       string
		start, end time.Time
	}{
		{"quarter", end.AddDate(0, -3, 0), end},
		{"year", end.AddDate(-1, 0, 0), end},
		{"old month", time.Date(2023, 5, 1, 0, 0, 0, 0, time.UTC), time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)},
	}
	var knitting, oneKnitter []string
	for m := range planSchdKnitMachines {
		knitting = append(knitting, planSchdMachine(m))
	}
	oneKnitter = knitting[7:8]
	var items, fewItems []string
	for n := range planSchdItems {
		items = append(items, planSchdItem(n))
	}
	fewItems = items[:5]

	for _, stats := range planStatsModesFor(t, db, []string{"batch"}) {
		t.Run("stats="+stats, func(t *testing.T) {
			usePlanStats(t, db, "batch", stats)
			t.Cleanup(func() { usePlanStats(t, db, "batch", "analyzed") })
			for _, w := range windows {
				for _, machines := range [][]string{knitting, oneKnitter} {
					t.Run(fmt.Sprintf("constraint/%s/%d machines", w.name, len(machines)), func(t *testing.T) {
						want := referenceRows(t, db, fmt.Sprintf(constraintHistoryReference, placeholders(len(machines))),
							append(append([]any{planSchdAccount, w.start, w.end}, stringsToAny(machines)...), planSchdDept(0))...)
						edb.statements = nil
						rows, apiErr := repo.GetConstraintBatchMeasurements(ctx, domain.GetConstraintBatchMeasurementsParams{
							AccountID: planSchdAccount, WindowStart: w.start, WindowEnd: w.end,
							MachineIDs: machines, ConstraintDepartmentID: planSchdDept(0),
						})
						require.Nil(t, apiErr)
						got := make([]string, len(rows))
						for k, row := range rows {
							got[k] = row.Measurement.BatchID + "|" + row.Measurement.MachineID
						}
						require.NotEmpty(t, want)
						require.Equal(t, want, got)

						// The window's batches of the department's steps, whichever machines ran them: no key holds a
						// machine's batches by scan time, and the plan reads all of its machines.
						read, _, plans := historyBatchReads(edb)
						var inWindow int
						require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM batch b JOIN production_step ps ON ps.id = b.production_step_id
							LEFT JOIN scanning_station ss ON ss.id = ps.scanning_station_id
							WHERE b.account_id = ? AND b.scanned_at >= ? AND b.scanned_at <= ? AND COALESCE(ps.department_id, ss.department_id) = ?`,
							planSchdAccount, w.start, w.end, planSchdDept(0)).Scan(&inWindow))
						require.LessOrEqual(t, read, float64(inWindow)+aggregateSlack, "read %.0f batch rows for a window of %d\n%s", read, inWindow, strings.Join(plans, "\n"))
					})
				}
				for _, items := range [][]string{items, fewItems} {
					t.Run(fmt.Sprintf("finishing/%s/%d items", w.name, len(items)), func(t *testing.T) {
						want := referenceRows(t, db, fmt.Sprintf(finishingHistoryReference, placeholders(len(items))),
							append(append([]any{planSchdAccount, w.start, w.end}, stringsToAny(items)...), planSchdDept(0))...)
						edb.statements = nil
						rows, apiErr := repo.GetFinishingBatchMeasurements(ctx, domain.GetFinishingBatchMeasurementsParams{
							AccountID: planSchdAccount, WindowStart: w.start, WindowEnd: w.end,
							ItemIDs: items, ConstraintDepartmentID: planSchdDept(0),
						})
						require.Nil(t, apiErr)
						got := make([]string, len(rows))
						for k, row := range rows {
							got[k] = row.Measurement.BatchID + "|" + row.Measurement.MachineID
						}
						require.NotEmpty(t, want)
						require.Equal(t, want, got)

						// A finished good's window holds its constraint-stage scans too, which the department test drops after the read.
						read, _, plans := historyBatchReads(edb)
						var inWindow int
						require.NoError(t, db.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM batch WHERE account_id = ? AND item_id IN (%s)
							AND scanned_at >= ? AND scanned_at <= ?`, placeholders(len(items))),
							append(append([]any{planSchdAccount}, stringsToAny(items)...), w.start, w.end)...).Scan(&inWindow))
						require.LessOrEqual(t, read, float64(inWindow)+aggregateSlack, "read %.0f batch rows for a window of %d\n%s", read, inWindow, strings.Join(plans, "\n"))
					})
				}
			}
			for _, item := range []string{planSchdItem(0), planSchdItem(123)} {
				t.Run("run rate/"+item, func(t *testing.T) {
					const limit = 20
					want := referenceRows(t, db, runRateHistoryReference, planSchdAccount, item, limit)
					edb.statements = nil
					rows, apiErr := repo.GetItemRunRateHistory(ctx, planSchdAccount, item, limit)
					require.Nil(t, apiErr)
					got := make([]string, len(rows))
					for k, row := range rows {
						got[k] = row.MachineID
					}
					require.Equal(t, want, got)
					require.Len(t, got, limit)

					read, _, plans := historyBatchReads(edb)
					require.LessOrEqual(t, read, float64(2*limit), "read %.0f batch rows for %d samples\n%s", read, limit, strings.Join(plans, "\n"))
				})
			}
		})
	}
}
