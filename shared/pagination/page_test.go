package pagination

import (
	"testing"
	"time"
)

type strItem struct {
	id         string
	occurredAt time.Time
	tier       int32
}

func getOccurredAt(i strItem) time.Time { return i.occurredAt }
func getStrID(i strItem) string         { return i.id }
func getMatchTier(i strItem) int32      { return i.tier }

// dataset is in the DESC order a first-page query returns: newest first, id breaking timestamp ties.
func dataset() []strItem {
	base := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	return []strItem{
		{id: "it_g", occurredAt: base, tier: 0},
		{id: "it_f", occurredAt: base.Add(-1 * time.Hour), tier: 0},
		{id: "it_e", occurredAt: base.Add(-2 * time.Hour), tier: 1},
		{id: "it_d", occurredAt: base.Add(-2 * time.Hour), tier: 1},
		{id: "it_c", occurredAt: base.Add(-3 * time.Hour), tier: 2},
		{id: "it_b", occurredAt: base.Add(-4 * time.Hour), tier: 2},
		{id: "it_a", occurredAt: base.Add(-5 * time.Hour), tier: 2},
	}
}

// beforeInDesc reports whether a sorts ahead of b under the keyset order every list query uses.
func beforeInDesc(a, b strItem) bool {
	if !a.occurredAt.Equal(b.occurredAt) {
		return a.occurredAt.After(b.occurredAt)
	}
	return a.id > b.id
}

// queryPage stands in for the repository keyset query the builders are paired with: rows strictly past
// the cursor, DESC for forward and ASC for backward, always limit+1 so the builder can detect an extra.
func queryPage(all []strItem, c *StringCursor, limit int32) []strItem {
	if c == nil {
		return takeN(all, limit+1)
	}

	at := strItem{id: c.ID, occurredAt: c.OccurredAt}

	var rows []strItem
	if c.Direction == DirectionForward {
		for _, x := range all {
			if beforeInDesc(at, x) {
				rows = append(rows, x)
			}
		}
	} else {
		for i := len(all) - 1; i >= 0; i-- {
			if beforeInDesc(all[i], at) {
				rows = append(rows, all[i])
			}
		}
	}
	return takeN(rows, limit+1)
}

func takeN(s []strItem, n int32) []strItem {
	if int(n) < len(s) {
		return append([]strItem(nil), s[:n]...)
	}
	return append([]strItem(nil), s...)
}

func ids(items []strItem) []string {
	out := make([]string, 0, len(items))
	for _, i := range items {
		out = append(out, i.id)
	}
	return out
}

func equalIDs(got []strItem, want []string) bool {
	g := ids(got)
	if len(g) != len(want) {
		return false
	}
	for i := range g {
		if g[i] != want[i] {
			return false
		}
	}
	return true
}

func mustDecodeString(t *testing.T, c *string) StringCursor {
	t.Helper()
	if c == nil {
		t.Fatal("expected a non-nil cursor")
	}
	decoded, err := DecodeStringCursor(*c)
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}
	return decoded
}

func TestBuildPageString_FirstPage_HasMore(t *testing.T) {
	t.Parallel()
	items, pi := BuildPageString(takeN(dataset(), 4), 3, nil, getOccurredAt, getStrID)

	if !equalIDs(items, []string{"it_g", "it_f", "it_e"}) {
		t.Errorf("items = %v, want [it_g it_f it_e]", ids(items))
	}
	if !pi.HasNextPage {
		t.Error("expected HasNextPage=true")
	}
	if pi.HasPrevPage {
		t.Error("expected HasPrevPage=false")
	}
	if pi.PrevCursor != nil {
		t.Error("expected nil PrevCursor on the first page")
	}

	next := mustDecodeString(t, pi.NextCursor)
	if next.ID != "it_e" {
		t.Errorf("NextCursor ID = %q, want it_e", next.ID)
	}
	if next.Direction != DirectionForward {
		t.Errorf("NextCursor direction = %q, want %q", next.Direction, DirectionForward)
	}
	if !next.OccurredAt.Equal(items[2].occurredAt) {
		t.Errorf("NextCursor OccurredAt = %v, want %v", next.OccurredAt, items[2].occurredAt)
	}
	if next.MatchTier != nil {
		t.Errorf("NextCursor MatchTier = %v, want nil", *next.MatchTier)
	}
}

