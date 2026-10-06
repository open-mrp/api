package repository

import (
	"context"
	"database/sql"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

// openBatchSummaryQuery totals the open, scanned batches waiting at each scanning station per item. Each counts for its quantity less what has already gone downstream into output batches, summed as recorded, without unit conversion, as the dashboard summed them; the unit is the earliest-scanned batch's.
func openBatchSummaryQuery(batchIndex, accountID string, itemIDs []string) (string, []any) {
	preds := []string{"b.account_id = ?", "b.closed_at IS NULL", "b.scanned_at IS NOT NULL", "b.scanning_station_id IS NOT NULL"}
	args := []any{accountID}
	if len(itemIDs) > 0 {
		preds = append(preds, "b.item_id IN ("+placeholders(len(itemIDs))+")")
		args = append(args, stringsToAny(itemIDs)...)
	}
	query := `SELECT g.scanning_station_id, g.item_id, COALESCE(d.name, ''), i.sku, g.total_count, g.unit_abbreviation
FROM (
    SELECT
        b.scanning_station_id,
        b.item_id,
        CAST(SUM(q.value - COALESCE((
            -- A batch's outputs are the downstream (A) side of _batch_flow rows where it is the upstream (B) batch. Correlated per batch: a grouped derived table cannot take the account filter.
            SELECT SUM(oq.value)
            FROM _batch_flow bf
            JOIN batch ob ON ob.id = bf.A
            JOIN quantity oq ON oq.id = ob.quantity_id
            WHERE bf.B = b.id
        ), 0)) AS DECIMAL(65,30)) AS total_count,
        SUBSTRING_INDEX(GROUP_CONCAT(qu.abbreviation ORDER BY b.scanned_at, b.id SEPARATOR '\n'), '\n', 1) AS unit_abbreviation
    FROM batch b FORCE INDEX (` + batchIndex + `)
    JOIN quantity q ON q.id = b.quantity_id
    JOIN unit qu ON qu.id = q.unit_id
    WHERE ` + strings.Join(preds, " AND ") + `
    GROUP BY b.scanning_station_id, b.item_id
) g
JOIN item i ON i.id = g.item_id
LEFT JOIN scanning_station ss ON ss.id = g.scanning_station_id
LEFT JOIN department d ON d.id = ss.department_id
ORDER BY d.name ASC, g.scanning_station_id ASC, g.total_count DESC, g.item_id ASC`
	return query, args
}

func (r *batchRepoImpl) FindOpenBatches(ctx context.Context, accountID string, itemIDs, productLineIDs []string) ([]domain.OpenBatchSummary, *apierror.APIError) {
	ctx, span := batchRepoTracer.Start(ctx, "repository.batch.find_open_batches")
	defer span.End()

	parts, apiErr := r.openBatchItemFilter(ctx, accountID, itemIDs, productLineIDs)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// The batches are read by the account's open scanned ones or by the items', whichever reaches fewer.
	keys := []keyRange{{
		index: "batch_account_closed_scanned_idx",
		where: "kso.account_id = ? AND kso.closed_at IS NULL AND kso.scanned_at IS NOT NULL",
		args:  []any{accountID},
	}}
	if len(parts) > 0 {
		keys = append(keys, keyRange{index: "batch_item_id_idx", where: "kso.item_id IN (" + placeholders(len(parts)) + ")", args: stringsToAny(parts)})
	}
	batchIndex, apiErr := cheapestKey(ctx, r.queries.DB(), "batch", keys)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	query, args := openBatchSummaryQuery(batchIndex, accountID, parts)
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	defer rows.Close()

	summaries := []domain.OpenBatchSummary{}
	for rows.Next() {
		var (
			s     domain.OpenBatchSummary
			count sql.NullString
		)
		if err := rows.Scan(&s.ScanningStationID, &s.ItemID, &s.DepartmentName, &s.ItemName, &count, &s.Unit); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		s.Count = decimal.Zero
		if count.Valid {
			s.Count, err = decimal.NewFromString(count.String)
			if err != nil {
				return nil, tracing.Trace(span, apierror.NewInternalError(err, "Invalid open batch count."))
			}
		}
		summaries = append(summaries, s)
	}
	if err := rows.Err(); err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	return summaries, nil
}

// openBatchItemFilter expands the selected items and product lines to the parts their production consumes, as the dashboard's open-batches view did: every part consumed upstream, recursively, plus any selected item no step produces, plus the selected items themselves. Nothing selected, or a selection that expands to nothing, filters nothing.
func (r *batchRepoImpl) openBatchItemFilter(ctx context.Context, accountID string, itemIDs, productLineIDs []string) ([]string, *apierror.APIError) {
	if len(itemIDs) == 0 && len(productLineIDs) == 0 {
		return nil, nil
	}
	selected := append([]string{}, itemIDs...)
	if len(productLineIDs) > 0 {
		lineItems, apiErr := queryStringColumn(ctx, r.queries.DB(),
			`SELECT p.item_id FROM product p JOIN item i ON i.id = p.item_id
WHERE i.account_id = ? AND p.product_line_id IN (`+placeholders(len(productLineIDs))+`)`,
			append([]any{accountID}, stringsToAny(productLineIDs)...)...)
		if apiErr != nil {
			return nil, apiErr
		}
		selected = append(selected, lineItems...)
	}

	parts, apiErr := r.consumedParts(ctx, accountID, selected)
	if apiErr != nil {
		return nil, apiErr
	}
	return dedupe(append(parts, itemIDs...)), nil
}

// consumedParts walks production upstream from items: an item no step produces is a part in its own right, and a step that produces one contributes the parts it consumes, which are walked in turn.
func (r *batchRepoImpl) consumedParts(ctx context.Context, accountID string, items []string) ([]string, *apierror.APIError) {
	visited := map[string]bool{}
	var parts []string
	frontier := items
	for len(frontier) > 0 {
		var next []string
		for _, id := range frontier {
			if !visited[id] {
				visited[id] = true
				next = append(next, id)
			}
		}
		if len(next) == 0 {
			break
		}

		rows, err := r.queries.DB().QueryContext(ctx, `SELECT p.item_id, c.item_id
FROM production p
JOIN production_step ps ON ps.id = p.production_step_id
LEFT JOIN consumption c ON c.production_step_id = ps.id
    AND EXISTS (SELECT 1 FROM item ci WHERE ci.id = c.item_id AND ci.item_type_code = 'part')
WHERE ps.account_id = ? AND p.item_id IN (`+placeholders(len(next))+`)`,
			append([]any{accountID}, stringsToAny(next)...)...)
		if err != nil {
			return nil, db.MapSQLError(err)
		}
		produced := map[string]bool{}
		var consumed []string
		for rows.Next() {
			var producedID string
			var consumedID sql.NullString
			if err := rows.Scan(&producedID, &consumedID); err != nil {
				_ = rows.Close()
				return nil, db.MapSQLError(err)
			}
			produced[producedID] = true
			if consumedID.Valid {
				consumed = append(consumed, consumedID.String)
			}
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, db.MapSQLError(err)
		}
		_ = rows.Close()

		for _, id := range next {
			if !produced[id] {
				parts = append(parts, id)
			}
		}
		parts = append(parts, consumed...)
		frontier = consumed
	}
	return parts, nil
}
