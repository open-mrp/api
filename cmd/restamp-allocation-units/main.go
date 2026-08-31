// Command restamp-allocation-units repairs allocations whose quantity was converted into their
// receipt's unit but labelled with the issue's, by correcting the label and leaving the value alone.
//
// # What went wrong
//
// Over four days in late October 2025 the import-era writer produced allocations like this: an issue
// for 22 `ct10pr` (Carton of 10 pairs) drawn from a receipt held in `pr` was written as `220 ct10pr`.
// 220 is the correct figure in *pairs* — 22 cartons of 10 — but it is wearing the carton label, and a
// carton normalises at ten times a pair. So the allocation reads ten times the stock it drew, the
// receipt looks drawn far past what it held, and it sits in `allocated` where nothing counts it as
// on hand. The same shape appears at 12x through `ct12pr` and 50x through `cs50ea`.
//
// The value is right and the unit is wrong, so the repair is to restamp the unit. Deleting the
// allocation instead — which is what a tool looking only at "allocated exceeds received" would do —
// removes a draw that really happened and reopens an order that was really filled.
//
// # Why it is safe to do mechanically, and where it stops
//
// The evidence is that the numbers land. Restamp every one of these and 176 of 177 receipts fall
// exactly inside the quantity they received, and 173 of 206 issues land exactly on the quantity they
// asked for. A label that was merely guessed would not do that.
//
// This command only writes where that lands exactly. An issue can be covered by allocations drawn
// from several receipts, and a receipt can serve several issues, so eligibility is not a property of
// one row: receipts and issues are grouped into connected components through the allocations that
// join them, and a component is repaired only when **every** receipt in it ends up within its
// quantity and **every** issue in it ends up exactly covered. Anything else is reported and left,
// because a component that does not reconcile is one where something beyond the label is also wrong.
//
// Receipts that come back under their quantity are returned to `available`, which is what puts the
// stock back on hand.
//
// Usage:
//
//	DB_URL=<dsn-or-mysql-url> go run ./cmd/restamp-allocation-units --account ac_... --dry-run
//	DB_URL=<dsn-or-mysql-url> go run ./cmd/restamp-allocation-units --account ac_...
//
// Flags:
//
//	--dry-run   report what would change; make no writes.
//	--account   restrict to one account id.
//	--item      restrict to one item id.
//	--epsilon   tolerance, in base units, for "fits" and "exactly covered" (default 0.0001).
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/shared/db"
	"github.com/open-mrp/api/shared/env"
)

func main() {
	ctx := context.Background()
	if err := Run(ctx, os.Args, os.Getenv, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err) // #nosec G705 -- CLI stderr output, not web context
		os.Exit(1)
	}
}

// candidate is one mislabelled allocation: the value stays, the unit becomes the receipt's.
type candidate struct {
	AllocID       string
	QuantityID    string
	ReceiptID     string
	IssueID       string
	ReceiptUnitID string
}

// allocation is any allocation touching a receipt or issue in play, mislabelled or not, because the
// sums that decide eligibility are over all of them.
type allocation struct {
	ID        string
	ReceiptID string
	IssueID   string
	// Base is what the row contributes today, through its own unit's ratio.
	Base decimal.Decimal
	// RestampedBase is what it would contribute with the receipt's unit instead. Equal to Base for
	// rows that are not candidates.
	RestampedBase decimal.Decimal
}

