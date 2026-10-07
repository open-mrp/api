package repository

import (
	gosql "database/sql"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/open-mrp/api/shared/db"
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
		// How many rows a prefix or a creation window holds decides between these, so MySQL estimates it.
		{"number prefix offers both", pickListQuery{Search: pickSearch{NumberPrefix: "22%"}}, pickCreatedIndex + ", " + pickAccountNumberIndex},
		{"customer number prefix offers both", pickListQuery{Search: pickSearch{NumberPrefix: "22%"}, BuyerIDs: []string{"ac_c"}, DriveFromBuyers: true}, pickBuyerCreatedIndex + ", " + pickAccountNumberIndex},
		{"ship-by window offers the created key", pickListQuery{SortByShipBy: true, StartDate: gosql.NullTime{Valid: true}}, pickShipByIndex + ", " + pickCreatedIndex},
		{"ship-by open customer window", pickListQuery{SortByShipBy: true, Status: new("open"), EndDate: gosql.NullTime{Valid: true}, BuyerIDs: []string{"ac_c"}, DriveFromBuyers: true}, pickBuyerOpenShipByIndex + ", " + pickBuyerOpenCreatedIndex},
		{"created window reads the created key in order", pickListQuery{StartDate: gosql.NullTime{Valid: true}}, pickCreatedIndex},
		// A counted set's size is known, so the narrower of it and a prefix is chosen here, not estimated.
		{"counted customer set reads its ranges", pickListQuery{Search: pickSearch{NumberPrefix: "2%"}, BuyerIDs: []string{"ac_c1", "ac_c2"}, DriveFromBuyers: true}, pickBuyerCreatedIndex},
		{"prefix narrower than a counted set", pickListQuery{Search: pickSearch{NumberPrefix: "2%"}, BuyerIDs: []string{"ac_c1", "ac_c2"}, DriveFromNumberPrefix: true}, pickAccountNumberIndex},
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
		{"three characters are a substring phrase", new("235"), pickSearch{Phrase: db.NgramSubstring{Tokens: "+23 +35", Like: "%235%"}}},
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

	t.Run("phrase with too many matches to read up front", func(t *testing.T) {
		t.Parallel()
		q := base
		q.Search = newPickSearch(new("235"))
		query, args := buildPickListQuery(q)

		want := []any{
			"ac_1",
			"ac_1", "+23 +35", "%235%", "ac_1", "+23 +35", "%235%", "ac_1", "+23 +35", "%235%", "ac_1", "+23 +35", "%235%",
			"ac_c1", "ac_c2", "pl_1", start.Time, end.Time, int32(51),
		}
		if !reflect.DeepEqual(args, want) {
			t.Errorf("args = %v\nwant %v", args, want)
		}
		if got := strings.Count(query, "?"); got != len(args) {
			t.Errorf("%d placeholders but %d args", got, len(args))
		}
		if !strings.Contains(query, "AND p.id IN (SELECT id FROM (") {
			t.Errorf("a common phrase is not a residual:\n%s", query)
		}
	})

	t.Run("phrase read from its few matches", func(t *testing.T) {
		t.Parallel()
		q := base
		q.Search = newPickSearch(new("235"))
		q.PhraseIDs, q.DriveFromPhrase = []string{"pk_1", "pk_2"}, true
		query, args := buildPickListQuery(q)

		want := []any{"ac_1", "pk_1", "pk_2", "ac_c1", "ac_c2", "pl_1", start.Time, end.Time, int32(51)}
		if !reflect.DeepEqual(args, want) {
			t.Errorf("args = %v\nwant %v", args, want)
		}
		if got := strings.Count(query, "?"); got != len(args) {
			t.Errorf("%d placeholders but %d args", got, len(args))
		}
		if !strings.Contains(query, "FROM pick p FORCE INDEX (PRIMARY) WHERE p.account_id = ? AND p.id IN (?, ?)") {
			t.Errorf("a rare phrase does not drive by primary key:\n%s", query)
		}
	})

	t.Run("product lines drive", func(t *testing.T) {
		t.Parallel()
		q := base
		q.DriveFromProductLines = true
		query, args := buildPickListQuery(q)

		want := []any{"pl_1", "ac_1", "ac_c1", "ac_c2", start.Time, end.Time, int32(51)}
		if !reflect.DeepEqual(args, want) {
			t.Errorf("args = %v\nwant %v", args, want)
		}
		if got := strings.Count(query, "?"); got != len(args) {
			t.Errorf("%d placeholders but %d args", got, len(args))
		}
		if !strings.Contains(query, ") matched JOIN pick p ON p.id = matched.id WHERE") || strings.Contains(query, "EXISTS") {
			t.Errorf("product lines do not drive from their matches:\n%s", query)
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
		{"a phrase search drives from its matches", pickListQuery{BuyerIDs: []string{"ac_c1", "ac_c2"}, Search: newPickSearch(new("235"))}, false},
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

// Every arm confirms the phrase with LIKE; a phrase with no token the index holds is LIKE alone, and the
// number arm then reads the covering number key.
func TestPickPhraseArms_ConfirmEveryMatchWithLike(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		term        string
		wantMatches int
		wantHint    bool
	}{
		{term: "235", wantMatches: pickPhraseArmCount},
		{term: "aia", wantMatches: 0, wantHint: true},
	} {
		t.Run(tc.term, func(t *testing.T) {
			t.Parallel()
			query, args := buildPickPhraseIDsQuery(pickListQuery{AccountID: "ac_1", Search: newPickSearch(new(tc.term))})

			if got := strings.Count(query, "MATCH("); got != tc.wantMatches {
				t.Errorf("%d MATCH predicates, want %d:\n%s", got, tc.wantMatches, query)
			}
			if got := strings.Count(query, " LIKE ?"); got != pickPhraseArmCount {
				t.Errorf("%d LIKE predicates, want one per arm:\n%s", got, query)
			}
			if got := strings.Contains(query, "FORCE INDEX ("+pickAccountNumberIndex+")"); got != tc.wantHint {
				t.Errorf("number key hinted = %v, want %v:\n%s", got, tc.wantHint, query)
			}
			if got := strings.Count(query, "?"); got != len(args) {
				t.Errorf("%d placeholders but %d args", got, len(args))
			}
		})
	}
}

// A phrase paged in memory reads its sort key with every match, and the customer arms force the buyer
// key that holds it: with the longer select list MySQL otherwise picks one that does not, and reads every
// pick of a matched customer. Where the page is read in SQL the arms stay id-only.
func TestPickPhraseArms_ReadThePageColumnsFromACoveringKey(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		q         pickListQuery
		wantCols  string
		wantIndex string
	}{
		{"ship-by", pickListQuery{SortByShipBy: true}, "pk.ship_by_sort_date FROM", pickBuyerOpenShipByIndex},
		{"created", pickListQuery{}, "pk.created_at FROM", pickBuyerOpenCreatedIndex},
		{"created window", pickListQuery{StartDate: gosql.NullTime{Valid: true}}, "pk.created_at FROM", pickBuyerOpenCreatedIndex},
		{"ship-by window", pickListQuery{SortByShipBy: true, EndDate: gosql.NullTime{Valid: true}}, "", ""},
		{"product lines", pickListQuery{ProductLineIDs: []string{"pl_1"}}, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			q := tc.q
			q.AccountID, q.Search = "ac_1", newPickSearch(new("235"))
			query, args := buildPickPhraseIDsQuery(q)

			if tc.wantCols == "" {
				if q.phrasePagesInMemory() || strings.Count(query, "SELECT pk.id FROM") != pickPhraseArmCount {
					t.Errorf("arms read more than ids for a page read in SQL:\n%s", query)
				}
				return
			}
			if !q.phrasePagesInMemory() {
				t.Fatalf("page is not computed in memory")
			}
			if got := strings.Count(query, "pk.buyer_account_id, pk.finished_at, "+tc.wantCols); got != pickPhraseArmCount {
				t.Errorf("%d arms read the page columns, want every arm:\n%s", got, query)
			}
			if got := strings.Count(query, "JOIN pick pk FORCE INDEX ("+tc.wantIndex+")"); got != 2 {
				t.Errorf("%d customer arms force %s, want 2:\n%s", got, tc.wantIndex, query)
			}
			if got := strings.Count(query, "?"); got != len(args) {
				t.Errorf("%d placeholders but %d args", got, len(args))
			}
		})
	}
}

func TestPickPhrasePage_FiltersSortsAndPages(t *testing.T) {
	t.Parallel()

	day := func(d int) time.Time { return time.Date(2026, 1, d, 0, 0, 0, 0, time.UTC) }
	open := gosql.NullTime{}
	done := gosql.NullTime{Time: day(1), Valid: true}
	buyer := func(id string) gosql.NullString { return gosql.NullString{String: id, Valid: id != ""} }
	matches := []pickPhraseMatch{
		{ID: "pk_c", BuyerID: buyer("ac_a"), FinishedAt: open, SortAt: day(2)},
		{ID: "pk_a", BuyerID: buyer("ac_b"), FinishedAt: done, SortAt: day(2)},
		{ID: "pk_d", BuyerID: buyer("ac_a"), FinishedAt: done, SortAt: day(1)},
		{ID: "pk_b", BuyerID: buyer(""), FinishedAt: open, SortAt: day(3)},
	}

	for _, tc := range []struct {
		name string
		q    pickListQuery
		want []string
	}{
		{"ship-by ascending, ties by id", pickListQuery{SortByShipBy: true}, []string{"pk_d", "pk_a", "pk_c", "pk_b"}},
		{"created descending", pickListQuery{}, []string{"pk_b", "pk_c", "pk_a", "pk_d"}},
		{"limit", pickListQuery{SortByShipBy: true, Limit: 2}, []string{"pk_d", "pk_a"}},
		{"open", pickListQuery{SortByShipBy: true, Status: new("open")}, []string{"pk_c", "pk_b"}},
		{"closed", pickListQuery{SortByShipBy: true, Status: new("closed")}, []string{"pk_d", "pk_a"}},
		{"customers exclude a pick with none", pickListQuery{SortByShipBy: true, BuyerIDs: []string{"ac_a"}}, []string{"pk_d", "pk_c"}},
		{"created window", pickListQuery{
			StartDate: gosql.NullTime{Time: day(2), Valid: true}, EndDate: gosql.NullTime{Time: day(2), Valid: true},
		}, []string{"pk_c", "pk_a"}},
		// A ship-by cursor compares by date, as CAST(? AS DATE) does.
		{"ship-by after a cursor", pickListQuery{
			SortByShipBy: true, CursorAt: gosql.NullTime{Time: day(2).Add(15 * time.Hour), Valid: true},
			CursorID: gosql.NullString{String: "pk_a", Valid: true},
		}, []string{"pk_c", "pk_b"}},
		{"ship-by before a cursor", pickListQuery{
			SortByShipBy: true, Direction: pagination.DirectionBackward,
			CursorAt: gosql.NullTime{Time: day(2), Valid: true}, CursorID: gosql.NullString{String: "pk_c", Valid: true},
		}, []string{"pk_a", "pk_d"}},
		{"created after a cursor", pickListQuery{
			CursorAt: gosql.NullTime{Time: day(2), Valid: true}, CursorID: gosql.NullString{String: "pk_c", Valid: true},
		}, []string{"pk_a", "pk_d"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			q := tc.q
			if q.Direction == "" {
				q.Direction = pagination.DirectionForward
			}
			if q.Limit == 0 {
				q.Limit = 10
			}
			q.PhraseMatches = matches
			if got := q.phrasePage(); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("page = %v, want %v", got, tc.want)
			}
		})
	}
}

