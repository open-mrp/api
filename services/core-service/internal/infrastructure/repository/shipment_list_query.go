package repository

import (
	gosql "database/sql"
	"strings"

	"github.com/open-mrp/api/shared/pagination"
)

// The shipment list is a deferred join: buildShipmentListQuery pages shipment ids from the shipment
// table, and GetShipmentsByIDs hydrates the page. Every customer filter (customer, group, sales rep)
// resolves to buyer_account_id, denormalized from the order, so an (account_id, [filter], created_at,
// id) key yields the account's shipments in list order and stops at the page.
const (
	shipmentCreatedIndex = "shipment_account_created_idx"
	shipmentStatusIndex  = "shipment_account_id_shipment_status_code_created_at_id_idx"
	shipmentBuyerIndex   = "shipment_account_buyer_created_idx"
)

// shipmentMatchCap is the most shipments an unordered filter may match and still drive the read: a
// set of customers (ranges of the buyer key), items or product lines (child tables). Reading every
// match and sorting is cheap for a few; with more, the filter matches often enough that walking a
// list-order key with it as a residual fills a page first.
const shipmentMatchCap = 2000

// shipmentDrive is where a page's candidate shipments come from.
type shipmentDrive int

const (
	// shipmentDriveListOrder walks a list-order key, every filter a residual.
	shipmentDriveListOrder shipmentDrive = iota
	// shipmentDriveBuyers reads the buyer set's ranges of the buyer key and sorts them.
	shipmentDriveBuyers
	// shipmentDriveItems and shipmentDriveProductLines join from the shipments whose lines match.
	shipmentDriveItems
	shipmentDriveProductLines
)

type shipmentListQuery struct {
	AccountID string
	Status    *string
	// Search is a LIKE pattern matched anywhere in the shipment's numbers and note, its order's number
	// and PO number, and its customer's name.
	Search gosql.NullString
	// BuyerIDs, when non-nil, limits the list to these customers: the customer filter intersected with
	// the customers of the group and sales-rep filters.
	BuyerIDs       []string
	ItemIDs        []string
	ProductLineIDs []string
	Drive          shipmentDrive
	StartDate      gosql.NullTime
	EndDate        gosql.NullTime
	Direction      pagination.Direction
	CursorAt       gosql.NullTime
	CursorID       gosql.NullString
	Limit          int32
}

// indexHint is the list-order keys a walk may use: the status key for a status, the buyer key for one
// customer, and the created key always. A set of customers walks only the others, since reading its
// ranges of the buyer key yields them out of order.
func (q shipmentListQuery) indexHint() []string {
	if q.Drive == shipmentDriveBuyers {
		return []string{shipmentBuyerIndex}
	}
	hint := []string{shipmentCreatedIndex}
	if q.Status != nil {
		hint = append(hint, shipmentStatusIndex)
	}
	if len(q.BuyerIDs) == 1 {
		hint = append(hint, shipmentBuyerIndex)
	}
	return hint
}

