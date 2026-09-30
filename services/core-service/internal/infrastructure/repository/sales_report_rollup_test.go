package repository

import (
	"strings"
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/stretchr/testify/require"
)

func utc(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		panic(err)
	}
	return t.UTC()
}

func TestPlanWindow(t *testing.T) {
	tests := []struct {
		name       string
		start, end string
		grains     []string
		rollups    []rollupRange
		raws       []rawRange
	}{
		{
			name:   "whole UTC months are one month range",
			start:  "2026-01-01T00:00:00Z",
			end:    "2026-03-31T23:59:59.999Z",
			grains: breakdownGrains,
			rollups: []rollupRange{
				{grain: rollupGrainMonth, from: utc("2026-01-01T00:00:00Z"), to: utc("2026-04-01T00:00:00Z")},
			},
		},
		{
			name:   "a local month reads its partial UTC days from facts",
			start:  "2026-01-01T05:00:00Z",
			end:    "2026-02-01T04:59:59.999Z",
			grains: breakdownGrains,
			rollups: []rollupRange{
				{grain: rollupGrainDay, from: utc("2026-01-02T00:00:00Z"), to: utc("2026-02-01T00:00:00Z")},
			},
			raws: []rawRange{
				{from: utc("2026-01-01T05:00:00Z"), to: utc("2026-01-02T00:00:00Z")},
				{from: utc("2026-02-01T00:00:00Z"), to: utc("2026-02-01T04:59:59.999Z"), toInclusive: true},
			},
		},
		{
			name:   "hour buckets cover a whole-hour offset's edges",
			start:  "2026-01-01T05:00:00Z",
			end:    "2026-02-01T04:59:59.999Z",
			grains: totalGrains,
			rollups: []rollupRange{
				{grain: rollupGrainHour, from: utc("2026-01-01T05:00:00Z"), to: utc("2026-01-02T00:00:00Z")},
				{grain: rollupGrainDay, from: utc("2026-01-02T00:00:00Z"), to: utc("2026-02-01T00:00:00Z")},
				{grain: rollupGrainHour, from: utc("2026-02-01T00:00:00Z"), to: utc("2026-02-01T05:00:00Z")},
			},
		},
		{
			name:   "months, then days, then a partial day",
			start:  "2025-11-01T00:00:00Z",
			end:    "2026-01-10T12:30:00Z",
			grains: breakdownGrains,
			rollups: []rollupRange{
				{grain: rollupGrainMonth, from: utc("2025-11-01T00:00:00Z"), to: utc("2026-01-01T00:00:00Z")},
				{grain: rollupGrainDay, from: utc("2026-01-01T00:00:00Z"), to: utc("2026-01-10T00:00:00Z")},
			},
			raws: []rawRange{
				{from: utc("2026-01-10T00:00:00Z"), to: utc("2026-01-10T12:30:00Z"), toInclusive: true},
			},
		},
		{
			name:   "a half-hour offset starts with a raw half hour",
			start:  "2026-01-01T05:30:00Z",
			end:    "2026-01-01T08:00:00Z",
			grains: totalGrains,
			rollups: []rollupRange{
				{grain: rollupGrainHour, from: utc("2026-01-01T06:00:00Z"), to: utc("2026-01-01T08:00:00Z")},
			},
			raws: []rawRange{
				{from: utc("2026-01-01T05:30:00Z"), to: utc("2026-01-01T06:00:00Z")},
				{from: utc("2026-01-01T08:00:00Z"), to: utc("2026-01-01T08:00:00Z"), toInclusive: true},
			},
		},
		{
			name:   "a window inside one hour is read raw",
			start:  "2026-01-01T10:15:00Z",
			end:    "2026-01-01T10:45:00Z",
			grains: totalGrains,
			raws: []rawRange{
				{from: utc("2026-01-01T10:15:00Z"), to: utc("2026-01-01T10:45:00Z"), toInclusive: true},
			},
		},
		{
			name:   "a sub-millisecond end still admits the bucket it closes",
			start:  "2026-01-01T00:00:00Z",
			end:    "2026-01-01T23:59:59.999999Z",
			grains: breakdownGrains,
			rollups: []rollupRange{
				{grain: rollupGrainDay, from: utc("2026-01-01T00:00:00Z"), to: utc("2026-01-02T00:00:00Z")},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := planWindow(utc(tt.start), utc(tt.end), tt.grains)
			require.Equal(t, tt.rollups, plan.rollups)
			require.Equal(t, tt.raws, plan.raws)
		})
	}
}

