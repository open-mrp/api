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
// index hint exists for — see buildPickShipByListQuery.
func TestBuildPickShipByListQuery_DrivingIndexFollowsStatusFilter(t *testing.T) {
	t.Parallel()

	open := "open"
	closed := "closed"

	for _, tc := range []struct {
		name   string
		status *string
		want   string
	}{
		// Open picks are a tiny slice of a mostly-closed table, so the filter has to lead the index or
		// the scan reads the account's closed history before a page fills.
		{"open pins the filter", &open, pickOpenShipByIndex},
		{"closed reads in order", &closed, pickShipByIndex},
		{"unfiltered reads in order", nil, pickShipByIndex},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			query, _ := buildPickShipByListQuery(
				"ac_1", gosql.NullString{}, tc.status, nil, nil, nil,
				gosql.NullTime{}, gosql.NullTime{}, pagination.DirectionForward,
				gosql.NullTime{}, gosql.NullString{}, 51,
			)

			if want := " FROM pick p FORCE INDEX (" + tc.want + ")"; !strings.Contains(query, want) {
				t.Errorf("driving index is not %s:\n%s", tc.want, query)
			}
		})
	}
}

// The hint names the index, so a rename or a dropped migration turns the query into a 1176 at runtime
// rather than a compile error.
func TestPickShipByIndexes_AreDeclaredInMigrations(t *testing.T) {
	t.Parallel()

	schema := migrationsText(t)
	for _, index := range []string{pickShipByIndex, pickOpenShipByIndex} {
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
