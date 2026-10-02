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
	"github.com/open-mrp/api/shared/pagination"
)

// The downtime corpus is a plant logging its stoppages for two years. Production holds only a few dozen
// events today, so the corpus is sized for the volume a busy plant reaches: one machine and one reason
// account for most stoppages, one machine, department and reason are rare, a handful are still open,
// and a third carry a note. Row widths follow production: UUID machine and department IDs.
const (
	planDTAccount     = "ac_plandt"
	planDTRows        = 20_000
	planDTMachines    = 40
	planDTDepartments = 10
	planDTSpan        = 2 * 365 * 24 * time.Hour

	// planDTCorpusVersion is bumped whenever the corpus's shape changes, so a stale one is rebuilt.
	planDTCorpusVersion = "Plan Test Downtime v1"

	// planDTRareNote appears in one note; planDTDenseNote in every note.
	planDTRareNote  = "zqrare"
	planDTDenseNote = "jam"
)

var planDTOrigin = time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC)

// planDTReasons are the reason codes, most common first; the last is rare.
var planDTReasons = []string{"breakdown", "changeover", "material_shortage", "no_operator", "maintenance", "quality_hold", "tooling"}

func planDTStartedAt(i int) time.Time {
	return planDTOrigin.Add(time.Duration(i) * (planDTSpan / planDTRows))
}

func planDTID(i int) string           { return fmt.Sprintf("mdte_%012d", i) }
func planDTMachineID(m int) string    { return fmt.Sprintf("0000%04d-plan-4dtm-8000-%012d", m, m) }
func planDTDepartmentID(d int) string { return fmt.Sprintf("0000%04d-plan-4dtd-8000-%012d", d, d) }

var planDTRareRows = map[int]bool{200: true, 10_000: true, 19_990: true}

// planDTMachine gives one machine a third of the stoppages; the last machine is rare. A machine sits
// in one department, so the last department holds only the rare machine.
func planDTMachine(i int) int {
	switch {
	case planDTRareRows[i]:
		return planDTMachines - 1
	case i%3 == 0:
		return 0
	}
	return 1 + (i*31)%(planDTMachines-2)
}

func planDTDepartment(m int) int {
	if m == planDTMachines-1 {
		return planDTDepartments - 1
	}
	return m % (planDTDepartments - 1)
}

func planDTReason(i int) string {
	switch {
	case planDTRareRows[i]:
		return planDTReasons[len(planDTReasons)-1]
	case i%5 < 2:
		return planDTReasons[0]
	}
	return planDTReasons[1+i%(len(planDTReasons)-2)]
}

// planDTOpen reports whether stoppage i is still open: the newest few and a forgotten handful.
func planDTOpen(i int) bool { return i >= planDTRows-5 || i%4000 == 17 }

var planDTCorpusOnce sync.Once

func ensureDowntimeCorpus(t *testing.T) {
	t.Helper()
	db := planDB(t)
	planDTCorpusOnce.Do(func() {
		var have int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM machine_downtime_event WHERE account_id = ?", planDTAccount).Scan(&have))
		var version string
		_ = db.QueryRow("SELECT name FROM account WHERE id = ?", planDTAccount).Scan(&version)
		if have >= planDTRows && version == planDTCorpusVersion {
			return
		}
		t.Logf("seeding the downtime plan corpus (%d events); it is kept for later runs", planDTRows)

		exec := func(query string, args ...any) {
			_, err := db.Exec(query, args...)
			require.NoError(t, err)
		}
		exec("DELETE FROM machine_downtime_event WHERE account_id = ?", planDTAccount)
		exec(`INSERT INTO account (id, name, account_type_code, onboarding_status_code) VALUES (?, ?, 'company', 'active')
		      ON DUPLICATE KEY UPDATE name = VALUES(name)`, planDTAccount, planDTCorpusVersion)

		const batch = 1_000
		for start := 0; start < planDTRows; start += batch {
			var vals []string
			var args []any
			for i := start; i < start+batch; i++ {
				started := planDTStartedAt(i)
				var ended, duration, note any
				if !planDTOpen(i) {
					ended, duration = started.Add(20*time.Minute), 1200
				}
				switch {
				case i == 7_777:
					note = "spindle " + planDTRareNote + " fault"
				case i%3 == 1:
					note = "operator reported a " + planDTDenseNote + " at the infeed"
				}
				m := planDTMachine(i)
				vals = append(vals, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'us_plandt_operator', ?, ?)")
				args = append(args, planDTID(i), planDTAccount, planDTMachineID(m), planDTDepartmentID(planDTDepartment(m)),
					planDTReason(i), started, ended, duration, started.Format("2006-01-02"), note, started, started)
			}
			exec(`INSERT INTO machine_downtime_event (id, account_id, machine_id, department_id, reason_code, started_at, ended_at,
			      duration_seconds, shift_date, note, reported_by_id, created_at, updated_at) VALUES `+strings.Join(vals, ","), args...)
		}
		exec("ANALYZE TABLE machine_downtime_event")
	})
}

