package repository

import (
	"context"
	"database/sql"
	"strings"

	"github.com/open-mrp/apikit/db/backfill"
	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// InventoryMovementTypesBackfillName is the movement-type backfill's row in backfill_progress.
const InventoryMovementTypesBackfillName = "inventory_movement_types"

// InventoryMovementTypesBackfill stores the type and actor of inventory movements written before movements stored
// them (00115), walking inventory_change_log by id. Readers derive both the same way until then, so the backfill only
// makes the types[], actor_ids[] and actor_types[] filters see old rows. It pages from read (a replica) and writes on
// write (the primary), one UPDATE per distinct type and actor in the page, and only onto rows still without a type,
// so a re-run batch changes nothing.
type InventoryMovementTypesBackfill struct {
	read  *sqlc.Queries
	write *sql.DB
}

// NewInventoryMovementTypesBackfill builds the backfill over a replica and the primary.
func NewInventoryMovementTypesBackfill(read *sqlc.Queries, write *sql.DB) *InventoryMovementTypesBackfill {
	return &InventoryMovementTypesBackfill{read: read, write: write}
}

type movementTypeKey struct {
	typeCode, actorType string
	actorID             sql.NullString
}

// Batch is a backfill.Batch over inventory_change_log ids.
func (b *InventoryMovementTypesBackfill) Batch(ctx context.Context, m *backfill.Meter, cursor string, limit int) (string, int, bool, error) {
	groups := map[movementTypeKey][]any{}
	var order []movementTypeKey
	last, n := cursor, 0
	if err := m.Time(func() error {
		rows, err := b.read.DB().QueryContext(ctx,
			"SELECT icl.id, icl.action_type_code, icl.scanning_station_id, icl.responsible_user_id, icl.type_code IS NULL, COALESCE(q.value, '0') "+
				"FROM inventory_change_log icl LEFT JOIN quantity q ON q.id = icl.quantity_id "+
				"WHERE icl.id > ? ORDER BY icl.id LIMIT ?", cursor, limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id, actionTypeCode, value string
			var stationID, userID sql.NullString
			var untyped bool
			if err := rows.Scan(&id, &actionTypeCode, &stationID, &userID, &untyped, &value); err != nil {
				return err
			}
			last, n = id, n+1
			if !untyped {
				continue
			}
			measure, _ := decimal.NewFromString(value)
			a := domain.LegacyMovementActor(nullStringPtr(stationID), nullStringPtr(userID))
			actorType, actorID := a.Columns()
			key := movementTypeKey{typeCode: string(domain.LegacyMovementType(actionTypeCode, measure)), actorType: actorType, actorID: actorID}
			if _, seen := groups[key]; !seen {
				order = append(order, key)
			}
			groups[key] = append(groups[key], id)
		}
		return rows.Err()
	}); err != nil {
		return cursor, 0, false, err
	}
	if n == 0 {
		return cursor, 0, true, nil
	}
	for _, key := range order {
		ids := groups[key]
		args := append([]any{key.typeCode, key.actorType, key.actorID}, ids...)
		if err := m.Time(func() error {
			_, err := b.write.ExecContext(ctx,
				"UPDATE inventory_change_log SET type_code = ?, actor_type = ?, actor_id = ? WHERE id IN ("+
					strings.TrimSuffix(strings.Repeat("?, ", len(ids)), ", ")+") AND type_code IS NULL", args...)
			return err
		}); err != nil {
			return cursor, 0, false, err
		}
	}
	return last, n, n < limit, nil
}
