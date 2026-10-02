package repository

import (
	gosql "database/sql"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/open-mrp/api/shared/pagination"
)

// Every filter together, under each drive: a placeholder/arg mismatch binds values to the wrong
// predicates, which fails loudly only when the count is off, not when the order is.
func TestBuildDeliveryListQuery_ArgsBindInPlaceholderOrder(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	cursor := gosql.NullTime{Time: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), Valid: true}
	base := deliveryListQuery{
		AccountID: "ac_1", Status: new("accepted"), Search: gosql.NullString{String: "%x%", Valid: true},
		SupplierIDs: []string{"ac_s1"}, ItemIDs: []string{"it_1"}, StartDate: &start, EndDate: &end,
		Direction: pagination.DirectionBackward, CursorAt: cursor, CursorID: gosql.NullString{String: "dv_9", Valid: true}, Limit: 26,
	}
	tail := []any{start, end, cursor.Time, cursor.Time, "dv_9", int32(26)}

	for _, tc := range []struct {
		name  string
		drive deliveryDrive
		want  []any
		from  string
	}{
		{"walk", deliveryDriveListOrder, concat([]any{"ac_1", "%x%", "%x%", "accepted", "it_1", "ac_s1"}, tail),
			"FROM delivery d FORCE INDEX (" + deliveryStatusIndex + ")"},
		{"suppliers", deliveryDriveSuppliers, concat([]any{"ac_s1", "ac_1", "%x%", "%x%", "accepted", "it_1"}, tail),
			") matched JOIN delivery d ON d.id = matched.id"},
		{"items", deliveryDriveItems, concat([]any{"it_1", "ac_1", "%x%", "%x%", "accepted", "ac_s1"}, tail),
			") matched JOIN delivery d ON d.id = matched.id"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			q := base
			q.Drive = tc.drive
			query, args := buildDeliveryListQuery(q)
			if !reflect.DeepEqual(args, tc.want) {
				t.Errorf("args = %v\nwant %v", args, tc.want)
			}
			if got := strings.Count(query, "?"); got != len(args) {
				t.Errorf("%d placeholders but %d args", got, len(args))
			}
			for _, want := range []string{tc.from, "(d.created_at > ? OR (d.created_at = ? AND d.id > ?))", "ORDER BY d.created_at ASC, d.id ASC LIMIT ?"} {
				if !strings.Contains(query, want) {
					t.Errorf("query lacks %q:\n%s", want, query)
				}
			}
		})
	}
}

func TestBuildDeliveryMatchCountQuery_StopsAtTheCap(t *testing.T) {
	t.Parallel()

	for _, drive := range []deliveryDrive{deliveryDriveSuppliers, deliveryDriveItems} {
		query, args := buildDeliveryMatchCountQuery(drive, []string{"a", "b"})
		if !strings.Contains(query, "LIMIT ?) capped") || strings.Contains(query, "DISTINCT") {
			t.Errorf("count is not a capped read:\n%s", query)
		}
		if !reflect.DeepEqual(args, []any{"a", "b", deliveryMatchCap}) {
			t.Errorf("args = %v", args)
		}
	}
}

func TestDeliveryListIndexes_AreDeclaredInMigrations(t *testing.T) {
	t.Parallel()

	schema := migrationsText(t)
	for _, index := range []string{deliveryCreatedIndex, deliveryStatusIndex} {
		if !strings.Contains(schema, index) {
			t.Errorf("%s is FORCE INDEX'd by the delivery list but no migration creates it", index)
		}
	}
}
