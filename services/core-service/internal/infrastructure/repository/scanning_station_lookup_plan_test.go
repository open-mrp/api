//go:build plans

package repository

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// The station corpus is the largest production tenant's floor: six stations holding over five thousand
// production steps between them, most on two stations, one station with a handful, and a few steps on
// none. A batch get of stations returns each with its steps, so a station's steps are what it reads.
const (
	planStationAccount = "ac_planstn"
	planStationSteps   = 5_300

	// planStationCorpusVersion is bumped whenever the corpus's shape changes, so a stale one is rebuilt.
	planStationCorpusVersion = "Plan Test Stations v1"
)

// planStationShares is how many steps each station holds; what is left over is on no station.
var planStationShares = []int{1_716, 1_500, 1_000, 600, 470, 7}

func planStationID(s int) string { return fmt.Sprintf("0000%04d-plan-4stn-8000-%012d", s, s) }

var planStationCorpusOnce sync.Once

func ensureScanningStationCorpus(t *testing.T) {
	t.Helper()
	db := planDB(t)
	planStationCorpusOnce.Do(func() {
		var have int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM production_step WHERE account_id = ?", planStationAccount).Scan(&have))
		var version string
		_ = db.QueryRow("SELECT name FROM account WHERE id = ?", planStationAccount).Scan(&version)
		if have >= planStationSteps && version == planStationCorpusVersion {
			return
		}
		t.Logf("seeding the scanning station plan corpus (%d steps); it is kept for later runs", planStationSteps)

		exec := func(query string, args ...any) {
			_, err := db.Exec(query, args...)
			require.NoError(t, err)
		}
		exec("DELETE FROM production_step WHERE account_id = ?", planStationAccount)
		exec("DELETE FROM scanning_station WHERE account_id = ?", planStationAccount)
		exec(`INSERT INTO account (id, name, account_type_code, onboarding_status_code) VALUES (?, ?, 'company', 'active')
		      ON DUPLICATE KEY UPDATE name = VALUES(name)`, planStationAccount, planStationCorpusVersion)
		created := time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC)
		for s := range planStationShares {
			exec(`INSERT INTO scanning_station (id, name, department_id, account_id, scanning_station_type_code, created_at, updated_at)
			      VALUES (?, ?, 'dept_planstn', ?, 'production', ?, ?)`, planStationID(s), fmt.Sprintf("Plan Station %d", s), planStationAccount, created, created)
		}

		var vals []string
		var args []any
		step, station, onStation := 0, 0, 0
		for step < planStationSteps {
			var stationID any
			if station < len(planStationShares) {
				stationID = planStationID(station)
				if onStation++; onStation == planStationShares[station] {
					station, onStation = station+1, 0
				}
			}
			vals = append(vals, "(?, ?, ?, ?, ?, ?, ?, 'dept_planstn', ?, ?)")
			args = append(args, fmt.Sprintf("ps_planstn_%06d", step), fmt.Sprintf("Plan Step %05d", step), planStationAccount,
				fmt.Sprintf("rt_planstn_lr_%06d", step), fmt.Sprintf("qu_planstn_lt_%06d", step), fmt.Sprintf("rt_planstn_or_%06d", step),
				stationID, created, created)
			step++
			if len(vals) == 1_000 || step == planStationSteps {
				exec(`INSERT INTO production_step (id, name, account_id, labor_rate_id, labor_time_id, overhead_rate_id,
				      scanning_station_id, department_id, created_at, updated_at) VALUES `+strings.Join(vals, ","), args...)
				vals, args = nil, nil
			}
		}
		exec("ANALYZE TABLE production_step, scanning_station")
	})
}

// TestScanningStationBatchGet_ReadsWhatItReturns holds BatchGetScanningStationsByIDs to reading about
// the stations and steps it returns (lookupPlanSuite).
func TestScanningStationBatchGet_ReadsWhatItReturns(t *testing.T) {
	ensureScanningStationCorpus(t)
	get := func(ids ...string) func(ctx context.Context, q *sqlc.Queries) error {
		return func(ctx context.Context, q *sqlc.Queries) error {
			if _, apiErr := NewScanningStationRepo(q).GetByIDs(ctx, planStationAccount, ids); apiErr != nil {
				return apiErr
			}
			return nil
		}
	}
	var every []string
	for s := range planStationShares {
		every = append(every, planStationID(s))
	}
	lookupPlanSuite{
		tables: []string{"scanning_station", "production_step"},
		cases: []lookupPlanCase{
			{"busiest", get(planStationID(0))},
			{"rare", get(planStationID(len(planStationShares) - 1))},
			{"every", get(every...)},
			{"unknown", get("sst_planstn_missing")},
		},
	}.run(t)
}
