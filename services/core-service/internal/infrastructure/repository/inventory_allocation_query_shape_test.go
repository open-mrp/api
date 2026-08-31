package repository

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const queriesDir = "../queries"

// Allocation is driven by a command that a batch scan, a receipt landing and a receiving order being
// stocked all enqueue, so two paging chains for one item overlap routinely. These reads are what stop
// the second chain drawing stock the first already took, and both properties matter:
//
//   - FOR UPDATE makes the second chain wait and then re-evaluate against what the first committed.
//     It also keeps the transaction's REPEATABLE READ view from being created before the locks are
//     held, which is what let the allocated-sum reads that follow see an issue as untouched.
//   - No join to `unit`. A locking read locks every row it touches, and `unit` rows are shared by
//     every account in the database; joining one here serialises allocation across the whole
//     estate. Callers resolve ratios through GetUnitRatios instead.
//
// Losing either is silent — the queries still return the right rows — until an account is busy enough
// for two chains to overlap, and then it shows up as receipts drawn on for twice what they held.
func TestAllocationClaimingQueries_AreLockingReadsWithoutUnitJoins(t *testing.T) {
	t.Parallel()

	for _, q := range []struct{ file, name string }{
		{"inventory_reservation.sql", "FindOpenIssuesForItemPaged"},
		{"inventory_reservation.sql", "FindReceiptsForAllocation"},
		{"receiving_order.sql", "FindOpenIssuesForItem"},
	} {
		body := queryBody(t, q.file, q.name)

		if !strings.Contains(body, "FOR UPDATE") {
			t.Errorf("%s is not a locking read: allocation transactions running side by side will both draw the same stock", q.name)
		}
		if unitJoinRe.MatchString(body) {
			t.Errorf("%s joins `unit` under FOR UPDATE, which locks rows every account shares; resolve the ratio through GetUnitRatios", q.name)
		}
	}
}

// FetchOnHandInventoryBulk is the item list's on-hand column and has to agree with the item detail
// page. It read several times the true level because the receipt sum carried a LEFT JOIN onto
// inventory_allocation that nothing referenced — fanning each receipt out once per allocation against
// it — and because it added pairs to each without going through either one's ratio.
func TestFetchOnHandInventoryBulk_NetsPerRowInUnits(t *testing.T) {
	t.Parallel()

	body := queryBody(t, "inventory_query.sql", "FetchOnHandInventoryBulk")

	if strings.Contains(body, "LEFT JOIN inventory_allocation") {
		t.Error("FetchOnHandInventoryBulk joins inventory_allocation into a sum it does not use: every receipt is counted once per allocation against it")
	}
	if !strings.Contains(body, "ratio_numerator") {
		t.Error("FetchOnHandInventoryBulk adds quantities without their unit ratios")
	}
	if strings.Contains(body, "AS SIGNED") {
		t.Error("FetchOnHandInventoryBulk rounds to a whole number, which loses the half-units an item stocked in pairs and drawn on in each holds")
	}
}

// The statuses an inventory issue is ever written with. FetchCurrentInventoryForItem subtracted
// issues with status_code = 'committed', which is not one of them, so it matched nothing and every
// item promised its whole shelf however much of it was already sold.
func TestInventoryIssueStatusFilters_UseStatusesTheLedgerWrites(t *testing.T) {
	t.Parallel()

	known := map[string]bool{"open": true, "reserved": true, "closed": true}

	for _, file := range sqlFiles(t) {
		body, err := os.ReadFile(file) // #nosec G304 -- fixed test fixture directory
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		for _, match := range issueStatusRe.FindAllStringSubmatch(string(body), -1) {
			// Group 1 is the `= 'x'` form, group 2 the `IN ('x', 'y')` list; only one ever matches.
			for _, status := range strings.Split(match[1]+match[2], ",") {
				status = strings.Trim(strings.TrimSpace(status), "'")
				if status != "" && !known[status] {
					t.Errorf("%s filters inventory_issue on status %q, which the ledger never writes", filepath.Base(file), status)
				}
			}
		}
	}
}

var (
	unitJoinRe = regexp.MustCompile(`(?i)JOIN\s+unit\s`)
	// Matches `ii.status_code = 'x'` and `ii.status_code IN ('x', 'y')` on an inventory_issue alias.
	issueStatusRe = regexp.MustCompile(`(?i)\bii\.status_code\s*(?:=\s*'([a-z_]+)'|IN\s*\(([^)]*)\))`)
	queryNameRe   = regexp.MustCompile(`(?m)^-- name: (\w+) :`)
)

// queryBody returns the text of one named sqlc query, from its `-- name:` line to the next one.
func queryBody(t *testing.T, file, name string) string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(queriesDir, file)) // #nosec G304 -- fixed query directory
	if err != nil {
		t.Fatalf("read %s: %v", file, err)
	}
	text := string(raw)

	locs := queryNameRe.FindAllStringSubmatchIndex(text, -1)
	for i, loc := range locs {
		if text[loc[2]:loc[3]] != name {
			continue
		}
		end := len(text)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		return text[loc[0]:end]
	}

	t.Fatalf("query %s not found in %s", name, file)
	return ""
}

func sqlFiles(t *testing.T) []string {
	t.Helper()

	files, err := filepath.Glob(filepath.Join(queriesDir, "*.sql"))
	if err != nil {
		t.Fatalf("list queries: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no query files found")
	}
	return files
}