func downtimePlanCases() []planCase[domain.ListMachineDowntimeEventsParams] {
	str := func(s string) *string { return &s }
	at := func(d time.Time) *time.Time { return &d }
	old := planDTStartedAt(planDTRows / 4)
	mid := planDTStartedAt(planDTRows / 2)
	cursor := func(dir pagination.Direction) func(*domain.ListMachineDowntimeEventsParams) {
		return func(p *domain.ListMachineDowntimeEventsParams) { p.Cursor = planCursorAt(mid, "mdte_~", dir) }
	}
	type P = domain.ListMachineDowntimeEventsParams
	return planCases(
		P{AccountID: planDTAccount, Limit: 25},
		[]planDim[P]{
			{"machine", []planValue[P]{
				{"machine-busy", func(p *P) { p.MachineIDs = []string{planDTMachineID(0)} }},
				{"machine-rare", func(p *P) { p.MachineIDs = []string{planDTMachineID(planDTMachines - 1)} }},
				{"machine-busy+rare", func(p *P) { p.MachineIDs = []string{planDTMachineID(0), planDTMachineID(planDTMachines - 1)} }},
			}},
			{"department", []planValue[P]{
				{"dept-busy", func(p *P) { p.DepartmentIDs = []string{planDTDepartmentID(0)} }},
				{"dept-rare", func(p *P) { p.DepartmentIDs = []string{planDTDepartmentID(planDTDepartments - 1)} }},
			}},
			{"reason", []planValue[P]{
				{"breakdown", func(p *P) { p.ReasonCodes = []string{planDTReasons[0]} }},
				{"tooling", func(p *P) { p.ReasonCodes = []string{planDTReasons[len(planDTReasons)-1]} }},
				{"breakdown+changeover", func(p *P) { p.ReasonCodes = planDTReasons[:2] }},
			}},
			{"open", []planValue[P]{
				{"open", func(p *P) { p.OpenOnly = true }},
			}},
			{"search", []planValue[P]{
				{"note-one", func(p *P) { p.Query = str(planDTRareNote) }},
				{"note-every", func(p *P) { p.Query = str(planDTDenseNote) }},
			}},
			{"dates", []planValue[P]{
				{"old30d", func(p *P) { p.StartDate, p.EndDate = at(old), at(old.Add(30*24*time.Hour)) }},
			}},
		},
		[]planValue[P]{
			{"first", func(*P) {}},
			{"deep-next", cursor(pagination.DirectionForward)},
			{"deep-prev", cursor(pagination.DirectionBackward)},
		},
	)
}

// downtimeNoteFloor is, for a request that searches notes, how many events its other filters match,
// or 0 without a search. A substring of free text is no B-tree's, so no key finds its matches: a plan
// tests every candidate the other filters' key yields until it has the page, which for a rare term
// is all of them.
func downtimeNoteFloor(t *testing.T, db *sql.DB, p domain.ListMachineDowntimeEventsParams) float64 {
	t.Helper()
	if p.Query == nil {
		return 0
	}
	where, args := []string{"account_id = ?"}, []any{p.AccountID}
	in := func(column string, values []string) {
		if len(values) > 0 {
			where, args = append(where, column+" IN ("+placeholders(len(values))+")"), append(args, stringArgs(values)...)
		}
	}
	in("machine_id", p.MachineIDs)
	in("department_id", p.DepartmentIDs)
	in("reason_code", p.ReasonCodes)
	if p.OpenOnly {
		where = append(where, "ended_at IS NULL")
	}
	if p.StartDate != nil {
		where, args = append(where, "started_at >= ?"), append(args, *p.StartDate)
	}
	if p.EndDate != nil {
		where, args = append(where, "started_at <= ?"), append(args, *p.EndDate)
	}
	var n float64
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM machine_downtime_event WHERE "+strings.Join(where, " AND "), args...).Scan(&n))
	return n
}

// TestDowntimeList_ReadsAboutAPage holds every filter combination ListDowntimeEvents accepts to
// reading about a page of events (listPlanSuite).
func TestDowntimeList_ReadsAboutAPage(t *testing.T) {
	ensureDowntimeCorpus(t)
	listPlanSuite[domain.ListMachineDowntimeEventsParams]{
		table: "machine_downtime_event", scopeColumn: "account_id",
		from: "FROM machine_downtime_event e", alias: "e",
		cases: downtimePlanCases(),
		limit: func(p domain.ListMachineDowntimeEventsParams) int32 { return p.Limit },
		list: func(ctx context.Context, q *sqlc.Queries, p domain.ListMachineDowntimeEventsParams) error {
			if _, apiErr := NewMachineDowntimeRepo(q).List(ctx, p); apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: downtimeNoteFloor,
	}.run(t)
}
