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

// The email log corpus is the production tenant's outbox: two years of mail, two in five of them sales
// order confirmations, most of the rest invoices and statements, one recipient each, sent by a handful
// of users and a few by no one (the system).
const (
	planEmailAccount = "ac_planeml"
	planEmailRows    = 36_000
	planEmailSenders = 11
	planEmailSpan    = 2 * 365 * 24 * time.Hour

	// planEmailCorpusVersion is bumped whenever the corpus's shape changes, so a stale one is rebuilt.
	planEmailCorpusVersion = "Plan Test Email v1"

	// planEmailRareSubject is in one subject, planEmailRareRecipient in one recipient's address, and
	// planEmailDenseSubject in two in five subjects.
	planEmailRareSubject   = "zqrare"
	planEmailRareRecipient = "zqrcpt"
	planEmailDenseSubject  = "Sales Order"
)

var planEmailOrigin = time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC)

func planEmailCreatedAt(i int) time.Time {
	return planEmailOrigin.Add(time.Duration(i) * (planEmailSpan / planEmailRows))
}

func planEmailID(i int) string { return fmt.Sprintf("emlg_%012d", i) }

func planEmailSubject(i int) string {
	switch {
	case i == 12_345:
		return "Statement " + planEmailRareSubject
	case i%5 < 2:
		return fmt.Sprintf("%s SO-%06d Confirmation", planEmailDenseSubject, i)
	case i%5 == 2:
		return fmt.Sprintf("Pay for your invoice INV-%06d", i)
	}
	return fmt.Sprintf("Statement of account %06d", i)
}

var planEmailCorpusOnce sync.Once

func ensureEmailLogCorpus(t *testing.T) {
	t.Helper()
	db := planDB(t)
	planEmailCorpusOnce.Do(func() {
		var have int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM email_log WHERE account_id = ?", planEmailAccount).Scan(&have))
		var version string
		_ = db.QueryRow("SELECT name FROM account WHERE id = ?", planEmailAccount).Scan(&version)
		if have >= planEmailRows && version == planEmailCorpusVersion {
			return
		}
		t.Logf("seeding the email log plan corpus (%d emails); it is kept for later runs", planEmailRows)

		exec := func(query string, args ...any) {
			_, err := db.Exec(query, args...)
			require.NoError(t, err)
		}
		exec("DELETE er FROM email_recipient er JOIN email_log el ON el.id = er.email_log_id WHERE el.account_id = ?", planEmailAccount)
		exec("DELETE FROM email_log WHERE account_id = ?", planEmailAccount)
		exec(`INSERT INTO account (id, name, account_type_code, onboarding_status_code) VALUES (?, ?, 'company', 'active')
		      ON DUPLICATE KEY UPDATE name = VALUES(name)`, planEmailAccount, planEmailCorpusVersion)
		for u := range planEmailSenders {
			exec("INSERT IGNORE INTO `user` (id, name) VALUES (?, ?)", fmt.Sprintf("us_planeml_%012d", u), fmt.Sprintf("Plan Sender %02d", u))
		}

		const batch = 1_000
		for start := 0; start < planEmailRows; start += batch {
			var lVals, rVals []string
			var lArgs, rArgs []any
			for i := start; i < start+batch; i++ {
				created := planEmailCreatedAt(i)
				var sender any
				if i%16 != 0 {
					sender = fmt.Sprintf("us_planeml_%012d", i%planEmailSenders)
				}
				lVals = append(lVals, "(?, 1, ?, ?, ?, ?, ?, ?, ?)")
				lArgs = append(lArgs, planEmailID(i), planEmailAccount, sender, planEmailSubject(i),
					fmt.Sprintf("Plan Customer %04d - Document %06d.pdf", i%900, i), fmt.Sprintf("0100018f%08x-plan-ses", i), created, created)
				recipient := fmt.Sprintf("buyer%04d@customer%04d.example.com", i%900, i%900)
				if i == 23_456 {
					recipient = planEmailRareRecipient + "@example.com"
				}
				rVals = append(rVals, "(?, ?, ?, ?, ?)")
				rArgs = append(rArgs, fmt.Sprintf("emrc_planeml_%012d", i), recipient, planEmailID(i), created, created)
			}
			exec(`INSERT INTO email_log (id, has_sent, account_id, sent_by_id, subject, filename, ses_message_id, created_at, updated_at)
			      VALUES `+strings.Join(lVals, ","), lArgs...)
			exec(`INSERT INTO email_recipient (id, email, email_log_id, created_at, updated_at) VALUES `+strings.Join(rVals, ","), rArgs...)
		}
		exec("ANALYZE TABLE email_log, email_recipient")
	})
}

func emailLogPlanCases() []planCase[domain.ListEmailLogsParams] {
	str := func(s string) *string { return &s }
	mid := planEmailCreatedAt(planEmailRows / 2)
	cursor := func(dir pagination.Direction) func(*domain.ListEmailLogsParams) {
		return func(p *domain.ListEmailLogsParams) { p.Cursor = planCursorAt(mid, "emlg_~", dir) }
	}
	type P = domain.ListEmailLogsParams
	return planCases(
		P{AccountID: planEmailAccount, Limit: 25},
		[]planDim[P]{
			{"search", []planValue[P]{
				{"subject-one", func(p *P) { p.Query = str(planEmailRareSubject) }},
				{"recipient-one", func(p *P) { p.Query = str(planEmailRareRecipient) }},
				{"subject-many", func(p *P) { p.Query = str(planEmailDenseSubject) }},
			}},
		},
		[]planValue[P]{
			{"first", func(*P) {}},
			{"deep-next", cursor(pagination.DirectionForward)},
			{"deep-prev", cursor(pagination.DirectionBackward)},
		},
	)
}

// emailLogSearchFloor is, for a request that searches, how many of the account's emails there are, or
// 0 without a search: a substring of a subject or an address is no B-tree's, so a plan tests every
// email until it has the page, which for a rare term is all of them.
func emailLogSearchFloor(t *testing.T, db *sql.DB, p domain.ListEmailLogsParams) float64 {
	t.Helper()
	if p.Query == nil {
		return 0
	}
	var n float64
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM email_log WHERE account_id = ?", p.AccountID).Scan(&n))
	return n
}

// TestEmailLogList_ReadsAboutAPage holds every request ListEmailLogs accepts to reading about a page
// of emails (listPlanSuite), and each page's recipients to lookups (lookupPlanSuite's bar).
func TestEmailLogList_ReadsAboutAPage(t *testing.T) {
	ensureEmailLogCorpus(t)
	listPlanSuite[domain.ListEmailLogsParams]{
		table: "email_log", scopeColumn: "account_id",
		from: "FROM email_log el", alias: "el",
		cases: emailLogPlanCases(),
		limit: func(p domain.ListEmailLogsParams) int32 { return p.Limit },
		list: func(ctx context.Context, q *sqlc.Queries, p domain.ListEmailLogsParams) error {
			if _, apiErr := NewEmailLogRepo(q).List(ctx, p); apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor:         emailLogSearchFloor,
		relatedTables: []string{},
		related: func(t *testing.T, stmts []explainedStatement, _ domain.ListEmailLogsParams) {
			for _, stmt := range stmts {
				if !strings.Contains(stmt.query, "FROM email_recipient") {
					continue
				}
				returned := planReturned(stmt.plan)
				if got := tableAccess(stmt.plan, "email_recipient"); got.rows > 2*returned+lookupSlack {
					t.Errorf("read %.0f recipients via %v to return %.0f\n%s", got.rows, got.indexes, returned, stmt.plan)
				}
			}
		},
	}.run(t)
}
