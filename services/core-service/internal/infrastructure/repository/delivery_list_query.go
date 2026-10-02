package repository

import (
	gosql "database/sql"
	"strings"
	"time"

	"github.com/open-mrp/api/shared/pagination"
)

// The delivery list is a deferred join: buildDeliveryListQuery pages delivery ids from the delivery
// table, and GetDeliveriesByIDs hydrates the page with its order and line count.
const (
	deliveryCreatedIndex = "delivery_account_created_idx"
	deliveryStatusIndex  = "delivery_account_id_delivery_status_code_created_at_id_idx"
)

// deliveryMatchCap is the most deliveries an unordered filter may match and still drive the read: the
// suppliers' purchase orders, or items (child tables). Reading every match and sorting is cheap for a
// few; with more, walking a list-order key with the filter as a residual fills a page first.
const deliveryMatchCap = 2000

// deliveryDrive is where a page's candidate deliveries come from.
type deliveryDrive int

const (
	// deliveryDriveListOrder walks a list-order key, every filter a residual.
	deliveryDriveListOrder deliveryDrive = iota
	// deliveryDriveSuppliers and deliveryDriveItems join from the deliveries the filter matches.
	deliveryDriveSuppliers
	deliveryDriveItems
)

type deliveryListQuery struct {
	AccountID string
	Status    *string
	// Search is a LIKE pattern matched anywhere in the delivery's number or its order's.
	Search      gosql.NullString
	SupplierIDs []string
	ItemIDs     []string
	Drive       deliveryDrive
	StartDate   *time.Time
	EndDate     *time.Time
	Direction   pagination.Direction
	CursorAt    gosql.NullTime
	CursorID    gosql.NullString
	Limit       int32
}

func (q deliveryListQuery) indexHint() []string {
	// Never the created key beside the status key: the planner may swap to it for the order and walk it
	// from the account's far end rather than range the status from the cursor.
	if q.Status != nil {
		return []string{deliveryStatusIndex}
	}
	return []string{deliveryCreatedIndex}
}

// buildDeliveryListQuery returns the query for one page of delivery ids (Limit rows, newest first
// going forward) and its bind args. Only the predicates the caller set are emitted.
func buildDeliveryListQuery(q deliveryListQuery) (string, []any) {
	args := make([]any, 0, 12+len(q.SupplierIDs)+len(q.ItemIDs))
	var b strings.Builder
	b.WriteString("SELECT STRAIGHT_JOIN d.id FROM ")
	switch q.Drive {
	case deliveryDriveSuppliers:
		b.WriteString("(" + deliveriesFromSuppliers(len(q.SupplierIDs)) + ") matched JOIN delivery d ON d.id = matched.id")
		args = append(args, stringArgs(q.SupplierIDs)...)
	case deliveryDriveItems:
		b.WriteString("(" + deliveriesWithItems(len(q.ItemIDs)) + ") matched JOIN delivery d ON d.id = matched.id")
		args = append(args, stringArgs(q.ItemIDs)...)
	default:
		b.WriteString("delivery d FORCE INDEX (" + strings.Join(q.indexHint(), ", ") + ")")
	}
	if q.Search.Valid || len(q.SupplierIDs) > 0 {
		b.WriteString(" JOIN sales_order so ON so.id = d.sales_order_id")
	}
	b.WriteString(" WHERE d.account_id = ?")
	args = append(args, q.AccountID)
	if !q.Search.Valid && len(q.SupplierIDs) == 0 {
		// GetDeliveriesByIDs inner-joins the order: a delivery whose order is gone would take a page slot
		// and then be dropped, leaving the page short.
		b.WriteString(" AND EXISTS (SELECT 1 FROM sales_order so0 WHERE so0.id = d.sales_order_id)")
	}
	if q.Search.Valid {
		b.WriteString(" AND (d.number LIKE ? OR so.number LIKE ?)")
		args = append(args, q.Search.String, q.Search.String)
	}
	if q.Status != nil {
		b.WriteString(" AND d.delivery_status_code = ?")
		args = append(args, *q.Status)
	}
	if len(q.ItemIDs) > 0 && q.Drive != deliveryDriveItems {
		b.WriteString(" AND EXISTS (SELECT 1 FROM delivery_line dl2" +
			" JOIN receiving_order_line rol2 ON rol2.id = dl2.receiving_order_line_id" +
			" JOIN sales_order_line sol2 ON sol2.id = rol2.sales_order_line_id" +
			" WHERE dl2.delivery_id = d.id AND sol2.item_id IN (" + placeholders(len(q.ItemIDs)) + "))")
		args = append(args, stringArgs(q.ItemIDs)...)
	}
	if len(q.SupplierIDs) > 0 && q.Drive != deliveryDriveSuppliers {
		b.WriteString(" AND so.seller_account_id IN (" + placeholders(len(q.SupplierIDs)) + ")")
		args = append(args, stringArgs(q.SupplierIDs)...)
	}
	if q.StartDate != nil {
		b.WriteString(" AND d.created_at >= ?")
		args = append(args, *q.StartDate)
	}
	if q.EndDate != nil {
		b.WriteString(" AND d.created_at <= ?")
		args = append(args, *q.EndDate)
	}
	order, cmp := "DESC", "<"
	if q.Direction == pagination.DirectionBackward {
		order, cmp = "ASC", ">"
	}
	if q.CursorAt.Valid {
		b.WriteString(" AND (d.created_at " + cmp + " ? OR (d.created_at = ? AND d.id " + cmp + " ?))")
		args = append(args, q.CursorAt.Time, q.CursorAt.Time, q.CursorID.String)
	}
	b.WriteString(" ORDER BY d.created_at " + order + ", d.id " + order + " LIMIT ?")
	args = append(args, q.Limit)
	return b.String(), args
}

// deliverySupplierRows and deliveryItemRows are the rows a supplier or item filter matches: the
// suppliers' orders' deliveries, or the items' delivery lines.
func deliverySupplierRows(n int) string {
	return "sales_order so2 JOIN delivery d2 ON d2.sales_order_id = so2.id" +
		" WHERE so2.seller_account_id IN (" + placeholders(n) + ")"
}

func deliveryItemRows(n int) string {
	return "sales_order_line sol3 JOIN receiving_order_line rol3 ON rol3.sales_order_line_id = sol3.id" +
		" JOIN delivery_line dl3 ON dl3.receiving_order_line_id = rol3.id" +
		" WHERE sol3.item_id IN (" + placeholders(n) + ")"
}

func deliveriesFromSuppliers(n int) string {
	return "SELECT d2.id FROM " + deliverySupplierRows(n)
}

func deliveriesWithItems(n int) string {
	return "SELECT DISTINCT dl3.delivery_id AS id FROM " + deliveryItemRows(n)
}

// buildDeliveryMatchCountQuery counts the rows one unordered filter matches, stopping at
// deliveryMatchCap. An item's delivery lines bound its deliveries from above, and counting them stops
// at the cap where collecting distinct deliveries would read every match first.
func buildDeliveryMatchCountQuery(drive deliveryDrive, ids []string) (string, []any) {
	rows := deliverySupplierRows(len(ids))
	if drive == deliveryDriveItems {
		rows = deliveryItemRows(len(ids))
	}
	args := append(stringArgs(ids), deliveryMatchCap)
	return "SELECT COUNT(*) FROM (SELECT 1 FROM " + rows + " LIMIT ?) capped", args
}
