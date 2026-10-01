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

// The EDI run corpus is the production tenant's import history: a run every few hours for three
// years, one in eleven failed with its failures recorded, UUID ids.
const (
	planEDIAccount = "ac_planedi"
	planEDIRows    = 6_000
	planEDISpan    = 3 * 365 * 24 * time.Hour

	// planEDICorpusVersion is bumped whenever the corpus's shape changes, so a stale one is rebuilt.
	planEDICorpusVersion = "Plan Test EDI v1"
)

var planEDIOrigin = time.Date(2023, 9, 1, 0, 0, 0, 0, time.UTC)

func planEDICompletedAt(i int) time.Time {
	return planEDIOrigin.Add(time.Duration(i) * (planEDISpan / planEDIRows))
}

func planEDIID(i int) string { return fmt.Sprintf("%08x-%04x-4ed1-8000-%012x", i*2654435761%(1<<32), i%65536, i) }

var planEDICorpusOnce sync.Once

func ensureEDIRunCorpus(t *testing.T) {
	t.Helper()
	db := planDB(t)
	planEDICorpusOnce.Do(func() {
		var have int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM edi_run WHERE account_id = ?", planEDIAccount).Scan(&have))
		var version string
		_ = db.QueryRow("SELECT name FROM account WHERE id = ?", planEDIAccount).Scan(&version)
		if have >= planEDIRows && version == planEDICorpusVersion {
			return
		}
		t.Logf("seeding the EDI run plan corpus (%d runs); it is kept for later runs", planEDIRows)

		exec := func(query string, args ...any) {
			_, err := db.Exec(query, args...)
			require.NoError(t, err)
		}
		exec("DELETE FROM edi_run WHERE account_id = ?", planEDIAccount)
		exec(`INSERT INTO account (id, name, account_type_code, onboarding_status_code) VALUES (?, ?, 'company', 'active')
		      ON DUPLICATE KEY UPDATE name = VALUES(name)`, planEDIAccount, planEDICorpusVersion)
		failure := `[{"code":"partner_rejected","message":"The trading partner rejected the 850 for PO 4500123456: unknown ship-to location","document":"850"}]`
		var vals []string
		var args []any
		for i := range planEDIRows {
			completed := planEDICompletedAt(i)
			succeeded := i%11 != 5
			failures := "[]"
			if !succeeded {
				failures = failure + strings.Repeat(" ", 200)
			}
			vals = append(vals, "(?, ?, ?, ?, ?, ?, ?)")
			args = append(args, planEDIID(i), completed, planEDIAccount, completed, succeeded, failures, completed)
			if len(vals) == 1_000 {
				exec(`INSERT INTO edi_run (id, completed_at, account_id, created_at, has_succeeded, failures, updated_at) VALUES `+
					strings.Join(vals, ","), args...)
				vals, args = nil, nil
			}
		}
		exec("ANALYZE TABLE edi_run")
	})
}

func ediRunPlanCases() []planCase[domain.ListEDIRunsParams] {
	b := func(v bool) *bool { return &v }
	str := func(s string) *string { return &s }
	mid := planEDICompletedAt(planEDIRows / 2)
	cursor := func(dir pagination.Direction) func(*domain.ListEDIRunsParams) {
		return func(p *domain.ListEDIRunsParams) { p.Cursor = planCursorAt(mid, "~", dir) }
	}
	type P = domain.ListEDIRunsParams
	return planCases(
		P{AccountID: planEDIAccount, Limit: 25},
		[]planDim[P]{
			{"status", []planValue[P]{
				{"succeeded", func(p *P) { p.HasSucceeded = b(true) }},
				{"failed", func(p *P) { p.HasSucceeded = b(false) }},
			}},
			{"search", []planValue[P]{
				{"id-one", func(p *P) { p.Query = str(planEDIID(4_321)[24:]) }},
				{"id-every", func(p *P) { p.Query = str("-4ed1-") }},
			}},
		},
		[]planValue[P]{
			{"first", func(*P) {}},
			{"deep-next", cursor(pagination.DirectionForward)},
			{"deep-prev", cursor(pagination.DirectionBackward)},
		},
	)
}

// ediRunSearchFloor is, for a request that searches ids, how many runs its status filter matches, or 0
// without a search: a substring of an id is no B-tree's, so a plan tests every candidate the status
// key yields until it has the page, which for a rare term is all of them.
func ediRunSearchFloor(t *testing.T, db *sql.DB, p domain.ListEDIRunsParams) float64 {
	t.Helper()
	if p.Query == nil {
		return 0
	}
	where, args := "account_id = ?", []any{p.AccountID}
	if p.HasSucceeded != nil {
		where, args = where+" AND has_succeeded = ?", append(args, *p.HasSucceeded)
	}
	var n float64
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM edi_run WHERE "+where, args...).Scan(&n))
	return n
}

// TestEDIRunList_ReadsAboutAPage holds every filter combination ListEDIRuns accepts to reading about a
// page of runs (listPlanSuite).
func TestEDIRunList_ReadsAboutAPage(t *testing.T) {
	ensureEDIRunCorpus(t)
	listPlanSuite[domain.ListEDIRunsParams]{
		table: "edi_run", scopeColumn: "account_id",
		from: "FROM edi_run er", alias: "er",
		cases: ediRunPlanCases(),
		limit: func(p domain.ListEDIRunsParams) int32 { return p.Limit },
		list: func(ctx context.Context, q *sqlc.Queries, p domain.ListEDIRunsParams) error {
			if _, apiErr := NewEDIRepo(q).ListEDIRuns(ctx, p); apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: ediRunSearchFloor,
	}.run(t)
}