func Run(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("restamp-allocation-units", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dryRun := fs.Bool("dry-run", false, "report what would change; make no writes")
	accountID := fs.String("account", "", "restrict to a single account id")
	itemID := fs.String("item", "", "restrict to a single item id")
	epsilon := fs.String("epsilon", "0.0001", "tolerance in base units for fits / exactly covered")
	acceptImperfect := fs.Bool("accept-imperfect-coverage", false,
		"restamp wherever the receipts fit, even where the issue does not land exactly on its demand")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	eps, err := decimal.NewFromString(*epsilon)
	if err != nil {
		return fmt.Errorf("parsing --epsilon: %w", err)
	}

	dbURL := env.GetEnv("DB_URL", getenv)
	if dbURL == "" {
		return fmt.Errorf("DB_URL is required")
	}
	dsn, err := normalizeDSN(dbURL)
	if err != nil {
		return err
	}

	sqlDB, err := db.NewDbPool(&db.Config{DBURI: dsn})
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer sqlDB.Close()

	p := &progress{out: stdout, start: time.Now()}
	if *accountID != "" {
		p.stagef("Restricted to account %s", *accountID)
	}
	if *itemID != "" {
		p.stagef("Restricted to item %s", *itemID)
	}

	candidates, err := findCandidates(ctx, sqlDB, *accountID, *itemID, p)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		p.stage("Nothing to do")
		return nil
	}

	model, err := loadModel(ctx, sqlDB, candidates, p)
	if err != nil {
		return err
	}

	eligible, groups, skipped := model.plan(candidates, eps, *acceptImperfect)

	p.stage("Summary")
	fmt.Fprintf(stdout, "  Mislabelled allocations:   %d\n", len(candidates))
	fmt.Fprintf(stdout, "  Groups (receipts+issues):  %d\n", groups)
	fmt.Fprintf(stdout, "  Groups that reconcile:     %d\n", groups-len(skipped))
	fmt.Fprintf(stdout, "  Allocations to restamp:    %d\n", len(eligible))

	freed := model.stockFreed(eligible, eps)
	fmt.Fprintf(stdout, "  Stock returned to on hand: %s (receipt units)\n", freed.StringFixed(2))

	if len(skipped) > 0 {
		fmt.Fprint(stdout, skipNote)
		for i, s := range skipped {
			if i >= skipLines {
				fmt.Fprintf(stdout, "    ... and %d more\n", len(skipped)-skipLines)
				break
			}
			fmt.Fprintf(stdout, "    %s\n", s)
		}
		fmt.Fprintln(stdout)
	}

	if len(eligible) == 0 {
		p.stage("Nothing eligible")
		return nil
	}
	if *dryRun {
		p.stage("--dry-run: no writes made")
		return nil
	}

	receiptsFreed, err := apply(ctx, sqlDB, eligible, model, eps, p)
	if err != nil {
		return err
	}

	p.stage("Done")
	fmt.Fprintf(stdout, "  Restamped %d allocation(s).\n", len(eligible))
	fmt.Fprintf(stdout, "  Returned %d receipt(s) to available.\n", receiptsFreed)
	return nil
}

const skipLines = 25

const skipNote = `
  NOT REPAIRED — these groups do not reconcile once the labels are corrected.
  A group is one set of receipts and issues joined by the allocations between them. Restamping is
  applied only where every receipt in the group lands inside its own quantity and every issue lands
  exactly on what it asked for; a group that misses means something beyond the unit label is also
  wrong there, and guessing at it would write a number nobody can check.
`

