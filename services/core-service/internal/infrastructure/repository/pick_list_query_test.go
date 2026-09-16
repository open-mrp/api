package repository

import (
	gosql "database/sql"
	"os"
	"strings"
	"testing"

	"github.com/open-mrp/api/shared/pagination"
)

// The pick list's plan is invisible from its results: every index returns the same page, so a wrong
// one shows up only as latency on the one account large enough to feel it. These pin the property the
// index hint exists for — see buildPickListQuery.
func TestBuildPickListQuery_DrivingIndexFollowsSortAndStatusFilter(t *testing.T) {
	t.Parallel()

	open := "open"
	closed := "closed"

	for _, tc := range []struct {
		name         string
		sortByShipBy bool
		status       *string
		want         string
	}{
		// Open picks are a tiny slice of a mostly-closed table, so the filter has to lead the index or
		// the scan reads the account's closed history before a page fills.
		{"ship-by open pins the filter", true, &open, pickOpenShipByIndex},
		{"ship-by closed reads in order", true, &closed, pickShipByIndex},
		{"ship-by unfiltered reads in order", true, nil, pickShipByIndex},
		{"created open pins the filter", false, &open, pickOpenCreatedIndex},
		{"created closed reads in order", false, &closed, pickCreatedIndex},
		{"created unfiltered reads in order", false, nil, pickCreatedIndex},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			query, _ := buildPickListQuery(
				tc.sortByShipBy, "ac_1", gosql.NullString{}, tc.status, nil, nil, nil,
				gosql.NullTime{}, gosql.NullTime{}, pagination.DirectionForward,
				gosql.NullTime{}, gosql.NullString{}, 51,
			)

			if want := " FROM pick p FORCE INDEX (" + tc.want + ")"; !strings.Contains(query, want) {
				t.Errorf("driving index is not %s:\n%s", tc.want, query)
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

			query, args := buildPickListQuery(
				tc.sortByShipBy, "ac_1", gosql.NullString{}, nil, nil, nil, nil,
				gosql.NullTime{}, gosql.NullTime{}, tc.dir, cursorAt, cursorID, 51,
			)

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

// The hint names the index, so a rename or a dropped migration turns the query into a 1176 at runtime
// rather than a compile error.
func TestPickListIndexes_AreDeclaredInMigrations(t *testing.T) {
	t.Parallel()

	schema := migrationsText(t)
	for _, index := range []string{pickShipByIndex, pickOpenShipByIndex, pickCreatedIndex, pickOpenCreatedIndex} {
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