// buildShipmentListQuery returns the query for one page of shipment ids (Limit rows, newest first
// going forward) and its bind args. Only the predicates the caller set are emitted.
func buildShipmentListQuery(q shipmentListQuery) (string, []any) {
	args := make([]any, 0, 16+len(q.BuyerIDs)+len(q.ItemIDs)+len(q.ProductLineIDs))
	var b strings.Builder
	b.WriteString("SELECT STRAIGHT_JOIN s.id FROM ")
	switch q.Drive {
	case shipmentDriveItems:
		b.WriteString("(" + shipmentsWithItems(len(q.ItemIDs)) + ") matched JOIN shipment s ON s.id = matched.id")
		args = append(args, stringArgs(q.ItemIDs)...)
	case shipmentDriveProductLines:
		b.WriteString("(" + shipmentsWithProductLines(len(q.ProductLineIDs)) + ") matched JOIN shipment s ON s.id = matched.id")
		args = append(args, stringArgs(q.ProductLineIDs)...)
	default:
		b.WriteString("shipment s FORCE INDEX (" + strings.Join(q.indexHint(), ", ") + ")")
	}
	if q.Search.Valid {
		b.WriteString(" JOIN sales_order so ON so.id = s.sales_order_id JOIN account ba ON ba.id = so.buyer_account_id")
	}
	b.WriteString(" WHERE s.account_id = ?")
	args = append(args, q.AccountID)
	if q.Status != nil {
		b.WriteString(" AND s.shipment_status_code = ?")
		args = append(args, *q.Status)
	}
	if len(q.BuyerIDs) > 0 {
		b.WriteString(" AND s.buyer_account_id IN (" + placeholders(len(q.BuyerIDs)) + ")")
		args = append(args, stringArgs(q.BuyerIDs)...)
	}
	if q.Search.Valid {
		b.WriteString(" AND (s.number LIKE ? OR s.note LIKE ? OR s.bill_of_lading LIKE ? OR s.master_tracking_number LIKE ?" +
			" OR so.number LIKE ? OR ba.name LIKE ? OR so.customer_po_number LIKE ?)")
		for range 7 {
			args = append(args, q.Search.String)
		}
	}
	if len(q.ItemIDs) > 0 && q.Drive != shipmentDriveItems {
		b.WriteString(" AND EXISTS (SELECT 1 FROM shipment_line sl2 JOIN sales_order_line sol2 ON sol2.id = sl2.sales_order_line_id" +
			" WHERE sl2.shipment_id = s.id AND sol2.item_id IN (" + placeholders(len(q.ItemIDs)) + "))")
		args = append(args, stringArgs(q.ItemIDs)...)
	}
	if len(q.ProductLineIDs) > 0 && q.Drive != shipmentDriveProductLines {
		b.WriteString(" AND EXISTS (SELECT 1 FROM shipment_line sl3 JOIN sales_order_line sol3 ON sol3.id = sl3.sales_order_line_id" +
			" JOIN product prod3 ON prod3.id = sol3.product_id" +
			" WHERE sl3.shipment_id = s.id AND prod3.product_line_id IN (" + placeholders(len(q.ProductLineIDs)) + "))")
		args = append(args, stringArgs(q.ProductLineIDs)...)
	}
	if q.StartDate.Valid {
		b.WriteString(" AND s.created_at >= ?")
		args = append(args, q.StartDate.Time)
	}
	if q.EndDate.Valid {
		b.WriteString(" AND s.created_at <= ?")
		args = append(args, q.EndDate.Time)
	}
	order, cmp := "DESC", "<"
	if q.Direction == pagination.DirectionBackward {
		order, cmp = "ASC", ">"
	}
	if q.CursorAt.Valid {
		b.WriteString(" AND (s.created_at " + cmp + " ? OR (s.created_at = ? AND s.id " + cmp + " ?))")
		args = append(args, q.CursorAt.Time, q.CursorAt.Time, q.CursorID.String)
	}
	b.WriteString(" ORDER BY s.created_at " + order + ", s.id " + order + " LIMIT ?")
	args = append(args, q.Limit)
	return b.String(), args
}

// shipmentItemLines and shipmentProductLineLines are the shipment lines for any of n items or
// product lines, read from the order lines (and products) that name them.
func shipmentItemLines(n int) string {
	return "sales_order_line sol JOIN shipment_line sl ON sl.sales_order_line_id = sol.id" +
		" WHERE sol.item_id IN (" + placeholders(n) + ")"
}

func shipmentProductLineLines(n int) string {
	return "product prod JOIN sales_order_line sol ON sol.product_id = prod.id" +
		" JOIN shipment_line sl ON sl.sales_order_line_id = sol.id" +
		" WHERE prod.product_line_id IN (" + placeholders(n) + ")"
}

func shipmentsWithItems(n int) string {
	return "SELECT DISTINCT sl.shipment_id AS id FROM " + shipmentItemLines(n)
}

func shipmentsWithProductLines(n int) string {
	return "SELECT DISTINCT sl.shipment_id AS id FROM " + shipmentProductLineLines(n)
}

// buildShipmentMatchCountQuery counts what one unordered filter matches, stopping at
// shipmentMatchCap: the buyer set's shipments, or the item or product-line filter's shipment lines.
// Lines bound their shipments from above, and counting them stops at the cap where collecting
// distinct shipments would read every match first.
func buildShipmentMatchCountQuery(accountID string, drive shipmentDrive, ids []string) (string, []any) {
	args := make([]any, 0, len(ids)+2)
	var inner string
	switch drive {
	case shipmentDriveBuyers:
		args = append(args, accountID)
		inner = "shipment FORCE INDEX (" + shipmentBuyerIndex + ")" +
			" WHERE account_id = ? AND buyer_account_id IN (" + placeholders(len(ids)) + ")"
	case shipmentDriveItems:
		inner = shipmentItemLines(len(ids))
	case shipmentDriveProductLines:
		inner = shipmentProductLineLines(len(ids))
	}
	args = append(args, stringArgs(ids)...)
	args = append(args, shipmentMatchCap)
	return "SELECT COUNT(*) FROM (SELECT 1 FROM " + inner + " LIMIT ?) capped", args
}
