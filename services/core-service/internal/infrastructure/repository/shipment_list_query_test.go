package repository

import (
	gosql "database/sql"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/open-mrp/api/shared/pagination"
)

// The hint decides the plan, which is invisible from the results: these pin which list-order keys a
// walk may use (see shipmentListQuery.indexHint).
func TestBuildShipmentListQuery_IndexHintFollowsFilters(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		q    shipmentListQuery
		want string
	}{
		{"unfiltered reads in order", shipmentListQuery{}, shipmentCreatedIndex},
		// Never the created key beside a filter's: the planner may walk it from the far end instead.
		{"status offers its key", shipmentListQuery{Status: new("packed")}, shipmentStatusIndex},
		{"one customer offers the buyer key", shipmentListQuery{BuyerIDs: []string{"ac_c"}}, shipmentBuyerIndex},
		{"status and one customer offer both", shipmentListQuery{Status: new("packed"), BuyerIDs: []string{"ac_c"}}, shipmentStatusIndex + ", " + shipmentBuyerIndex},
		// A set's ranges of the buyer key come out of order, so walking it for the set would sort.
		{"a customer set walks the others", shipmentListQuery{BuyerIDs: []string{"ac_c1", "ac_c2"}}, shipmentCreatedIndex},
		{"a few customers read their ranges", shipmentListQuery{BuyerIDs: []string{"ac_c1", "ac_c2"}, Drive: shipmentDriveBuyers}, shipmentBuyerIndex},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.q.AccountID, tc.q.Limit = "ac_1", 26
			query, _ := buildShipmentListQuery(tc.q)
			if want := "FROM shipment s FORCE INDEX (" + tc.want + ")"; !strings.Contains(query, want) {
				t.Errorf("index hint is not %s:\n%s", tc.want, query)
			}
		})
	}
}

// A flipped comparison or ORDER BY silently repeats or skips rows at page boundaries.
func TestBuildShipmentListQuery_KeysetMatchesDirection(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		dir  pagination.Direction
		want []string
	}{
		{pagination.DirectionForward, []string{"(s.created_at < ? OR (s.created_at = ? AND s.id < ?))", "ORDER BY s.created_at DESC, s.id DESC LIMIT ?"}},
		{pagination.DirectionBackward, []string{"(s.created_at > ? OR (s.created_at = ? AND s.id > ?))", "ORDER BY s.created_at ASC, s.id ASC LIMIT ?"}},
	} {
		query, _ := buildShipmentListQuery(shipmentListQuery{
			AccountID: "ac_1", Direction: tc.dir, Limit: 26,
			CursorAt: gosql.NullTime{Valid: true}, CursorID: gosql.NullString{String: "sh_1", Valid: true},
		})
		for _, want := range tc.want {
			if !strings.Contains(query, want) {
				t.Errorf("%s: query lacks %q:\n%s", tc.dir, want, query)
			}
		}
	}
}

// Every filter together, under each drive: a placeholder/arg mismatch binds values to the wrong
// predicates, which fails loudly only when the count is off, not when the order is.
func TestBuildShipmentListQuery_ArgsBindInPlaceholderOrder(t *testing.T) {
	t.Parallel()

	start := gosql.NullTime{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true}
	end := gosql.NullTime{Time: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), Valid: true}
	cursor := gosql.NullTime{Time: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), Valid: true}
	base := shipmentListQuery{
		AccountID: "ac_1", Status: new("shipped"), Search: gosql.NullString{String: "%x%", Valid: true},
		BuyerIDs: []string{"ac_c1", "ac_c2"}, ItemIDs: []string{"it_1"}, ProductLineIDs: []string{"pl_1"},
		StartDate: start, EndDate: end, CursorAt: cursor, CursorID: gosql.NullString{String: "sh_9", Valid: true}, Limit: 26,
	}
	search := []any{"%x%", "%x%", "%x%", "%x%", "%x%", "%x%", "%x%"}
	tail := []any{start.Time, end.Time, cursor.Time, cursor.Time, "sh_9", int32(26)}

	for _, tc := range []struct {
		name  string
		drive shipmentDrive
		want  []any
		from  string
	}{
		{"walk", shipmentDriveListOrder,
			concat([]any{"ac_1", "shipped", "ac_c1", "ac_c2"}, search, []any{"it_1", "pl_1"}, tail), "FROM shipment s FORCE INDEX"},
		{"buyers", shipmentDriveBuyers,
			concat([]any{"ac_1", "shipped", "ac_c1", "ac_c2"}, search, []any{"it_1", "pl_1"}, tail), "FROM shipment s FORCE INDEX (" + shipmentBuyerIndex + ")"},
		{"items", shipmentDriveItems,
			concat([]any{"it_1", "ac_1", "shipped", "ac_c1", "ac_c2"}, search, []any{"pl_1"}, tail), ") matched JOIN shipment s ON s.id = matched.id"},
		{"product lines", shipmentDriveProductLines,
			concat([]any{"pl_1", "ac_1", "shipped", "ac_c1", "ac_c2"}, search, []any{"it_1"}, tail), ") matched JOIN shipment s ON s.id = matched.id"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			q := base
			q.Drive = tc.drive
			query, args := buildShipmentListQuery(q)
			if !reflect.DeepEqual(args, tc.want) {
				t.Errorf("args = %v\nwant %v", args, tc.want)
			}
			if got := strings.Count(query, "?"); got != len(args) {
				t.Errorf("%d placeholders but %d args", got, len(args))
			}
			if !strings.Contains(query, tc.from) {
				t.Errorf("query is not read from %q:\n%s", tc.from, query)
			}
		})
	}
}

// Each count stops at the cap, so sizing a filter costs at most shipmentMatchCap index entries.
func TestBuildShipmentMatchCountQuery_StopsAtTheCap(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		drive shipmentDrive
		want  []any
	}{
		{shipmentDriveBuyers, []any{"ac_1", "ac_c1", "ac_c2", shipmentMatchCap}},
		{shipmentDriveItems, []any{"ac_c1", "ac_c2", shipmentMatchCap}},
		{shipmentDriveProductLines, []any{"ac_c1", "ac_c2", shipmentMatchCap}},
	} {
		query, args := buildShipmentMatchCountQuery("ac_1", tc.drive, []string{"ac_c1", "ac_c2"})
		if !strings.Contains(query, "LIMIT ?) capped") || strings.Contains(query, "DISTINCT") {
			t.Errorf("count is not a capped read:\n%s", query)
		}
		if !reflect.DeepEqual(args, tc.want) {
			t.Errorf("args = %v, want %v", args, tc.want)
		}
		if got := strings.Count(query, "?"); got != len(args) {
			t.Errorf("%d placeholders but %d args", got, len(args))
		}
	}
}

// The hint names the index, so a rename or a dropped migration turns the query into a 1176 at runtime.
func TestShipmentListIndexes_AreDeclaredInMigrations(t *testing.T) {
	t.Parallel()

	schema := migrationsText(t)
	for _, index := range []string{shipmentCreatedIndex, shipmentStatusIndex, shipmentBuyerIndex} {
		if !strings.Contains(schema, index) {
			t.Errorf("%s is FORCE INDEX'd by the shipment list but no migration creates it", index)
		}
	}
}

func concat(parts ...[]any) []any {
	var out []any
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}
