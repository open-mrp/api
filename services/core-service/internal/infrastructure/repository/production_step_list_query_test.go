package repository

import (
	"strings"
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestBuildStepListPageQuery_EmitsOnlySuppliedPredicates(t *testing.T) {
	query, args := buildStepListPageQuery(domain.ListProductionStepsParams{AccountID: "ac_1"}, nil, 11)

	assert.Equal(t, "SELECT ps.id FROM production_step ps JOIN production p ON p.production_step_id = ps.id"+
		" WHERE ps.account_id = ? ORDER BY ps.created_at DESC, ps.id DESC LIMIT ?", query)
	assert.Equal(t, []any{"ac_1", int32(11)}, args)
	assert.NotContains(t, query, " OR ", "an unfiltered list carries no optional guards")
}

func TestBuildStepListPageQuery_SearchSplitsShortWordsFromTheIndex(t *testing.T) {
	q := "QA init P1"
	query, args := buildStepListPageQuery(domain.ListProductionStepsParams{AccountID: "ac_1", Query: &q}, nil, 11)

	assert.Contains(t, query, "AND MATCH(ps.name) AGAINST(? IN BOOLEAN MODE) AND REGEXP_LIKE(ps.name, ?, 'i')",
		"the pattern follows the MATCH, so it reads only the index's matches")
	assert.Equal(t, []any{"ac_1", "+init*", `^(?=.*\bQA)(?=.*\bP1)`, int32(11)}, args)

	short := "QA"
	query, args = buildStepListPageQuery(domain.ListProductionStepsParams{AccountID: "ac_1", Query: &short}, nil, 11)
	assert.NotContains(t, query, "MATCH", "no word for the index means no MATCH")
	assert.Equal(t, []any{"ac_1", `^(?=.*\bQA)`, int32(11)}, args)
}

func TestBuildStepListPageQuery_Filters(t *testing.T) {
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	query, args := buildStepListPageQuery(domain.ListProductionStepsParams{
		AccountID:          "ac_1",
		ItemIDs:            []string{"it_1", "it_2"},
		MachineIDs:         []string{"mc_1"},
		ScanningStationIDs: []string{"ss_1"},
		InputStepIDs:       []string{"ps_up"},
		OutputStepIDs:      []string{"ps_down"},
		StartDate:          &start,
		EndDate:            &end,
	}, nil, 11)

	for _, want := range []string{
		"(p.item_id IN (?, ?) OR EXISTS (SELECT 1 FROM consumption c WHERE c.production_step_id = ps.id AND c.item_id IN (?, ?)))",
		"EXISTS (SELECT 1 FROM machine m WHERE m.production_step_id = ps.id AND m.id IN (?))",
		"ps.scanning_station_id IN (?)",
		"EXISTS (SELECT 1 FROM _parent_child_production_steps pcps WHERE pcps.A = ps.id AND pcps.B IN (?))",
		"EXISTS (SELECT 1 FROM _parent_child_production_steps pcps WHERE pcps.B = ps.id AND pcps.A IN (?))",
		"ps.created_at >= ?",
		"ps.created_at <= ?",
	} {
		assert.Contains(t, query, want)
	}
	assert.Equal(t, []any{"ac_1", "it_1", "it_2", "it_1", "it_2", "mc_1", "ss_1", "ps_up", "ps_down", start, end, int32(11)}, args)
	assert.Equal(t, strings.Count(query, "?"), len(args))
}

func TestBuildStepListPageQuery_Cursors(t *testing.T) {
	at := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)

	query, args := buildStepListPageQuery(domain.ListProductionStepsParams{AccountID: "ac_1"},
		&stepListCursor{createdAt: at, id: "ps_9"}, 11)
	assert.Contains(t, query, "(ps.created_at < ? OR (ps.created_at = ? AND ps.id < ?)) ORDER BY ps.created_at DESC, ps.id DESC")
	assert.Equal(t, []any{"ac_1", at, at, "ps_9", int32(11)}, args)

	query, _ = buildStepListPageQuery(domain.ListProductionStepsParams{AccountID: "ac_1"},
		&stepListCursor{createdAt: at, id: "ps_9", backward: true}, 11)
	assert.Contains(t, query, "(ps.created_at > ? OR (ps.created_at = ? AND ps.id > ?)) ORDER BY ps.created_at ASC, ps.id ASC")
}