func TestRollupScopeFor(t *testing.T) {
	buyers := &salesFactQuery{buyers: []string{"ac_1"}, buyersFiltered: true}
	none := &salesFactQuery{}
	tests := []struct {
		name      string
		filter    domain.SalesReportFilter
		q         *salesFactQuery
		dimension string
		ok        bool
		want      rollupScope
	}{
		{name: "no filters", q: none, dimension: rollupDimBuyer, ok: true, want: rollupScope{dimension: rollupDimBuyer}},
		{name: "sales reps are part of the key", filter: domain.SalesReportFilter{SalesRepIDs: []string{"au_1"}}, q: none, dimension: rollupDimItem, ok: true,
			want: rollupScope{dimension: rollupDimItem, salesRepIDs: []string{"au_1"}}},
		{name: "one product line has its own rows", filter: domain.SalesReportFilter{ProductLineIDs: []string{"pl_1"}}, q: none, dimension: rollupDimBuyer, ok: true,
			want: rollupScope{dimension: rollupDimBuyer, lineKey: "pl_1"}},
		{name: "several product lines add up their own rows", filter: domain.SalesReportFilter{ProductLineIDs: []string{"pl_1", "pl_2"}}, q: none, dimension: rollupDimBuyer, ok: true,
			want: rollupScope{dimension: rollupDimBuyer, lineKeys: []string{"pl_1", "pl_2"}}},
		{name: "several product lines with a sales rep", filter: domain.SalesReportFilter{ProductLineIDs: []string{"pl_1", "pl_2"}, SalesRepIDs: []string{"au_1"}}, q: none, dimension: rollupDimTotal, ok: true,
			want: rollupScope{dimension: rollupDimTotal, lineKeys: []string{"pl_1", "pl_2"}, salesRepIDs: []string{"au_1"}}},
		{name: "product lines filter the product line breakdown's groups", filter: domain.SalesReportFilter{ProductLineIDs: []string{"pl_1", "pl_2"}}, q: none, dimension: rollupDimProductLine, ok: true,
			want: rollupScope{dimension: rollupDimProductLine, dimensionIDs: []string{"pl_1", "pl_2"}}},
		{name: "customers filter the buyer breakdown's groups", q: buyers, dimension: rollupDimBuyer, ok: true,
			want: rollupScope{dimension: rollupDimBuyer, dimensionIDs: []string{"ac_1"}}},
		{name: "customers need lines for another dimension", q: buyers, dimension: rollupDimItem},
		{name: "items need lines for another dimension", filter: domain.SalesReportFilter{ItemIDs: []string{"it_1"}}, q: none, dimension: rollupDimTotal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := rollupScopeFor(tt.filter, tt.q, tt.dimension)
			require.Equal(t, tt.ok, ok)
			if ok {
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestAMultiLineScopeCountsInvoicesFromTheFactsOnly(t *testing.T) {
	q := &salesFactQuery{accountID: "ac_1"}
	q.add("f.account_id = ?", "ac_1")
	q.add("f.product_line_id IN (?, ?)", "pl_1", "pl_2")
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 3, 15, 12, 30, 0, 0, time.UTC)
	plan := planWindow(start, end, breakdownGrains)
	require.NotEmpty(t, plan.rollups)
	require.NotEmpty(t, plan.raws)

	sql, args := rollupPeriodRows(q, rollupScope{dimension: rollupDimBuyer, lineKeys: []string{"pl_1", "pl_2"}}, plan, start, end, "c", "f.buyer_account_id", "r.dimension_id")

	parts := strings.Split(sql, "\nUNION ALL\n")
	require.Len(t, parts, 3, "whole buckets, raw edges, and the invoice count")
	require.Contains(t, parts[0], "r.product_line_key IN (?,?)")
	require.Contains(t, parts[0], "0 AS ic", "summed per-line rows would count an invoice once per line it spans")
	require.Contains(t, parts[1], "0 AS ic")
	require.Contains(t, parts[2], "COUNT(DISTINCT f.invoice_id) AS ic")
	require.Contains(t, parts[2], "f.invoiced_at >= ? AND f.invoiced_at <= ?", "counted over the whole period")
	require.Equal(t, strings.Count(sql, "?"), len(args))
	require.Equal(t, []any{start, end}, args[len(args)-2:])

	one, _ := rollupPeriodRows(q, rollupScope{dimension: rollupDimBuyer, lineKey: "pl_1"}, plan, start, end, "c", "f.buyer_account_id", "r.dimension_id")
	require.Len(t, strings.Split(one, "\nUNION ALL\n"), 2, "one line's rows count its invoices exactly")
	require.Contains(t, one, "r.invoice_count AS ic")
}
