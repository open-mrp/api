package domain

import "time"

// DeliveryOrderFact is one committed sales order's sale lines in one product line, summed: a row of
// delivery_order_fact. The order's own fields repeat on each of its rows.
type DeliveryOrderFact struct {
	OwnerAccountID string
	ShipByDate     time.Time
	SalesOrderID   string
	// ProductLineKey is the lines' product line, or "" for lines with none.
	ProductLineKey     string
	SalesOrderNumber   string
	BuyerAccountID     string
	SalesRepID         *string
	StatusCode         string
	IssuedAt           *time.Time
	FirstShipAt        *time.Time
	CompletedAt        *time.Time
	LeadTimeDays       *int32
	LeadTimeSourceCode *string
	// QuantityOrdered and QuantityPacked are DECIMAL(65,30) text: raw stored quantities, summed.
	QuantityOrdered string
	QuantityPacked  string
}

// DeliveryFactScope names what a delivery dirty mark covers; the refresher resolves each to the orders it touches.
type DeliveryFactScope string

const (
	DeliveryFactScopeSalesOrder     DeliveryFactScope = "sales_order"
	DeliveryFactScopeSalesOrderLine DeliveryFactScope = "sales_order_line"
	DeliveryFactScopePick           DeliveryFactScope = "pick"
	DeliveryFactScopePickLine       DeliveryFactScope = "pick_line"
	// DeliveryFactScopeQuantity is an order line's or pick line's quantity row, edited on its own.
	DeliveryFactScopeQuantity DeliveryFactScope = "quantity"
	// DeliveryFactScopeProduct is a product whose product line or type changed.
	DeliveryFactScopeProduct DeliveryFactScope = "product"
	// DeliveryFactScopeReconcile asks for the reconcile pass to start again now, for a change no
	// narrower scope reaches (a customer merge moving orders to another buyer).
	DeliveryFactScopeReconcile DeliveryFactScope = "reconcile"
)

// DeliveryFactDirtyMark is one delivery_fact_dirty row as it was read.
type DeliveryFactDirtyMark struct {
	ScopeType DeliveryFactScope
	ScopeID   string
	AccountID string
	MarkedAt  time.Time
}

// DeliveryFactSync is the delivery reconcile pass's state. A pass in progress has a cursor; the first
// pass to complete is the backfill, and the fact is read only once one has.
type DeliveryFactSync struct {
	// CursorSalesOrderID is the last sales order the pass in progress reached, "" at its start.
	CursorSalesOrderID *string
	PassStartedAt      *time.Time
	LastCompletedAt    *time.Time
}
