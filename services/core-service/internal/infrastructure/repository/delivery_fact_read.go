package repository

import (
	"context"
	gosql "database/sql"
	"strings"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/scheduling"
)

// deliveryFactsReady reports whether delivery_order_fact has been filled: until its first reconcile pass
// completes, the facts are partial and the report reads the live tables instead.
func (r *productionScheduleInputRepoImpl) deliveryFactsReady(ctx context.Context) (bool, error) {
	sync, apiErr := NewDeliveryFactRepo(r.queries).GetSync(ctx)
	if apiErr != nil {
		return false, apiErr
	}
	return sync.LastCompletedAt != nil, nil
}

// deliveryFactWhere is the window and filters over delivery_order_fact (f). The window is a range of the
// clustered key, so the read is the account's committed orders in that range and nothing else.
func deliveryFactWhere(accountID string, start, end time.Time, args deliveryFilterArgs) (string, []any) {
	where := []string{"f.owner_account_id = ?", "f.ship_by_date >= ?", "f.ship_by_date <= ?",
		"f.sales_order_status_code IN ('issued', 'fulfilled')"}
	params := []any{accountID, start, end}
	in := func(column string, values []any) {
		where = append(where, column+" IN ("+placeholders(len(values))+")")
		params = append(params, values...)
	}
	if args.includeSalesRep {
		in("f.sales_rep_id", nullStringArgs(args.salesRepIDs))
	}
	if args.includeProductLine {
		in("f.product_line_key", nullStringArgs(args.productLineIDs))
	}
	if args.includeBuyer {
		in("f.buyer_account_id", stringsToAny(args.buyerIDs))
	}
	return "\nWHERE " + strings.Join(where, " AND "), params
}

func nullStringArgs(values []gosql.NullString) []any {
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = v.String
	}
	return out
}

// listDeliveryOutcomesFromFacts reads the window from delivery_order_fact: the orders summed over their
// product lines, then the product lines each order spans. Customer names and groups are read once per
// buyer after, rather than joined on every order.
func (r *productionScheduleInputRepoImpl) listDeliveryOutcomesFromFacts(ctx context.Context, accountID string, start, end time.Time, args deliveryFilterArgs) ([]scheduling.DeliveryOutcome, error) {
	q := r.queries.DB()
	where, params := deliveryFactWhere(accountID, start, end, args)

	rows, err := q.QueryContext(ctx, `SELECT f.sales_order_id, f.sales_order_number, f.buyer_account_id, f.sales_rep_id,
    f.ship_by_date, f.issued_at, f.first_ship_at, f.lead_time_days, f.lead_time_source_code,
    CAST(SUM(f.quantity_ordered) AS DECIMAL(65,30)), CAST(SUM(f.quantity_packed) AS DECIMAL(65,30))
FROM delivery_order_fact f`+where+`
GROUP BY f.ship_by_date, f.sales_order_id, f.sales_order_number, f.buyer_account_id, f.sales_rep_id,
    f.issued_at, f.first_ship_at, f.lead_time_days, f.lead_time_source_code
ORDER BY f.ship_by_date, f.sales_order_id`, params...)
	if err != nil {
		return nil, err
	}
	var (
		out    []scheduling.DeliveryOutcome
		buyers []string
		seen   = map[string]bool{}
	)
	for rows.Next() {
		var (
			o                 scheduling.DeliveryOutcome
			rep, source       gosql.NullString
			issued, firstShip gosql.NullTime
			leadTime          gosql.NullInt32
			ordered, packed   string
		)
		if err := rows.Scan(&o.SalesOrderID, &o.SalesOrderNumber, &o.BuyerAccountID, &rep, &o.ShipByDate, &issued, &firstShip,
			&leadTime, &source, &ordered, &packed); err != nil {
			_ = rows.Close()
			return nil, err
		}
		o.SalesRepID, o.CommitmentSource = nullSQLString(rep), nullSQLString(source)
		o.QuantityOrdered, o.QuantityPacked = decimalToFloat64(ordered), decimalToFloat64(packed)
		if issued.Valid {
			o.IssuedAt = &issued.Time
		}
		if firstShip.Valid {
			o.FirstShipAt = &firstShip.Time
		}
		if leadTime.Valid {
			o.CommittedLeadTimeDays = int(leadTime.Int32)
		}
		if !seen[o.BuyerAccountID] {
			seen[o.BuyerAccountID] = true
			buyers = append(buyers, o.BuyerAccountID)
		}
		out = append(out, o)
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return []scheduling.DeliveryOutcome{}, nil
	}

	productLines, err := r.deliveryFactProductLines(ctx, where, params)
	if err != nil {
		return nil, err
	}
	customers, err := r.deliveryCustomers(ctx, accountID, buyers)
	if err != nil {
		return nil, err
	}
	for i := range out {
		o := &out[i]
		o.ProductLines = productLines[o.SalesOrderID]
		if c, ok := customers[o.BuyerAccountID]; ok {
			o.CustomerName, o.CustomerGroupID, o.CustomerGroupName = c.name, c.groupID, c.groupName
		}
	}
	return out, nil
}

// deliveryFactProductLines maps each order in the window onto the named product lines its sale lines are
// in, under the same filters. A line with no product line, or one whose product line is gone, names none.
func (r *productionScheduleInputRepoImpl) deliveryFactProductLines(ctx context.Context, where string, params []any) (map[string][]scheduling.ProductLineRef, error) {
	rows, err := r.queries.DB().QueryContext(ctx, `SELECT f.sales_order_id, f.product_line_key, pl.name
FROM delivery_order_fact f
JOIN product_line pl ON pl.id = f.product_line_key`+where+`
ORDER BY f.sales_order_id, f.product_line_key`, params...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[string][]scheduling.ProductLineRef{}
	for rows.Next() {
		var orderID, lineID, name string
		if err := rows.Scan(&orderID, &lineID, &name); err != nil {
			return nil, err
		}
		out[orderID] = append(out[orderID], scheduling.ProductLineRef{ID: lineID, Name: name})
	}
	return out, rows.Err()
}

type deliveryCustomer struct {
	name, groupID, groupName string
}

// deliveryCustomers reads each buyer's name and, through the account's customer relation, its group.
func (r *productionScheduleInputRepoImpl) deliveryCustomers(ctx context.Context, accountID string, buyers []string) (map[string]deliveryCustomer, error) {
	out := make(map[string]deliveryCustomer, len(buyers))
	for start := 0; start < len(buyers); start += deliveryFactIDBatch {
		batch := buyers[start:min(start+deliveryFactIDBatch, len(buyers))]
		rows, err := r.queries.DB().QueryContext(ctx, `SELECT a.id, a.name, ar.account_group_id, ag.name
FROM account a
LEFT JOIN account_relation ar ON ar.owner_account_id = ? AND ar.counterparty_account_id = a.id
    AND ar.account_relation_role_code = 'customer'
LEFT JOIN account_group ag ON ag.id = ar.account_group_id
WHERE a.id IN (`+placeholders(len(batch))+`)`, append([]any{accountID}, stringsToAny(batch)...)...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var (
				id                 string
				name               gosql.NullString
				groupID, groupName gosql.NullString
			)
			if err := rows.Scan(&id, &name, &groupID, &groupName); err != nil {
				_ = rows.Close()
				return nil, err
			}
			out[id] = deliveryCustomer{name: nullSQLString(name), groupID: nullSQLString(groupID), groupName: nullSQLString(groupName)}
		}
		_ = rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return out, nil
}