func TestBuildPageString_FirstPage_NoMore(t *testing.T) {
	t.Parallel()
	items, pi := BuildPageString(takeN(dataset(), 2), 3, nil, getOccurredAt, getStrID)

	if len(items) != 2 {
		t.Errorf("items = %v, want 2", ids(items))
	}
	if pi.HasNextPage || pi.HasPrevPage {
		t.Errorf("expected a single page, got %+v", pi)
	}
	if pi.NextCursor != nil || pi.PrevCursor != nil {
		t.Error("expected nil cursors")
	}
}

func TestBuildPageString_ForwardCursor_MiddlePage(t *testing.T) {
	t.Parallel()
	dir := DirectionForward
	items, pi := BuildPageString(dataset()[3:], 3, &dir, getOccurredAt, getStrID)

	if !equalIDs(items, []string{"it_d", "it_c", "it_b"}) {
		t.Errorf("items = %v, want [it_d it_c it_b]", ids(items))
	}
	if !pi.HasNextPage || !pi.HasPrevPage {
		t.Errorf("expected both neighbors, got %+v", pi)
	}

	next := mustDecodeString(t, pi.NextCursor)
	if next.ID != "it_b" || next.Direction != DirectionForward {
		t.Errorf("NextCursor = %q/%q, want it_b/%q", next.ID, next.Direction, DirectionForward)
	}
	prev := mustDecodeString(t, pi.PrevCursor)
	if prev.ID != "it_d" || prev.Direction != DirectionBackward {
		t.Errorf("PrevCursor = %q/%q, want it_d/%q", prev.ID, prev.Direction, DirectionBackward)
	}
}

func TestBuildPageString_ForwardCursor_LastPage(t *testing.T) {
	t.Parallel()
	dir := DirectionForward
	items, pi := BuildPageString(dataset()[6:], 3, &dir, getOccurredAt, getStrID)

	if !equalIDs(items, []string{"it_a"}) {
		t.Errorf("items = %v, want [it_a]", ids(items))
	}
	if pi.HasNextPage {
		t.Error("expected HasNextPage=false")
	}
	if pi.NextCursor != nil {
		t.Error("expected nil NextCursor")
	}
	if !pi.HasPrevPage {
		t.Error("expected HasPrevPage=true behind a forward cursor")
	}
	if prev := mustDecodeString(t, pi.PrevCursor); prev.ID != "it_a" || prev.Direction != DirectionBackward {
		t.Errorf("PrevCursor = %q/%q, want it_a/%q", prev.ID, prev.Direction, DirectionBackward)
	}
}

func TestBuildPageString_BackwardCursor_Reverses(t *testing.T) {
	t.Parallel()
	// A backward query returns ASC rows: [it_e it_f it_g] plus one extra proving a page exists behind.
	rows := []strItem{
		{id: "it_e", occurredAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)},
		{id: "it_f", occurredAt: time.Date(2026, 4, 1, 11, 0, 0, 0, time.UTC)},
		{id: "it_g", occurredAt: time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)},
		{id: "it_h", occurredAt: time.Date(2026, 4, 1, 13, 0, 0, 0, time.UTC)},
	}

	dir := DirectionBackward
	items, pi := BuildPageString(rows, 3, &dir, getOccurredAt, getStrID)

	if !equalIDs(items, []string{"it_g", "it_f", "it_e"}) {
		t.Errorf("items = %v, want [it_g it_f it_e] (reversed to DESC)", ids(items))
	}
	if !pi.HasNextPage {
		t.Error("expected HasNextPage=true: a backward page always came from a later one")
	}
	if !pi.HasPrevPage {
		t.Error("expected HasPrevPage=true")
	}

	// Cursors must be taken after the reversal, or a client walking back lands on the wrong boundary.
	if next := mustDecodeString(t, pi.NextCursor); next.ID != "it_e" {
		t.Errorf("NextCursor ID = %q, want it_e", next.ID)
	}
	if prev := mustDecodeString(t, pi.PrevCursor); prev.ID != "it_g" {
		t.Errorf("PrevCursor ID = %q, want it_g", prev.ID)
	}
}

