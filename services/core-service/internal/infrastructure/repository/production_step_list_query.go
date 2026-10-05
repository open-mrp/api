package repository

import (
	"strings"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/db"
)

// stepListCursor is where a page of production steps starts: after (forward, newest first) or before
// (backward) the step at createdAt and id.
type stepListCursor struct {
	createdAt time.Time
	id        string
	backward  bool
}

// buildStepListPageQuery selects one page of production step IDs in list order. Only the predicates the
// caller supplied are emitted: an `(? IS NULL OR ...)` guard is not sargable, and its cost falls on every
// list, filtered or not. The rows are read after, by ID (ListProductionStepsByIDs).
//
// A search requires every word to begin a word of the name. Words the FULLTEXT index holds go to MATCH;
// shorter ones go to a regular expression, which then reads only the index's matches.
func buildStepListPageQuery(params domain.ListProductionStepsParams, cursor *stepListCursor, limit int32) (string, []any) {
	var where []string
	var args []any
	add := func(predicate string, values ...any) {
		where = append(where, predicate)
		args = append(args, values...)
	}

	add("ps.account_id = ?", params.AccountID)

	fulltext, shortWords := db.AllWordsSearch(params.Query)
	if fulltext.Valid {
		add("MATCH(ps.name) AGAINST(? IN BOOLEAN MODE)", fulltext.String)
	}
	if shortWords.Valid {
		add("REGEXP_LIKE(ps.name, ?, 'i')", shortWords.String)
	}

	if len(params.ItemIDs) > 0 {
		in := stepListPlaceholders(len(params.ItemIDs))
		add("(p.item_id IN ("+in+") OR EXISTS (SELECT 1 FROM consumption c WHERE c.production_step_id = ps.id AND c.item_id IN ("+in+")))",
			append(stepListArgs(params.ItemIDs), stepListArgs(params.ItemIDs)...)...)
	}
	if len(params.MachineIDs) > 0 {
		add("EXISTS (SELECT 1 FROM machine m WHERE m.production_step_id = ps.id AND m.id IN ("+stepListPlaceholders(len(params.MachineIDs))+"))",
			stepListArgs(params.MachineIDs)...)
	}
	if len(params.ScanningStationIDs) > 0 {
		add("ps.scanning_station_id IN ("+stepListPlaceholders(len(params.ScanningStationIDs))+")",
			stepListArgs(params.ScanningStationIDs)...)
	}
	// The edge table's A is the downstream step and B the upstream one.
	if len(params.InputStepIDs) > 0 {
		add("EXISTS (SELECT 1 FROM _parent_child_production_steps pcps WHERE pcps.A = ps.id AND pcps.B IN ("+stepListPlaceholders(len(params.InputStepIDs))+"))",
			stepListArgs(params.InputStepIDs)...)
	}
	if len(params.OutputStepIDs) > 0 {
		add("EXISTS (SELECT 1 FROM _parent_child_production_steps pcps WHERE pcps.B = ps.id AND pcps.A IN ("+stepListPlaceholders(len(params.OutputStepIDs))+"))",
			stepListArgs(params.OutputStepIDs)...)
	}
	if params.StartDate != nil {
		add("ps.created_at >= ?", *params.StartDate)
	}
	if params.EndDate != nil {
		add("ps.created_at <= ?", *params.EndDate)
	}

	order := " ORDER BY ps.created_at DESC, ps.id DESC"
	if cursor != nil {
		if cursor.backward {
			add("(ps.created_at > ? OR (ps.created_at = ? AND ps.id > ?))", cursor.createdAt, cursor.createdAt, cursor.id)
			order = " ORDER BY ps.created_at ASC, ps.id ASC"
		} else {
			add("(ps.created_at < ? OR (ps.created_at = ? AND ps.id < ?))", cursor.createdAt, cursor.createdAt, cursor.id)
		}
	}

	// A step is listed only with its production, as the list's rows join it.
	query := "SELECT ps.id FROM production_step ps JOIN production p ON p.production_step_id = ps.id WHERE " +
		strings.Join(where, " AND ") + order + " LIMIT ?"
	return query, append(args, limit)
}

func stepListPlaceholders(n int) string {
	return strings.Repeat("?, ", n-1) + "?"
}

func stepListArgs(values []string) []any {
	args := make([]any, len(values))
	for i, v := range values {
		args[i] = v
	}
	return args
}