// Sizing a filter must stop at its cap: the phrase read streams (UNION ALL) and the counts are capped
// reads, so none of them collects every match first.
func TestPickFilterSizing_StopsAtTheCap(t *testing.T) {
	t.Parallel()

	query, args := buildPickPhraseIDsQuery(pickListQuery{AccountID: "ac_1", Search: newPickSearch(new("235"))})
	if strings.Contains(query, " UNION SELECT") || !strings.HasSuffix(query, ") LIMIT ?") {
		t.Errorf("phrase read does not stream to a limit:\n%s", query)
	}
	if args[len(args)-1] != pickPhraseScanCap+1 {
		t.Errorf("phrase read limit = %v, want %d", args[len(args)-1], pickPhraseScanCap+1)
	}

	for name, built := range map[string]func() (string, []any){
		"product lines": func() (string, []any) { return buildPickProductLineCountQuery([]string{"pl_1", "pl_2"}) },
		"prefix":        func() (string, []any) { return buildPickPrefixCountQuery("ac_1", "2%", 300) },
	} {
		query, args := built()
		if !strings.Contains(query, "LIMIT ?) capped") || strings.Contains(query, "DISTINCT") {
			t.Errorf("%s: count is not a capped read:\n%s", name, query)
		}
		if got := strings.Count(query, "?"); got != len(args) {
			t.Errorf("%s: %d placeholders but %d args", name, got, len(args))
		}
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
