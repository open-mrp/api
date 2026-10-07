package repository

import (
	"context"
	"database/sql"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
)

// selectedProductionParts expands the selected items and product lines to the parts their production consumes, as the dashboard's open-batches and production-costs reports did: every part consumed upstream, recursively, plus any selected item no step produces, plus the selected items themselves. Nothing selected, or a selection that expands to nothing, filters nothing.
func selectedProductionParts(ctx context.Context, q sqlc.DBTX, accountID string, itemIDs, productLineIDs []string) ([]string, *apierror.APIError) {
	if len(itemIDs) == 0 && len(productLineIDs) == 0 {
		return nil, nil
	}
	selected := append([]string{}, itemIDs...)
	if len(productLineIDs) > 0 {
		lineItems, apiErr := queryStringColumn(ctx, q,
			`SELECT p.item_id FROM product p JOIN item i ON i.id = p.item_id
WHERE i.account_id = ? AND p.product_line_id IN (`+placeholders(len(productLineIDs))+`)`,
			append([]any{accountID}, stringsToAny(productLineIDs)...)...)
		if apiErr != nil {
			return nil, apiErr
		}
		selected = append(selected, lineItems...)
	}

	parts, apiErr := consumedParts(ctx, q, accountID, selected)
	if apiErr != nil {
		return nil, apiErr
	}
	return dedupe(append(parts, itemIDs...)), nil
}

// consumedParts walks production upstream from items: an item no step produces is a part in its own right, and a step that produces one contributes the parts it consumes, which are walked in turn.
func consumedParts(ctx context.Context, q sqlc.DBTX, accountID string, items []string) ([]string, *apierror.APIError) {
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

		rows, err := q.QueryContext(ctx, `SELECT p.item_id, c.item_id
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