func TestBuildPageString_BackwardCursor_FirstPageReached(t *testing.T) {
	t.Parallel()
	rows := []strItem{
		{id: "it_f", occurredAt: time.Date(2026, 4, 1, 11, 0, 0, 0, time.UTC)},
		{id: "it_g", occurredAt: time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)},
	}

	dir := DirectionBackward
	items, pi := BuildPageString(rows, 3, &dir, getOccurredAt, getStrID)

	if !equalIDs(items, []string{"it_g", "it_f"}) {
		t.Errorf("items = %v, want [it_g it_f]", ids(items))
	}
	if pi.HasPrevPage {
		t.Error("expected HasPrevPage=false")
	}
	if pi.PrevCursor != nil {
		t.Error("expected nil PrevCursor")
	}
	if !pi.HasNextPage || pi.NextCursor == nil {
		t.Errorf("expected a forward cursor back to where we came from, got %+v", pi)
	}
}

// TestBuildPageString_WalkPagesWithoutGapOrDuplicate is the regression that page-boundary bugs hide in:
// a trim or a reversal that is off by one drops or repeats a row only at the seam between pages.
func TestBuildPageString_WalkPagesWithoutGapOrDuplicate(t *testing.T) {
	t.Parallel()
	all := dataset()
	const limit int32 = 3

	var (
		seen  []string
		pages [][]string
		cur   *StringCursor
	)
	for range len(all) {
		var dir *Direction
		if cur != nil {
			dir = &cur.Direction
		}
		items, pi := BuildPageString(queryPage(all, cur, limit), limit, dir, getOccurredAt, getStrID)

		seen = append(seen, ids(items)...)
		pages = append(pages, ids(items))
		if !pi.HasNextPage {
			break
		}
		next := mustDecodeString(t, pi.NextCursor)
		cur = &next
	}

	want := ids(all)
	if len(seen) != len(want) {
		t.Fatalf("walked %v (%d rows) across %v, want all %d rows exactly once", seen, len(seen), pages, len(want))
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("walked %v, want %v", seen, want)
		}
	}
}

// TestBuildPageString_BackwardWalkReturnsTheSamePage pins the reversal: paging back from page two must
// reproduce page one exactly, in the same order, not a window shifted by one row.
func TestBuildPageString_BackwardWalkReturnsTheSamePage(t *testing.T) {
	t.Parallel()
	all := dataset()
	const limit int32 = 3

	first, firstPI := BuildPageString(queryPage(all, nil, limit), limit, nil, getOccurredAt, getStrID)
	next := mustDecodeString(t, firstPI.NextCursor)

	second, secondPI := BuildPageString(queryPage(all, &next, limit), limit, &next.Direction, getOccurredAt, getStrID)
	if !equalIDs(second, []string{"it_d", "it_c", "it_b"}) {
		t.Fatalf("second page = %v, want [it_d it_c it_b]", ids(second))
	}

	back := mustDecodeString(t, secondPI.PrevCursor)
	if back.Direction != DirectionBackward {
		t.Fatalf("PrevCursor direction = %q, want %q", back.Direction, DirectionBackward)
	}

	reReadFirst, backPI := BuildPageString(queryPage(all, &back, limit), limit, &back.Direction, getOccurredAt, getStrID)
	if !equalIDs(reReadFirst, ids(first)) {
		t.Errorf("paging back gave %v, want the original first page %v", ids(reReadFirst), ids(first))
	}
	if backPI.HasPrevPage {
		t.Error("expected HasPrevPage=false at the start of the list")
	}
}

