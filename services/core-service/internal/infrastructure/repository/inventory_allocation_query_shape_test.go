package repository

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const queriesDir = "../queries"

// The two locking reads an allocate transaction depends on, and both properties matter:
//
//   - FOR UPDATE. The claim is what makes a second chain wait and then re-evaluate `status_code`
//     against what the first committed; the receipt read is what stops two allocators drawing the
//     same stock. Both are also what keep the transaction's REPEATABLE READ view from opening before
//     the locks are held.
//   - No join to `unit`. A locking read locks every row it touches, and `unit` rows are shared by
//     every account in the database; joining one here serialises allocation across the whole estate.
//     Callers resolve ratios through GetUnitRatios instead.
//
// Losing either is silent — the queries still return the right rows — until an account is busy enough
// for two chains to overlap, and then it shows up as receipts drawn on for twice what they held.
func TestAllocationClaimingQueries_AreLockingReadsWithoutUnitJoins(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"ClaimOpenIssueForAllocation", "FindReceiptsForAllocation"} {
		body := queryBody(t, "inventory_reservation.sql", name)

		if !strings.Contains(body, "FOR UPDATE") {
			t.Errorf("%s is not a locking read: allocation transactions running side by side will both "+
				"draw the same stock", name)
		}
		if unitJoinRe.MatchString(body) {
			t.Errorf("%s joins `unit` under FOR UPDATE, which locks rows every account shares; resolve "+
				"the ratio through GetUnitRatios", name)
		}
	}
}

// The verification reads must lock BOTH tables they touch, and after the vtgate fix that is a
// property of two things together rather than one clause.
//
// They exist to see allocations committed by writers this transaction never serialised against, which
// only a current read can do. A locking read is current for the tables it locks and snapshot-bound for
// everything else, so if `quantity` were reachable any other way — a subquery, a lateral, a second
// statement — an allocation committed after this transaction's view opened would be found in `ia`,
// joined against a quantity row the snapshot cannot see, and silently dropped by the INNER JOIN.
//
// It used to say FOR UPDATE OF ia, q, which stated that directly. vtgate rejects the OF clause with a
// 1105 syntax error, so it is now a bare FOR UPDATE over a two-table join, which locks exactly the
// same two tables. That makes the join load-bearing rather than incidental: hence the second
// assertion.
func TestVerificationReads_AreCurrentAndCoverTheirSatellite(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"ReadReceiptAllocationsForUpdate", "ReadIssueCoverageForUpdate"} {
		body := queryBody(t, "inventory_reservation.sql", name)
		statement := body[strings.Index(body, "-- name:"):]

		if !strings.Contains(statement, "FOR UPDATE") {
			t.Errorf("%s is not a locking read: a snapshot read cannot see the unlocked writer this "+
				"query exists to catch", name)
		}
		if !strings.Contains(statement, "JOIN quantity q") {
			t.Errorf("%s no longer joins quantity: with a bare FOR UPDATE the join is what brings the "+
				"satellite row under the lock, so without it an allocation committed after this "+
				"transaction's view opened is silently dropped", name)
		}
		if unitJoinRe.MatchString(statement) {
			t.Errorf("%s joins `unit` under a locking read, which locks rows every account shares; sum "+
				"through GetUnitRatios instead", name)
		}
	}
}

// Discovery must NOT be a locking read, and must project nothing but the keyset.
//
// It names candidates and decides nothing — every id it returns is re-read by
// ClaimOpenIssueForAllocation, by primary key, under FOR UPDATE, in its own transaction. That is what
// makes a non-locking read safe here and is exactly what 3e99b962 did not do when it made
// FindOpenIssuesForItem non-locking and then fed its quantity and allocated sum straight into the
// arithmetic.
//
// Locking here cost the item's whole (account_id, item_id, 'open', created_at) range plus every gap
// between and after it — so no batch scan, shipment or reservation for that item could record new
// demand for the life of the transaction — and X locks on up to 200 shared `quantity` rows. Joining
// `quantity` would reintroduce the second half of that even without the lock, by making the read
// touch rows it has no reason to.
func TestOpenIssueDiscovery_IsNotALockingReadAndProjectsOnlyTheKeyset(t *testing.T) {
	t.Parallel()

	body := queryBody(t, "inventory_reservation.sql", "ListOpenIssueIDsForItemPaged")

	if strings.Contains(body, "FOR UPDATE") {
		t.Error("ListOpenIssueIDsForItemPaged is a locking read: it locks the item's whole open range " +
			"and the trailing gap every new open issue lands in, which stalls every flow that records " +
			"demand for that item until the allocate transaction commits")
	}
	if strings.Contains(body, "JOIN quantity") {
		t.Error("ListOpenIssueIDsForItemPaged joins quantity: discovery must be answerable from " +
			"inventory_issue_open_paging_idx alone, and nothing may be decided from what it returns")
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
	// vtgate rejects the OF clause; see TestVitessCompat_NoForUpdateOfClause.
	forUpdateOfRe = regexp.MustCompile(`(?i)\bFOR\s+UPDATE\s+OF\b`)
)

// queryBody returns one named sqlc query: its `-- name:` line and the statement below it, stopping at
// the statement's terminating semicolon.
//
// It used to run to the NEXT `-- name:` line, which swept in the prose comment block above the
// following query — so every assertion here was really made against two queries' worth of text. That
// is silent in both directions: a stray "FOR UPDATE" in a neighbouring comment fails a test that
// should pass, and a query that LOSES its FOR UPDATE keeps passing while a neighbour's comment
// happens to mention one. These comment blocks discuss each other's locking constantly, so it was a
// matter of time.
func queryBody(t *testing.T, file, name string) string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(queriesDir, file)) // #nosec G304 -- fixed query directory
	if err != nil {
		t.Fatalf("read %s: %v", file, err)
	}
	text := string(raw)

	for _, loc := range queryNameRe.FindAllStringSubmatchIndex(text, -1) {
		if text[loc[2]:loc[3]] != name {
			continue
		}
		// The first semicolon on a line that is not a comment. The prose in these files uses
		// semicolons freely — "may only draw from stock sitting there; an unpinned issue draws from
		// anywhere" sits inside FindReceiptsForAllocation — so scanning for a bare ";" ends the
		// statement in the middle of its own documentation and drops the FOR UPDATE below it.
		rest := text[loc[0]:]
		offset := 0
		for _, line := range strings.SplitAfter(rest, "\n") {
			if !strings.HasPrefix(strings.TrimSpace(line), "--") {
				if i := strings.Index(line, ";"); i >= 0 {
					return rest[:offset+i+1]
				}
			}
			offset += len(line)
		}
		t.Fatalf("query %s in %s has no terminating semicolon", name, file)
	}

	t.Fatalf("query %s not found in %s", name, file)
	return ""
}

// A guard on the guard. If queryBody ever runs past its own statement again, every assertion in this
// file quietly starts testing the wrong text.
func TestQueryBody_StopsAtItsOwnStatement(t *testing.T) {
	t.Parallel()

	body := queryBody(t, "inventory_reservation.sql", "GetUnitRatios")
	if n := strings.Count(body, "-- name:"); n != 1 {
		t.Errorf("queryBody returned %d queries' worth of text for GetUnitRatios; every assertion in this "+
			"file is being made against the wrong input", n)
	}
	if !strings.HasSuffix(strings.TrimSpace(body), ";") {
		t.Error("queryBody did not stop at the statement's semicolon")
	}
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
