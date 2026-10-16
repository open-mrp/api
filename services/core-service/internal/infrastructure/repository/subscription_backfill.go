package repository

import (
	"context"
	"database/sql"
	"strings"

	"github.com/open-mrp/apikit/db/backfill"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/id"
)

// SubscriptionBackfillName is the subscription backfill's row in backfill_progress.
const SubscriptionBackfillName = "record_subscriptions"

// SubscriptionBackfill subscribes the creators and responsible people of orders and production runs made before
// subscriptions existed (00119): an order's sales rep and a run's responsible user as assignee, and whoever created
// either (from its create audit event) as creator. It walks sales_order, then production_run, by id, reading from a
// replica and inserting on the primary with INSERT IGNORE, so a person who already has a row for the record (or
// unsubscribed) keeps it and a re-run batch changes nothing. The cursor is "<table>:<last id>".
type SubscriptionBackfill struct {
	read  *sqlc.Queries
	write *sql.DB
}

// NewSubscriptionBackfill builds the backfill over a replica and the primary.
func NewSubscriptionBackfill(read *sqlc.Queries, write *sql.DB) *SubscriptionBackfill {
	return &SubscriptionBackfill{read: read, write: write}
}

type subscriptionBackfillTable struct {
	name string
	// records selects id, account_id, record type and the assignee account user of up to limit records after id.
	records string
	// auditType is the records' audit resource type.
	auditType constants.ObjectType
}

var subscriptionBackfillTables = []subscriptionBackfillTable{
	{
		name: "sales_order",
		records: "SELECT id, owner_account_id, sales_order_type_code, sales_rep_id FROM sales_order " +
			"WHERE id > ? ORDER BY id LIMIT ?",
		auditType: constants.ObjectTypeSalesOrder,
	},
	{
		name: "production_run",
		records: "SELECT id, account_id, 'production_run', responsible_user_id FROM production_run " +
			"WHERE id > ? ORDER BY id LIMIT ?",
		auditType: constants.ObjectTypeProductionRun,
	},
}

type backfillSubscription struct {
	accountID, accountUserID, recordType, recordID, reason string
}

// Batch is a backfill.Batch over sales_order then production_run ids.
func (b *SubscriptionBackfill) Batch(ctx context.Context, m *backfill.Meter, cursor string, limit int) (string, int, bool, error) {
	table, after := subscriptionBackfillTables[0].name, ""
	if name, last, ok := strings.Cut(cursor, ":"); ok {
		table, after = name, last
	}
	idx := -1
	for i, t := range subscriptionBackfillTables {
		if t.name == table {
			idx = i
		}
	}
	if idx < 0 {
		return cursor, 0, true, nil
	}
	t := subscriptionBackfillTables[idx]

	type record struct{ id, accountID, recordType string }
	var records []record
	var subs []backfillSubscription
	if err := m.Time(func() error {
		rows, err := b.read.DB().QueryContext(ctx, t.records, after, limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var r record
			var assignee sql.NullString
			if err := rows.Scan(&r.id, &r.accountID, &r.recordType, &assignee); err != nil {
				return err
			}
			records = append(records, r)
			if assignee.Valid && assignee.String != "" {
				subs = append(subs, backfillSubscription{r.accountID, assignee.String, r.recordType, r.id, string(constants.SubscriptionReasonAssignee)})
			}
		}
		return rows.Err()
	}); err != nil {
		return cursor, 0, false, err
	}
	if len(records) == 0 {
		if idx+1 < len(subscriptionBackfillTables) {
			return subscriptionBackfillTables[idx+1].name + ":", 0, false, nil
		}
		return cursor, 0, true, nil
	}

	typeOf := make(map[string]string, len(records))
	args := []any{string(t.auditType)}
	for _, r := range records {
		typeOf[r.id] = r.recordType
		args = append(args, r.id)
	}
	if err := m.Time(func() error {
		rows, err := b.read.DB().QueryContext(ctx,
			"SELECT ae.resource_id, ae.account_id, au.id FROM audit_event ae "+
				"JOIN account_user au ON au.user_id = ae.actor_id AND au.account_id = ae.account_id "+
				"WHERE ae.resource_type = ? AND ae.resource_id IN ("+strings.TrimSuffix(strings.Repeat("?, ", len(records)), ", ")+") "+
				"AND ae.action = 'create' AND ae.identity_type = 'user'", args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var recordID, accountID, accountUserID string
			if err := rows.Scan(&recordID, &accountID, &accountUserID); err != nil {
				return err
			}
			subs = append(subs, backfillSubscription{accountID, accountUserID, typeOf[recordID], recordID, string(constants.SubscriptionReasonCreator)})
		}
		return rows.Err()
	}); err != nil {
		return cursor, 0, false, err
	}

	if len(subs) > 0 {
		values := make([]any, 0, len(subs)*6)
		for _, s := range subs {
			subID, apiErr := id.GenID(id.SubscriptionIDPrefix, nil)
			if apiErr != nil {
				return cursor, 0, false, apiErr
			}
			values = append(values, subID, s.accountID, s.accountUserID, s.recordType, s.recordID, s.reason)
		}
		if err := m.Time(func() error {
			_, err := b.write.ExecContext(ctx,
				"INSERT IGNORE INTO subscription (id, account_id, account_user_id, record_type, record_id, reason, status, created_at, updated_at) VALUES "+
					strings.TrimSuffix(strings.Repeat("(?, ?, ?, ?, ?, ?, 'subscribed', NOW(3), NOW(3)), ", len(subs)), ", "), values...)
			return err
		}); err != nil {
			return cursor, 0, false, err
		}
	}

	next := t.name + ":" + records[len(records)-1].id
	if len(records) < limit && idx+1 < len(subscriptionBackfillTables) {
		return subscriptionBackfillTables[idx+1].name + ":", len(records), false, nil
	}
	return next, len(records), len(records) < limit, nil
}
