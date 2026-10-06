package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPlanBuyerSummaryReads(t *testing.T) {
	month := func(m int) time.Time { return time.Date(2026, time.Month(m), 1, 0, 0, 0, 0, time.UTC) }
	lines := map[string]map[time.Time]int64{
		"small_1": {month(1): 1000},
		"small_2": {month(2): 1500},
		"small_3": {month(3): 900},
		"big":     {month(1): 2000, month(2): 1500, month(3): 1000, month(4): 2500, month(5): 100},
	}

	reads := planBuyerSummaryReads([]string{"small_1", "small_2", "unknown", "big", "small_3"}, lines)

	require.Equal(t, []buyerSummaryRead{
		// The big buyer's months split where the next would pass the budget; the outer windows stay open.
		{buyers: []string{"big"}, to: month(2)},
		{buyers: []string{"big"}, from: month(2), to: month(4)},
		{buyers: []string{"big"}, from: month(4)},
		{buyers: []string{"small_1", "small_2", "unknown"}},
		{buyers: []string{"small_3"}},
	}, reads)
}

func TestPlanBuyerSummaryReadsCapsBuyersWithoutCounts(t *testing.T) {
	ids := make([]string, salesBuyerSummaryMaxBuyers+1)
	for i := range ids {
		ids[i] = string(rune('a' + i))
	}
	reads := planBuyerSummaryReads(ids, nil)
	require.Len(t, reads, 2)
	require.Len(t, reads[0].buyers, salesBuyerSummaryMaxBuyers)
}
