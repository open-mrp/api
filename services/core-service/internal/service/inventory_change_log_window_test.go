package service

import (
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
)

func TestWithDefaultInventoryChangeLogWindow(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	explicitStart := now.Add(-400 * 24 * time.Hour)
	end := now.Add(-30 * 24 * time.Hour)
	ninetyDays := 90 * 24 * time.Hour

	cases := map[string]struct {
		params    domain.ListInventoryChangeLogsParams
		wantStart time.Time
	}{
		"no bounds reaches back ninety days from now":   {params: domain.ListInventoryChangeLogsParams{}, wantStart: now.Add(-ninetyDays)},
		"an end alone reaches back ninety days from it": {params: domain.ListInventoryChangeLogsParams{EndDate: &end}, wantStart: end.Add(-ninetyDays)},
		"an explicit start is the caller's choice":      {params: domain.ListInventoryChangeLogsParams{StartDate: &explicitStart}, wantStart: explicitStart},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			params := tc.params
			withDefaultInventoryChangeLogWindow(&params, now)
			if params.StartDate == nil || !params.StartDate.Equal(tc.wantStart) {
				t.Errorf("start = %v, want %v", params.StartDate, tc.wantStart)
			}
		})
	}
}

// An item's history is wanted whole, so a list for particular items gets no default window.
func TestWithDefaultInventoryChangeLogWindow_ItemListsAreWhole(t *testing.T) {
	params := domain.ListInventoryChangeLogsParams{ItemIDs: []string{"it_1"}}
	withDefaultInventoryChangeLogWindow(&params, time.Now())
	if params.StartDate != nil {
		t.Errorf("an item-scoped list gained a start: %v", params.StartDate)
	}
}
