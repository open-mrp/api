package repository

import (
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
		{name: "several product lines would double count invoices", filter: domain.SalesReportFilter{ProductLineIDs: []string{"pl_1", "pl_2"}}, q: none, dimension: rollupDimBuyer},
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
