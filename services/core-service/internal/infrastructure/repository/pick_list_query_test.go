package repository

import (
	gosql "database/sql"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/open-mrp/api/shared/pagination"
)

// The pick list's plan is invisible from its results: every index returns the same page, so a wrong
// one shows up only as latency on the one account large enough to feel it. These pin the property the
// index hint exists for — see pickListQuery.indexHint.
func TestBuildPickListQuery_IndexHintFollowsSortAndFilters(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		q    pickListQuery
		want string
	}{
		// Open picks are a tiny slice of a mostly-closed table, so the filter has to lead the index or
		// the scan reads the account's closed history before a page fills.
		{"ship-by open pins the filter", pickListQuery{SortByShipBy: true, Status: new("open")}, pickOpenShipByIndex},
		{"ship-by closed reads in order", pickListQuery{SortByShipBy: true, Status: new("closed")}, pickShipByIndex},
		{"ship-by unfiltered reads in order", pickListQuery{SortByShipBy: true}, pickShipByIndex},
		{"created open pins the filter", pickListQuery{Status: new("open")}, pickOpenCreatedIndex},
		{"created closed reads in order", pickListQuery{Status: new("closed")}, pickCreatedIndex},
		{"created unfiltered reads in order", pickListQuery{}, pickCreatedIndex},
		// A small customer set's picks are a few of the account's; walking the sort index for them reads it all.
		{"ship-by open customer pins buyer and status", pickListQuery{SortByShipBy: true, Status: new("open"), BuyerIDs: []string{"ac_c"}, DriveFromBuyers: true}, pickBuyerOpenShipByIndex},
		{"ship-by customer pins the buyer", pickListQuery{SortByShipBy: true, BuyerIDs: []string{"ac_c"}, DriveFromBuyers: true}, pickBuyerShipByIndex},
		{"created open customer pins buyer and status", pickListQuery{Status: new("open"), BuyerIDs: []string{"ac_c"}, DriveFromBuyers: true}, pickBuyerOpenCreatedIndex},
		{"created closed customer pins the buyer", pickListQuery{Status: new("closed"), BuyerIDs: []string{"ac_c"}, DriveFromBuyers: true}, pickBuyerCreatedIndex},
		// A large set matches often, so reading in sort order fills a page sooner than sorting the set.
		{"large customer set reads in order", pickListQuery{BuyerIDs: []string{"ac_c"}}, pickCreatedIndex},
		// How many rows a prefix matches decides between these, so MySQL estimates it.
		{"number prefix offers both", pickListQuery{Search: pickSearch{NumberPrefix: "22%"}}, pickCreatedIndex + ", " + pickAccountNumberIndex},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.q.AccountID, tc.q.Limit = "ac_1", 51
			query, _ := buildPickListQuery(tc.q)

			if want := " FROM pick p FORCE INDEX (" + tc.want + ")"; !strings.Contains(query, want) {
				t.Errorf("index hint is not %s:\n%s", tc.want, query)
			}
		})
	}
}

// Each sort pages in its own direction, and a flipped comparison or ORDER BY silently repeats or skips
// rows at page boundaries.
func TestBuildPickListQuery_KeysetMatchesSortDirection(t *testing.T) {
	t.Parallel()

	cursorAt := gosql.NullTime{Valid: true}
	cursorID := gosql.NullString{String: "pk_1", Valid: true}

	for _, tc := range []struct {
		name         string
		sortByShipBy bool
		dir          pagination.Direction
		want         []string
	}{
		{"ship-by forward is soonest first", true, pagination.DirectionForward, []string{
			"(p.ship_by_sort_date > CAST(? AS DATE) OR (p.ship_by_sort_date = CAST(? AS DATE) AND p.id > ?))",
			"ORDER BY p.ship_by_sort_date ASC, p.id ASC",
		}},
		{"ship-by backward", true, pagination.DirectionBackward, []string{
			"(p.ship_by_sort_date < CAST(? AS DATE) OR (p.ship_by_sort_date = CAST(? AS DATE) AND p.id < ?))",
			"ORDER BY p.ship_by_sort_date DESC, p.id DESC",
		}},
		{"created forward is newest first", false, pagination.DirectionForward, []string{
			"(p.created_at < ? OR (p.created_at = ? AND p.id < ?))",
			"ORDER BY p.created_at DESC, p.id DESC",
		}},
		{"created backward", false, pagination.DirectionBackward, []string{
			"(p.created_at > ? OR (p.created_at = ? AND p.id > ?))",
			"ORDER BY p.created_at ASC, p.id ASC",
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			query, args := buildPickListQuery(pickListQuery{
				AccountID: "ac_1", SortByShipBy: tc.sortByShipBy, Direction: tc.dir,
				CursorAt: cursorAt, CursorID: cursorID, Limit: 51,
			})

			for _, want := range tc.want {
				if !strings.Contains(query, want) {
					t.Errorf("query lacks %q:\n%s", want, query)
				}
			}
			if got, want := strings.Count(query, "?"), len(args); got != want {
				t.Errorf("%d placeholders but %d args", got, want)
			}
		})
	}
}

