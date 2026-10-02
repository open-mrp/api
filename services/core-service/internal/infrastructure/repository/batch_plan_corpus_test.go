//go:build plans

package repository

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// The batch corpus is one manufacturer's floor shaped like the largest production tenant: six stations
// with very different volumes, a third of batches scanned, and production runs whose batches are made
// on the busiest station and flow on to the others. A few items hold most of each station's batches,
// one item is rare, a run holds a few hundred batches and a rare run a handful, and machines are
// attached to most run batches. Row widths follow production: UUID station, item and machine IDs,
// short prefixed batch and quantity IDs.
const (
	planBatchAccount  = "ac_planbat"
	planBatchRows     = 60_000
	planBatchItems    = 2_000
	planBatchRuns     = 180
	planBatchMachines = 40
	planBatchSpan     = 2 * 365 * 24 * time.Hour

	// planBatchCorpusVersion is bumped whenever the corpus's shape changes, so a stale one is rebuilt.
	planBatchCorpusVersion = "Plan Test Batches v4"

	// planBatchRareSKU matches only the rare item's SKU; planBatchDenseSKU matches every item's.
	planBatchRareSKU  = "ZQRARE"
	planBatchDenseSKU = "PLANBAT"

	// planBatchRareRun holds only a few batches; planBatchBigRun is the busiest.
	planBatchRareRun = planBatchRuns - 1
	planBatchBigRun  = 0
)

// planBatchStationShares is each station's share of batches, in thousandths, busiest first; the rest
// are on no station (planned run batches not yet made).
var planBatchStationShares = []int{370, 240, 170, 160, 40, 15}

var planBatchOrigin = time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC)

func planBatchCreatedAt(i int) time.Time {
	return planBatchOrigin.Add(time.Duration(i) * (planBatchSpan / planBatchRows))
}

func planBatchID(i int) string        { return fmt.Sprintf("btch_%012d", i) }
func planBatchStationID(s int) string { return fmt.Sprintf("0000%04d-plan-4bst-8000-%012d", s, s) }
func planBatchItemID(n int) string    { return fmt.Sprintf("0000%04d-plan-4bit-8000-%012d", n, n) }
func planBatchRunID(r int) string     { return fmt.Sprintf("pnrn_planbat_%06d", r) }
func planBatchMachineID(m int) string { return fmt.Sprintf("0000%04d-plan-4bmc-8000-%012d", m, m) }
func planBatchItemSKU(n int) string {
	if n == planBatchItems-1 {
		return planBatchDenseSKU + "-" + planBatchRareSKU
	}
	return fmt.Sprintf("%s-%05d", planBatchDenseSKU, n)
}

// planBatchStation is the station batch i was made on, or -1 for none.
func planBatchStation(i int) int {
	slot := (i * 7) % 1000
	for s, share := range planBatchStationShares {
		if slot < share {
			return s
		}
		slot -= share
	}
	return -1
}

// planBatchScanned reports whether batch i has been scanned: a third of those on a station.
func planBatchScanned(i int) bool { return planBatchStation(i) >= 0 && i%3 == 0 }

var planBatchRareItemRows = map[int]bool{300: true, 30_000: true, 59_700: true}

// planBatchItem gives one item a tenth of the volume and spreads the rest; the last item is rare.
func planBatchItem(i int) int {
	if planBatchRareItemRows[i] {
		return planBatchItems - 1
	}
	if i%10 == 1 {
		return 0
	}
	return 1 + (i*7919)%(planBatchItems-2)
}

// planBatchRun is the run batch i belongs to, or -1. Runs take most batches on the busiest station and
// those on no station, in time order; the rare run holds three.
func planBatchRun(i int) int {
	switch {
	case i == 1_000 || i == 1_007 || i == 1_014:
		return planBatchRareRun
	case planBatchStation(i) != 0 && planBatchStation(i) != -1:
		return -1
	case i%4 == 3:
		return -1
	}
	// Runs are consecutive in time; the first run is the busiest.
	r := i * (planBatchRuns - 1) / planBatchRows
	if r > 0 && i%40 == 0 {
		return planBatchBigRun
	}
	return r
}

// planBatchRareMachine made only the rare run's batches.
const planBatchRareMachine = planBatchMachines - 1