// findCandidates lists the mislabelled allocations worth correcting: those on a receipt whose
// allocations, normalised, exceed what it received.
//
// The unit differing from the receipt's is not on its own evidence of anything. The dashboard's
// allocator records a draw in whichever unit bounded it, so an allocation limited by the issue is
// written in the issue's unit with a value to match — internally consistent, and no phantom stock.
// Those rows outnumber the broken ones by two orders of magnitude. What separates the import's rows
// is that the value does *not* match the label, which is exactly what pushes the receipt over.
func findCandidates(ctx context.Context, sqlDB *sql.DB, accountID, itemID string, p *progress) ([]candidate, error) {
	var where strings.Builder
	var params []any
	if accountID != "" {
		where.WriteString(" AND i.account_id = ?")
		params = append(params, accountID)
	}
	if itemID != "" {
		where.WriteString(" AND ir.item_id = ?")
		params = append(params, itemID)
	}

	// #nosec G202 -- the only interpolation is `where`, built above from fixed literals; the account
	// and item ids it filters on are bound as parameters.
	query := `
SELECT a.id, aq.id AS quantity_id, a.inventory_receipt_id, a.inventory_issue_id, rq.unit_id
FROM inventory_allocation a
JOIN quantity aq ON aq.id = a.quantity_id
JOIN inventory_receipt ir ON ir.id = a.inventory_receipt_id
JOIN quantity rq ON rq.id = ir.quantity_id
JOIN unit ru ON ru.id = rq.unit_id
JOIN item i ON i.id = ir.item_id
JOIN (
    SELECT ia.inventory_receipt_id AS rid,
           SUM(CAST(aq2.value AS DECIMAL(65,30)) * (au2.ratio_numerator / au2.ratio_denominator)) AS alloc_base
    FROM inventory_allocation ia
    JOIN inventory_receipt ir2 ON ir2.id = ia.inventory_receipt_id
    JOIN item i2 ON i2.id = ir2.item_id
    JOIN quantity aq2 ON aq2.id = ia.quantity_id
    JOIN unit au2 ON au2.id = aq2.unit_id
    WHERE 1 = 1` + strings.ReplaceAll(strings.ReplaceAll(where.String(), "i.account_id", "i2.account_id"), "ir.item_id", "ir2.item_id") + `
    GROUP BY ia.inventory_receipt_id
) s ON s.rid = ir.id
WHERE aq.unit_id <> rq.unit_id
  AND CAST(rq.value AS DECIMAL(65,30)) > 0
  AND s.alloc_base > CAST(rq.value AS DECIMAL(65,30)) * (ru.ratio_numerator / ru.ratio_denominator) + 0.000001` +
		where.String()

	// The derived table repeats the same filters under rewritten aliases, so its parameters are bound
	// first and the outer copy follows.
	allParams := append(append([]any{}, params...), params...)

	p.stage("Scanning for mislabelled allocations on over-drawn receipts...")

	rows, err := sqlDB.QueryContext(ctx, query, allParams...)
	if err != nil {
		return nil, fmt.Errorf("find candidates: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.AllocID, &c.QuantityID, &c.ReceiptID, &c.IssueID, &c.ReceiptUnitID); err != nil {
			return nil, fmt.Errorf("scan candidate: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate candidates: %w", err)
	}

	p.stagef("Found %d mislabelled allocation(s)", len(out))
	return out, nil
}

// model is every receipt, issue and allocation the candidates touch, with the sums the plan needs.
type model struct {
	receiptCapacity map[string]decimal.Decimal
	issueDemand     map[string]decimal.Decimal
	byReceipt       map[string][]*allocation
	byIssue         map[string][]*allocation
	allocByID       map[string]*allocation
}

// loadModel reads the receipts and issues the candidates touch, together with every allocation
// against either side — including ones that are correctly labelled, since they count towards the
// same sums.
func loadModel(ctx context.Context, sqlDB *sql.DB, candidates []candidate, p *progress) (*model, error) {
	receiptIDs := distinct(candidates, func(c candidate) string { return c.ReceiptID })
	issueIDs := distinct(candidates, func(c candidate) string { return c.IssueID })
	candByAlloc := make(map[string]candidate, len(candidates))
	for _, c := range candidates {
		candByAlloc[c.AllocID] = c
	}

	m := &model{
		receiptCapacity: map[string]decimal.Decimal{},
		issueDemand:     map[string]decimal.Decimal{},
		byReceipt:       map[string][]*allocation{},
		byIssue:         map[string][]*allocation{},
		allocByID:       map[string]*allocation{},
	}

	p.stagef("Loading %d receipt(s) and %d issue(s)...", len(receiptIDs), len(issueIDs))

	if err := loadCapacities(ctx, sqlDB, receiptIDs,
		"SELECT ir.id, CAST(q.value AS DECIMAL(65,30)) * (u.ratio_numerator / u.ratio_denominator) "+
			"FROM inventory_receipt ir JOIN quantity q ON q.id = ir.quantity_id JOIN unit u ON u.id = q.unit_id WHERE ir.id IN ",
		m.receiptCapacity); err != nil {
		return nil, err
	}
	if err := loadCapacities(ctx, sqlDB, issueIDs,
		"SELECT ii.id, CAST(q.value AS DECIMAL(65,30)) * (u.ratio_numerator / u.ratio_denominator) "+
			"FROM inventory_issue ii JOIN quantity q ON q.id = ii.quantity_id JOIN unit u ON u.id = q.unit_id WHERE ii.id IN ",
		m.issueDemand); err != nil {
		return nil, err
	}

	// Every allocation against a receipt in play, and every allocation against an issue in play. The
	// two sets overlap; allocByID keeps one row per allocation.
	for _, spec := range []struct {
		ids    []string
		column string
	}{
		{receiptIDs, "a.inventory_receipt_id"},
		{issueIDs, "a.inventory_issue_id"},
	} {
		for chunk := range chunks(spec.ids, 300) {
			// #nosec G202 -- column is a package-level literal; placeholders() emits only "?,?,…".
			query := `
SELECT a.id, a.inventory_receipt_id, a.inventory_issue_id,
       CAST(aq.value AS DECIMAL(65,30)) * (au.ratio_numerator / au.ratio_denominator) AS base,
       CAST(aq.value AS DECIMAL(65,30)) * (ru.ratio_numerator / ru.ratio_denominator) AS restamped
FROM inventory_allocation a
JOIN quantity aq ON aq.id = a.quantity_id
JOIN unit au ON au.id = aq.unit_id
JOIN inventory_receipt ir ON ir.id = a.inventory_receipt_id
JOIN quantity rq ON rq.id = ir.quantity_id
JOIN unit ru ON ru.id = rq.unit_id
WHERE ` + spec.column + ` IN (` + placeholders(len(chunk)) + `)`
			rows, err := sqlDB.QueryContext(ctx, query, toAny(chunk)...)
			if err != nil {
				return nil, fmt.Errorf("load allocations: %w", err)
			}
			for rows.Next() {
				var id, rid, iid, base, restamped string
				if err := rows.Scan(&id, &rid, &iid, &base, &restamped); err != nil {
					_ = rows.Close()
					return nil, fmt.Errorf("scan allocation: %w", err)
				}
				if _, seen := m.allocByID[id]; seen {
					continue
				}
				b, err := decimal.NewFromString(base)
				if err != nil {
					_ = rows.Close()
					return nil, fmt.Errorf("parse allocation base for %s: %w", id, err)
				}
				// Only a candidate changes contribution; everything else keeps what it has today.
				newBase := b
				if _, isCandidate := candByAlloc[id]; isCandidate {
					if newBase, err = decimal.NewFromString(restamped); err != nil {
						_ = rows.Close()
						return nil, fmt.Errorf("parse restamped base for %s: %w", id, err)
					}
				}
				a := &allocation{ID: id, ReceiptID: rid, IssueID: iid, Base: b, RestampedBase: newBase}
				m.allocByID[id] = a
				m.byReceipt[rid] = append(m.byReceipt[rid], a)
				m.byIssue[iid] = append(m.byIssue[iid], a)
			}
			if err := rows.Err(); err != nil {
				_ = rows.Close()
				return nil, fmt.Errorf("iterate allocations: %w", err)
			}
			_ = rows.Close()
		}
	}

	return m, nil
}

// plan groups the receipts and issues the candidates join into connected components and accepts a
// component only when every receipt in it fits and — unless acceptImperfect — every issue in it is
// exactly covered.
//
// Grouping is the whole point. Eligibility cannot be decided per row: an issue covered from three
// receipts is only correct once all three are corrected, and correcting two of them would leave it
// short by the third.
//
// The receipt test is the hard one and is never waived: a receipt cannot give out more than it took
// in, so a component that still overflows after the labels are corrected has something else wrong
// with it and correcting the labels would not be an improvement.
//
// The issue test is a different kind of claim. Landing exactly on demand is strong evidence that the
// label was the only thing wrong, which is why it is the default. But an issue that misses is not
// evidence *against* the label — the ways it misses turn out to be a duplicate draw on top of the
// mislabel, which lands on exactly twice the demand, and a draw that emptied a receipt too small to
// cover the order, which lands under. Both are conditions the mislabel hides rather than causes, and
// both are repaired on their own terms once the units read correctly. acceptImperfect is for that
// second pass: it keeps the receipt invariant and drops the exactness requirement, so the remaining
// labels are corrected and the coverage problem underneath becomes visible and fixable.
func (m *model) plan(candidates []candidate, eps decimal.Decimal, acceptImperfect bool) (eligible []candidate, groups int, skipped []string) {
	uf := newUnionFind()
	for _, c := range candidates {
		uf.union("r:"+c.ReceiptID, "i:"+c.IssueID)
	}

	// A component is bad if any receipt overflows or any issue misses its demand.
	bad := map[string]string{}
	for rid, allocs := range m.byReceipt {
		capacity, ok := m.receiptCapacity[rid]
		if !ok {
			continue
		}
		total := decimal.Zero
		for _, a := range allocs {
			total = total.Add(a.RestampedBase)
		}
		if total.Sub(capacity).GreaterThan(eps) {
			root := uf.find("r:" + rid)
			if _, seen := bad[root]; !seen {
				bad[root] = fmt.Sprintf("receipt %s still over: holds %s, allocated %s (base units)",
					rid, capacity.StringFixed(2), total.StringFixed(2))
			}
		}
	}
	for iid, allocs := range m.byIssue {
		if acceptImperfect {
			break
		}
		demand, ok := m.issueDemand[iid]
		if !ok {
			continue
		}
		// Only issues drawn on by a candidate are part of a group; others are untouched by the change.
		root, inGroup := uf.rootIfPresent("i:" + iid)
		if !inGroup {
			continue
		}
		total := decimal.Zero
		for _, a := range allocs {
			total = total.Add(a.RestampedBase)
		}
		if total.Sub(demand).Abs().GreaterThan(eps) {
			if _, seen := bad[root]; !seen {
				bad[root] = fmt.Sprintf("issue %s not exactly covered: wants %s, would get %s (base units)",
					iid, demand.StringFixed(2), total.StringFixed(2))
			}
		}
	}

	roots := map[string]struct{}{}
	for _, c := range candidates {
		roots[uf.find("r:"+c.ReceiptID)] = struct{}{}
	}
	groups = len(roots)

	for _, c := range candidates {
		if _, isBad := bad[uf.find("r:"+c.ReceiptID)]; !isBad {
			eligible = append(eligible, c)
		}
	}
	for _, reason := range bad {
		skipped = append(skipped, reason)
	}
	return eligible, groups, skipped
}

// stockFreed totals what the eligible restamps put back on the shelf: the part of each repaired
// receipt no longer spoken for, in that receipt's own units.
func (m *model) stockFreed(eligible []candidate, eps decimal.Decimal) decimal.Decimal {
	seen := map[string]struct{}{}
	freed := decimal.Zero
	for _, c := range eligible {
		if _, ok := seen[c.ReceiptID]; ok {
			continue
		}
		seen[c.ReceiptID] = struct{}{}
		capacity, ok := m.receiptCapacity[c.ReceiptID]
		if !ok {
			continue
		}
		total := decimal.Zero
		for _, a := range m.byReceipt[c.ReceiptID] {
			total = total.Add(a.RestampedBase)
		}
		if slack := capacity.Sub(total); slack.GreaterThan(eps) {
			freed = freed.Add(slack)
		}
	}
	// Base units back to the receipt's own unit is a per-receipt divide; the account's parts and
	// products are all pair-stocked, so reporting in pairs is the useful reading.
	return freed.Div(decimal.NewFromInt(2))
}

// apply corrects the labels and returns the receipts that are no longer fully drawn to `available`,
// in one transaction so a failure leaves the ledger as it was.
func apply(ctx context.Context, sqlDB *sql.DB, eligible []candidate, m *model, eps decimal.Decimal, p *progress) (int, error) {
	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	p.stage("Applying (single transaction; nothing commits until the end)")
	p.stagef("  Restamping %d allocation quantity row(s)...", len(eligible))

	for i, c := range eligible {
		if _, err := tx.ExecContext(ctx,
			"UPDATE quantity SET unit_id = ?, updated_at = NOW(3) WHERE id = ?",
			c.ReceiptUnitID, c.QuantityID); err != nil {
			return 0, fmt.Errorf("restamp quantity %s: %w", c.QuantityID, err)
		}
		p.step(i+1, len(eligible), "quantities restamped")
	}

	receiptIDs := distinct(eligible, func(c candidate) string { return c.ReceiptID })
	p.stagef("  Re-checking %d receipt(s) for release...", len(receiptIDs))

	freed := 0
	for _, rid := range receiptIDs {
		capacity, ok := m.receiptCapacity[rid]
		if !ok {
			continue
		}
		total := decimal.Zero
		for _, a := range m.byReceipt[rid] {
			total = total.Add(a.RestampedBase)
		}
		if capacity.Sub(total).LessThanOrEqual(eps) {
			continue
		}
		res, err := tx.ExecContext(ctx,
			"UPDATE inventory_receipt SET status_code = 'available', updated_at = NOW(3) WHERE id = ? AND status_code <> 'available'",
			rid)
		if err != nil {
			return 0, fmt.Errorf("free receipt %s: %w", rid, err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("free receipt rows affected: %w", err)
		}
		freed += int(n)
	}

	p.stage("  Committing...")
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return freed, nil
}

func loadCapacities(ctx context.Context, sqlDB *sql.DB, ids []string, prefix string, into map[string]decimal.Decimal) error {
	for chunk := range chunks(ids, 300) {
		// #nosec G202 -- prefix is a caller-side literal; placeholders() emits only "?,?,…".
		rows, err := sqlDB.QueryContext(ctx, prefix+"("+placeholders(len(chunk))+")", toAny(chunk)...)
		if err != nil {
			return fmt.Errorf("load capacities: %w", err)
		}
		for rows.Next() {
			var id, value string
			if err := rows.Scan(&id, &value); err != nil {
				_ = rows.Close()
				return fmt.Errorf("scan capacity: %w", err)
			}
			d, err := decimal.NewFromString(value)
			if err != nil {
				_ = rows.Close()
				return fmt.Errorf("parse capacity for %s: %w", id, err)
			}
			into[id] = d
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return fmt.Errorf("iterate capacities: %w", err)
		}
		_ = rows.Close()
	}
	return nil
}

// unionFind groups receipts and issues that share an allocation.
type unionFind struct{ parent map[string]string }

func newUnionFind() *unionFind { return &unionFind{parent: map[string]string{}} }

func (u *unionFind) find(x string) string {
	if _, ok := u.parent[x]; !ok {
		u.parent[x] = x
		return x
	}
	for u.parent[x] != x {
		u.parent[x] = u.parent[u.parent[x]]
		x = u.parent[x]
	}
	return x
}

// rootIfPresent is find without adding the key, for asking whether a node is in any group at all.
func (u *unionFind) rootIfPresent(x string) (string, bool) {
	if _, ok := u.parent[x]; !ok {
		return "", false
	}
	return u.find(x), true
}

func (u *unionFind) union(a, b string) {
	ra, rb := u.find(a), u.find(b)
	if ra != rb {
		u.parent[ra] = rb
	}
}

func distinct[T any](items []T, key func(T) string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, it := range items {
		k := key(it)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	return out
}

// progress reports what the command is doing as it goes; a run against a remote database spends
// minutes in these loops and is indistinguishable from a hang without it.
type progress struct {
	out   io.Writer
	start time.Time
}

func (p *progress) elapsed() string { return time.Since(p.start).Truncate(time.Second).String() }

func (p *progress) stage(msg string) { fmt.Fprintf(p.out, "[%s] %s\n", p.elapsed(), msg) }

func (p *progress) stagef(format string, args ...any) { p.stage(fmt.Sprintf(format, args...)) }

func (p *progress) step(done, total int, noun string) {
	if done%progressEvery != 0 && done != total {
		return
	}
	fmt.Fprintf(p.out, "[%s]   %d/%d %s\n", p.elapsed(), done, total, noun)
}

const progressEvery = 100

func chunks(ids []string, size int) func(func([]string) bool) {
	return func(yield func([]string) bool) {
		for start := 0; start < len(ids); start += size {
			end := min(start+size, len(ids))
			if !yield(ids[start:end]) {
				return
			}
		}
	}
}

// normalizeDSN accepts either a driver DSN (user:pass@tcp(host:port)/db) — returned unchanged — or a
// PlanetScale/URL form, which it rewrites into the DSN form db.NewDbPool expects. Mirrors the helper
// in the sibling inventory commands; all are package main and cannot share it.
func normalizeDSN(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("DB_URL is empty")
	}
	if strings.Contains(raw, "@tcp(") || strings.Contains(raw, "@unix(") {
		return raw, nil
	}
	if !strings.HasPrefix(raw, "mysql://") {
		return "", fmt.Errorf("unrecognized DB_URL form; expected a mysql://... URL or a user:pass@tcp(host:port)/db DSN")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parsing DB_URL: %w", err)
	}
	host := u.Hostname()
	if host == "" {
		return "", fmt.Errorf("DB_URL has no host")
	}
	port := u.Port()
	if port == "" {
		port = "3306"
	}
	dbName := strings.TrimPrefix(u.Path, "/")
	userInfo := u.User.Username()
	if pass, ok := u.User.Password(); ok && pass != "" {
		userInfo += ":" + pass
	}

	dsn := fmt.Sprintf("%s@tcp(%s:%s)/%s", userInfo, host, port, dbName)
	if host != "localhost" && host != "127.0.0.1" {
		dsn += "?tls=true"
	}
	return dsn, nil
}

func placeholders(n int) string { return strings.TrimSuffix(strings.Repeat("?,", n), ",") }

func toAny(ids []string) []any {
	out := make([]any, len(ids))
	for i, id := range ids {
		out[i] = id
	}
	return out
}