func TestNewPickSearch(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		q    *string
		want pickSearch
	}{
		{"absent", nil, pickSearch{}},
		{"empty", new(""), pickSearch{}},
		{"one character is a number prefix", new("2"), pickSearch{NumberPrefix: "2%"}},
		{"two characters are a number prefix", new("22"), pickSearch{NumberPrefix: "22%"}},
		{"a prefix escapes LIKE wildcards", new("_%"), pickSearch{NumberPrefix: `\_\%%`}},
		{"three characters are a substring phrase", new("235"), pickSearch{Phrase: `"235"`}},
		{"characters are counted, not bytes", new("ñé"), pickSearch{NumberPrefix: "ñé%"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := newPickSearch(tc.q); got != tc.want {
				t.Errorf("newPickSearch = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// Every filter and search mode together: a placeholder/arg mismatch binds values to the wrong
// predicates, which fails loudly only when the count is off, not when the order is.
func TestBuildPickListQuery_ArgsBindInPlaceholderOrder(t *testing.T) {
	t.Parallel()

	start := gosql.NullTime{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true}
	end := gosql.NullTime{Time: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), Valid: true}
	base := pickListQuery{
		AccountID: "ac_1", Status: new("open"),
		BuyerIDs: []string{"ac_c1", "ac_c2"}, ProductLineIDs: []string{"pl_1"},
		StartDate: start, EndDate: end, Limit: 51,
	}

	t.Run("phrase search", func(t *testing.T) {
		t.Parallel()
		q := base
		q.Search = pickSearch{Phrase: `"235"`}
		query, args := buildPickListQuery(q)

		want := []any{
			"ac_1", `"235"`, "ac_1", `"235"`, "ac_1", `"235"`, "ac_1", `"235"`,
			"ac_1", "ac_c1", "ac_c2", "pl_1", start.Time, end.Time, int32(51),
		}
		if !reflect.DeepEqual(args, want) {
			t.Errorf("args = %v\nwant %v", args, want)
		}
		if got := strings.Count(query, "?"); got != len(args) {
			t.Errorf("%d placeholders but %d args", got, len(args))
		}
		if strings.Contains(query, "FORCE INDEX") {
			t.Errorf("a phrase search drives from its match set, not an index hint:\n%s", query)
		}
	})

	t.Run("prefix search for one customer", func(t *testing.T) {
		t.Parallel()
		q := base
		q.BuyerIDs = []string{"ac_c1"}
		q.DriveFromBuyers = true
		q.Search = pickSearch{NumberPrefix: "22%"}
		query, args := buildPickListQuery(q)

		want := []any{"ac_1", "ac_c1", "22%", "pl_1", start.Time, end.Time, int32(51)}
		if !reflect.DeepEqual(args, want) {
			t.Errorf("args = %v\nwant %v", args, want)
		}
		if got := strings.Count(query, "?"); got != len(args) {
			t.Errorf("%d placeholders but %d args", got, len(args))
		}
	})
}

// Several customers are each paged from their own run of the buyer index and merged, so no read
// sorts a customer's whole history or walks other customers' picks.
func TestBuildPickListQuery_MergesAFewCustomers(t *testing.T) {
	t.Parallel()

	cursorAt := gosql.NullTime{Time: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), Valid: true}
	q := pickListQuery{
		AccountID: "ac_1", Status: new("open"), SortByShipBy: true,
		BuyerIDs: []string{"ac_c1", "ac_c2"}, ProductLineIDs: []string{"pl_1"},
		CursorAt: cursorAt, CursorID: gosql.NullString{String: "pk_9", Valid: true},
		Direction: pagination.DirectionForward, Limit: 51,
	}
	query, args := buildPickListQuery(q)

	for _, want := range []string{
		"FORCE INDEX (" + pickBuyerOpenShipByIndex + ") WHERE p.account_id = ? AND p.buyer_account_id = ?",
		" UNION ALL ",
		"ORDER BY p.ship_by_sort_date ASC, p.id ASC LIMIT ?)",
		") merged ORDER BY sort_at ASC, id ASC LIMIT ?",
	} {
		if !strings.Contains(query, want) {
			t.Errorf("query lacks %q:\n%s", want, query)
		}
	}
	if got := strings.Count(query, "(SELECT p.id, p.ship_by_sort_date AS sort_at"); got != 2 {
		t.Errorf("%d per-customer reads, want 2", got)
	}

	arm := func(buyer string) []any {
		return []any{"ac_1", buyer, "pl_1", cursorAt.Time, cursorAt.Time, "pk_9", int32(51)}
	}
	want := append(append(arm("ac_c1"), arm("ac_c2")...), int32(51))
	if !reflect.DeepEqual(args, want) {
		t.Errorf("args = %v\nwant %v", args, want)
	}
	if got := strings.Count(query, "?"); got != len(args) {
		t.Errorf("%d placeholders but %d args", got, len(args))
	}
}

func TestPickListQuery_MergesOnlyAFewCustomersOffThePhrasePath(t *testing.T) {
	t.Parallel()

	many := make([]string, pickBuyerMergeMax+1)
	for i := range many {
		many[i] = "ac_c"
	}
	for _, tc := range []struct {
		name string
		q    pickListQuery
		want bool
	}{
		{"one customer is a single sorted read", pickListQuery{BuyerIDs: []string{"ac_c1"}}, false},
		{"a few customers merge", pickListQuery{BuyerIDs: []string{"ac_c1", "ac_c2"}}, true},
		{"too many customers to merge", pickListQuery{BuyerIDs: many}, false},
		{"a phrase search drives from its matches", pickListQuery{BuyerIDs: []string{"ac_c1", "ac_c2"}, Search: pickSearch{Phrase: `"235"`}}, false},
	} {
		if got := tc.q.mergesBuyers(); got != tc.want {
			t.Errorf("%s: mergesBuyers = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// The count stops at the cap, so a customer set's size costs at most pickBuyerScanCap index entries.
func TestBuildPickBuyerCountQuery_StopsAtTheCap(t *testing.T) {
	t.Parallel()

	query, args := buildPickBuyerCountQuery("ac_1", []string{"ac_c1", "ac_c2"})
	if !strings.Contains(query, "FORCE INDEX ("+pickBuyerCreatedIndex+")") || !strings.Contains(query, "LIMIT ?) capped") {
		t.Errorf("count is not a capped read of the buyer index:\n%s", query)
	}
	want := []any{"ac_1", "ac_c1", "ac_c2", pickBuyerScanCap}
	if !reflect.DeepEqual(args, want) {
		t.Errorf("args = %v, want %v", args, want)
	}
	if got := strings.Count(query, "?"); got != len(args) {
		t.Errorf("%d placeholders but %d args", got, len(args))
	}
}

// The hint names the index, so a rename or a dropped migration turns the query into a 1176 at runtime
// rather than a compile error.
func TestPickListIndexes_AreDeclaredInMigrations(t *testing.T) {
	t.Parallel()

	schema := migrationsText(t)
	for _, index := range []string{
		pickShipByIndex, pickOpenShipByIndex, pickCreatedIndex, pickOpenCreatedIndex,
		pickBuyerShipByIndex, pickBuyerCreatedIndex, pickBuyerOpenShipByIndex, pickBuyerOpenCreatedIndex, pickAccountNumberIndex,
	} {
		if !strings.Contains(schema, index) {
			t.Errorf("%s is FORCE INDEX'd by the pick list but no migration creates it", index)
		}
	}
}

func migrationsText(t *testing.T) string {
	t.Helper()

	const dir = "../../../../../shared/db/migrations"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}

	var b strings.Builder
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		body, err := os.ReadFile(dir + "/" + entry.Name())
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		b.Write(body)
	}
	return b.String()
}
