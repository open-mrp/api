package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/apikit/pagination"
)

func TestMachBuildSearchParams_MatchesANamePrefixLiterally(t *testing.T) {
	t.Parallel()

	if got := machBuildSearchParams(new("")); got.Valid {
		t.Errorf("an empty term searched for %q", got.String)
	}
	if got, want := machBuildSearchParams(new(`1_%`)), `1\_\%%`; !got.Valid || got.String != want {
		t.Errorf("pattern = %q, want %q", got.String, want)
	}
}

func TestBuildMachineListQuery_EmitsOnlySuppliedFilters(t *testing.T) {
	t.Parallel()

	query, args := buildMachineListQuery(machineListFilter{accountID: "ac_1", limit: 26})
	assert.Equal(t, "SELECT "+machineListColumns+", 0 AS match_tier\nFROM machine m\nJOIN department d ON d.id = m.department_id"+
		"\nWHERE m.account_id = ?\nORDER BY m.created_at DESC, m.id DESC\nLIMIT ?", query)
	assert.Equal(t, []any{"ac_1", int32(26)}, args)

	archived := false
	at := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	query, args = buildMachineListQuery(machineListFilter{
		accountID: "ac_1", departmentIDs: []string{"dp_1", "dp_2"}, isActive: &archived,
		cursor: &pagination.StringCursor{Direction: pagination.DirectionBackward, OccurredAt: at, ID: "mch_9"}, limit: 11,
	})
	assert.Contains(t, query, "WHERE m.account_id = ?\nAND m.department_id IN (?,?)\nAND m.is_active = ?"+
		"\nAND (m.created_at > ? OR (m.created_at = ? AND m.id > ?))")
	assert.Contains(t, query, "ORDER BY m.created_at ASC, m.id ASC")
	assert.Equal(t, []any{"ac_1", "dp_1", "dp_2", false, at, at, "mch_9", int32(11)}, args)
}

// A search ranks the exact name first; the tier leads the cursor, and a cursor without a tier is from tier 0.
func TestBuildMachineListQuery_SearchRanksTheExactNameFirst(t *testing.T) {
	t.Parallel()

	query, args := buildMachineListQuery(machineListFilter{accountID: "ac_1", search: "Loom_1", limit: 26})
	assert.Contains(t, query, ", (m.name <> ?) AS match_tier\n")
	assert.Contains(t, query, "AND m.name LIKE ?\nORDER BY match_tier ASC, m.created_at DESC, m.id DESC")
	assert.Equal(t, []any{"Loom_1", "ac_1", `Loom\_1%`, int32(26)}, args)

	at := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	query, args = buildMachineListQuery(machineListFilter{
		accountID: "ac_1", search: "Loom",
		cursor: &pagination.StringCursor{Direction: pagination.DirectionForward, OccurredAt: at, ID: "mch_9"}, limit: 26,
	})
	assert.Contains(t, query, "AND ((m.name <> ?) > ? OR ((m.name <> ?) = ? AND (m.created_at < ? OR (m.created_at = ? AND m.id < ?))))")
	assert.Equal(t, []any{"Loom", "ac_1", "Loom%", "Loom", 0, "Loom", 0, at, at, "mch_9", int32(26)}, args)

	query, _ = buildMachineListQuery(machineListFilter{
		accountID: "ac_1", search: "Loom",
		cursor: &pagination.StringCursor{Direction: pagination.DirectionBackward, OccurredAt: at, ID: "mch_9", MatchTier: new(1)}, limit: 26,
	})
	assert.Contains(t, query, "ORDER BY match_tier DESC, m.created_at ASC, m.id ASC")
}

func TestBuildScanningStationListQuery_EmitsOnlySuppliedFilters(t *testing.T) {
	t.Parallel()

	query, args := buildScanningStationListQuery(scanningStationListFilter{accountID: "ac_1", limit: 26})
	assert.Equal(t, "SELECT "+scanningStationListColumns+"\nFROM scanning_station ss\nLEFT JOIN department d ON d.id = ss.department_id"+
		"\nWHERE ss.account_id = ?\nORDER BY ss.created_at DESC, ss.id DESC\nLIMIT ?", query)
	assert.Equal(t, []any{"ac_1", int32(26)}, args)

	query, args = buildScanningStationListQuery(scanningStationListFilter{
		accountID: "ac_1", search: "%pack%", departmentIDs: []string{"dp_1"}, limit: 11,
	})
	assert.Contains(t, query, "WHERE ss.account_id = ?\nAND ss.name LIKE ?\nAND ss.department_id IN (?)")
	assert.Equal(t, []any{"ac_1", "%pack%", "dp_1", int32(11)}, args)
}