func TestBuildPageStringWithSearchRank_CarriesMatchTier(t *testing.T) {
	t.Parallel()
	dir := DirectionForward
	items, pi := BuildPageStringWithSearchRank(dataset()[2:], 3, &dir, true, getOccurredAt, getStrID, getMatchTier)

	if !equalIDs(items, []string{"it_e", "it_d", "it_c"}) {
		t.Fatalf("items = %v, want [it_e it_d it_c]", ids(items))
	}

	next := mustDecodeString(t, pi.NextCursor)
	if next.ID != "it_c" {
		t.Errorf("NextCursor ID = %q, want it_c", next.ID)
	}
	if next.MatchTier == nil || *next.MatchTier != 2 {
		t.Errorf("NextCursor MatchTier = %v, want the last row's tier 2", next.MatchTier)
	}

	prev := mustDecodeString(t, pi.PrevCursor)
	if prev.ID != "it_e" {
		t.Errorf("PrevCursor ID = %q, want it_e", prev.ID)
	}
	if prev.MatchTier == nil || *prev.MatchTier != 1 {
		t.Errorf("PrevCursor MatchTier = %v, want the first row's tier 1", prev.MatchTier)
	}
}

// TestBuildPageStringWithSearchRank_OmitsMatchTierWhenDisabled guards the unranked path: a tier baked
// into a plain list cursor would re-enter the ranked branch of the query on the next page.
func TestBuildPageStringWithSearchRank_OmitsMatchTierWhenDisabled(t *testing.T) {
	t.Parallel()
	dir := DirectionForward
	_, pi := BuildPageStringWithSearchRank(dataset()[2:], 3, &dir, false, getOccurredAt, getStrID, getMatchTier)

	if next := mustDecodeString(t, pi.NextCursor); next.MatchTier != nil {
		t.Errorf("NextCursor MatchTier = %d, want nil", *next.MatchTier)
	}
	if prev := mustDecodeString(t, pi.PrevCursor); prev.MatchTier != nil {
		t.Errorf("PrevCursor MatchTier = %d, want nil", *prev.MatchTier)
	}
}

// TestBuildPageStringWithSearchRank_BackwardCursor_ReversesBeforeTiering catches tiers read off the
// pre-reversal boundaries, which would send the ranked query back into the wrong tier.
func TestBuildPageStringWithSearchRank_BackwardCursor_ReversesBeforeTiering(t *testing.T) {
	t.Parallel()
	rows := []strItem{
		{id: "it_c", occurredAt: time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC), tier: 2},
		{id: "it_e", occurredAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC), tier: 1},
		{id: "it_g", occurredAt: time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC), tier: 0},
		{id: "it_h", occurredAt: time.Date(2026, 4, 1, 13, 0, 0, 0, time.UTC), tier: 0},
	}

	dir := DirectionBackward
	items, pi := BuildPageStringWithSearchRank(rows, 3, &dir, true, getOccurredAt, getStrID, getMatchTier)

	if !equalIDs(items, []string{"it_g", "it_e", "it_c"}) {
		t.Fatalf("items = %v, want [it_g it_e it_c]", ids(items))
	}

	next := mustDecodeString(t, pi.NextCursor)
	if next.ID != "it_c" || next.MatchTier == nil || *next.MatchTier != 2 {
		t.Errorf("NextCursor = %q tier %v, want it_c tier 2", next.ID, next.MatchTier)
	}
	prev := mustDecodeString(t, pi.PrevCursor)
	if prev.ID != "it_g" || prev.MatchTier == nil || *prev.MatchTier != 0 {
		t.Errorf("PrevCursor = %q tier %v, want it_g tier 0", prev.ID, prev.MatchTier)
	}
}