// planBatchMachine is the machine run batch i was made on.
func planBatchMachine(i int) int {
	if planBatchRun(i) == planBatchRareRun {
		return planBatchRareMachine
	}
	return i % (planBatchMachines - 1)
}

var planBatchCorpusOnce sync.Once

func ensureBatchCorpus(t *testing.T) {
	t.Helper()
	db := planDB(t)
	planBatchCorpusOnce.Do(func() {
		var have int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM batch WHERE account_id = ?", planBatchAccount).Scan(&have))
		var version string
		_ = db.QueryRow("SELECT name FROM account WHERE id = ?", planBatchAccount).Scan(&version)
		if have >= planBatchRows && version == planBatchCorpusVersion {
			return
		}
		t.Logf("seeding the batch plan corpus (%d batches); it is kept for later runs", planBatchRows)

		exec := func(query string, args ...any) {
			_, err := db.Exec(query, args...)
			require.NoError(t, err)
		}
		exec("DELETE bm FROM _batches_machines bm JOIN batch b ON b.id = bm.A WHERE b.account_id = ?", planBatchAccount)
		exec("DELETE bf FROM _batch_flow bf JOIN batch b ON b.id = bf.A WHERE b.account_id = ?", planBatchAccount)
		exec("DELETE FROM batch WHERE account_id = ?", planBatchAccount)
		exec("DELETE FROM inventory_issue WHERE account_id = ?", planBatchAccount)
		exec("DELETE FROM quantity WHERE id LIKE 'qu\\_planbat\\_%'")
		exec("DELETE FROM production_run WHERE account_id = ?", planBatchAccount)
		exec("DELETE FROM machine WHERE account_id = ?", planBatchAccount)
		exec("DELETE FROM scanning_station WHERE account_id = ?", planBatchAccount)
		exec("DELETE FROM item WHERE account_id = ?", planBatchAccount)
		exec(`INSERT IGNORE INTO unit (id, name, abbreviation, unit_dimension_code, account_id,
		                               ratio_numerator, ratio_denominator, is_base_unit, created_at, updated_at)
		      VALUES ('un_planbat', 'planbat', 'un_planbat', 'quantity', NULL, 1, 1, 0, NOW(3), NOW(3))`)
		exec(`INSERT INTO account (id, name, account_type_code, onboarding_status_code) VALUES (?, ?, 'company', 'active')
		      ON DUPLICATE KEY UPDATE name = VALUES(name)`, planBatchAccount, planBatchCorpusVersion)
		exec("INSERT IGNORE INTO `user` (id, name) VALUES ('us_planbat_operator', 'Plan Batch Operator')")
		exec(`INSERT IGNORE INTO account_user (id, user_id, account_id) VALUES ('acus_planbat_operator', 'us_planbat_operator', ?)`, planBatchAccount)
		for s := range planBatchStationShares {
			exec(`INSERT INTO scanning_station (id, name, department_id, account_id, scanning_station_type_code, created_at, updated_at)
			      VALUES (?, ?, 'dept_planbat', ?, 'production', ?, ?)`, planBatchStationID(s), fmt.Sprintf("Plan Batch Station %d", s),
				planBatchAccount, planBatchOrigin, planBatchOrigin)
		}
		for m := range planBatchMachines {
			exec(`INSERT INTO machine (id, account_id, name, serial_number, department_id, created_at, updated_at)
			      VALUES (?, ?, ?, ?, 'dept_planbat', ?, ?)`, planBatchMachineID(m), planBatchAccount, fmt.Sprintf("Plan Machine %02d", m),
				fmt.Sprintf("SN-%04d", m), planBatchOrigin, planBatchOrigin)
		}

		var itemVals []string
		var itemArgs []any
		for n := range planBatchItems {
			itemVals = append(itemVals, "(?, ?, ?, ?, ?, ?, 'product', 'ic_planbat', ?, ?)")
			itemArgs = append(itemArgs, planBatchItemID(n), planBatchItemSKU(n), fmt.Sprintf("uv_planbat_%05d", n),
				fmt.Sprintf("br_planbat_%05d", n), fmt.Sprintf("uc_planbat_%05d", n), planBatchAccount, planBatchOrigin, planBatchOrigin)
		}
		exec(`INSERT INTO item (id, sku, unit_value_id, burn_rate_id, unit_cost_id, account_id, item_type_code,
		      item_category_id, created_at, updated_at) VALUES `+strings.Join(itemVals, ","), itemArgs...)

		// A run is created when its first batch is; two in three are complete.
		runCreated := map[int]time.Time{}
		for i := range planBatchRows {
			if r := planBatchRun(i); r >= 0 {
				if _, ok := runCreated[r]; !ok {
					runCreated[r] = planBatchCreatedAt(i).Add(-time.Hour)
				}
			}
		}
		var runVals []string
		var runArgs []any
		for r := range planBatchRuns {
			created, ok := runCreated[r]
			if !ok {
				created = planBatchOrigin
			}
			var completed any
			if r%3 != 2 {
				completed = created.Add(14 * 24 * time.Hour)
			}
			runVals = append(runVals, "(?, 'acus_planbat_operator', ?, ?, ?, ?, ?, ?)")
			runArgs = append(runArgs, planBatchRunID(r), fmt.Sprintf("PR-%05d", r), planBatchAccount, created, created, created, completed)
		}
		exec(`INSERT INTO production_run (id, responsible_user_id, number, account_id, created_at, updated_at, started_at, completed_at)
		      VALUES `+strings.Join(runVals, ","), runArgs...)

		const batch = 1_000
		for start := 0; start < planBatchRows; start += batch {
			var qVals, bVals, fVals, mVals, iVals []string
			var qArgs, bArgs, fArgs, mArgs, iArgs []any
			for i := start; i < start+batch; i++ {
				createdAt := planBatchCreatedAt(i)
				var station, scanned, closed, run any
				if s := planBatchStation(i); s >= 0 {
					station = planBatchStationID(s)
				}
				if planBatchScanned(i) {
					scanned = createdAt.Add(30 * time.Minute)
				}
				if i < planBatchRows*95/100 {
					closed = createdAt.Add(2 * time.Hour)
				}
				if r := planBatchRun(i); r >= 0 {
					run = planBatchRunID(r)
					mVals = append(mVals, "(?, ?)")
					mArgs = append(mArgs, planBatchID(i), planBatchMachineID(planBatchMachine(i)))
					// Run batches consume material: two issues each.
					for k := range 2 {
						iVals = append(iVals, "(?, ?, ?, 'issued', ?, ?, ?, ?, ?)")
						iArgs = append(iArgs, fmt.Sprintf("inis_planbat_%06d_%d", i, k), planBatchAccount,
							planBatchItemID((i*13+k)%planBatchItems), fmt.Sprintf("qu_planbat_is_%06d_%d", i, k), planBatchID(i),
							createdAt, createdAt, createdAt)
					}
				}
				// Batches flow in chains of five, each into the next.
				if next := i + 1; i%5 != 4 && next < planBatchRows {
					fVals = append(fVals, "(?, ?)")
					fArgs = append(fArgs, planBatchID(next), planBatchID(i))
				}
				qID := fmt.Sprintf("qu_planbat_%06d", i)
				qVals = append(qVals, "(?, '24', 'un_planbat', ?, ?)")
				qArgs = append(qArgs, qID, createdAt, createdAt)
				bVals = append(bVals, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
				bArgs = append(bArgs, planBatchID(i), planBatchAccount, planBatchItemID(planBatchItem(i)), qID, station,
					run, scanned, closed, createdAt, createdAt)
			}
			exec(`INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES `+strings.Join(qVals, ","), qArgs...)
			exec(`INSERT INTO batch (id, account_id, item_id, quantity_id, scanning_station_id, production_run_id, scanned_at,
			      closed_at, created_at, updated_at) VALUES `+strings.Join(bVals, ","), bArgs...)
			if len(mVals) > 0 {
				exec(`INSERT INTO _batches_machines (A, B) VALUES `+strings.Join(mVals, ","), mArgs...)
			}
			if len(fVals) > 0 {
				exec(`INSERT INTO _batch_flow (A, B) VALUES `+strings.Join(fVals, ","), fArgs...)
			}
			if len(iVals) > 0 {
				exec(`INSERT INTO inventory_issue (id, account_id, item_id, status_code, quantity_id, batch_id, issued_at,
				      created_at, updated_at) VALUES `+strings.Join(iVals, ","), iArgs...)
			}
		}
		exec("ANALYZE TABLE batch, production_run, _batches_machines, _batch_flow, item, inventory_issue")
	})
}
