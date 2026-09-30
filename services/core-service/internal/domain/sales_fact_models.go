package domain

import "time"

// SalesLineFact is one invoice line priced for sales analytics, as stored in sales_line_fact.
//
// Amounts are exact DECIMAL(28,10) strings, nil where the legacy expression is NULL (e.g. an item whose category has no base unit has no base quantity). They stay strings end to end so a refresh can compare and rewrite them without rounding.
type SalesLineFact struct {
	AccountID          string
	InvoicedAt         time.Time
	InvoiceLineID      string
	InvoiceID          string
	SalesOrderID       string
	SalesOrderTypeCode string
	BuyerAccountID     string
	SalesRepID         *string
	OrderDiscountID    *string
	ProductID          string
	ItemID             string
	ProductLineID      string
	QuantityBase       *string
	TotalInvoiced      *string
	TotalCost          *string
}

// SalesFactScope names what a dirty mark covers; the refresher resolves each to the invoices it touches.
type SalesFactScope string

const (
	SalesFactScopeInvoice        SalesFactScope = "invoice"
	SalesFactScopeSalesOrder     SalesFactScope = "sales_order"
	SalesFactScopeSalesOrderLine SalesFactScope = "sales_order_line"
	SalesFactScopeProduct        SalesFactScope = "product"
	// SalesFactScopeQuantity is an invoice line's own quantity row.
	SalesFactScopeQuantity SalesFactScope = "quantity"
	// SalesFactScopeRate is an order line's unit price or unit cost rate row.
	SalesFactScopeRate SalesFactScope = "rate"
	// SalesFactScopeItem is an item whose category (and so base unit) changed.
	SalesFactScopeItem SalesFactScope = "item"
	// SalesFactScopeBuyer is a customer whose orders moved to another (a merge): the invoices whose
	// facts still name it as buyer.
	SalesFactScopeBuyer SalesFactScope = "buyer"
	// SalesFactScopeReconcile asks for the full reconcile pass to start again now. It covers changes that
	// can reprice any line, such as a unit's ratio or a category's base unit; its scope id is unused.
	SalesFactScopeReconcile SalesFactScope = "reconcile"
)

// SalesFactDirtyMark is a scope whose facts are waiting to be recomputed.
type SalesFactDirtyMark struct {
	ScopeType SalesFactScope
	ScopeID   string
	AccountID string
	// MarkedAt is compared on clear, so a mark refreshed while the refresh ran is kept.
	MarkedAt time.Time
}

// SalesRollupDirtyMark is an (account, UTC day) whose rollup buckets must be rebuilt from its facts.
type SalesRollupDirtyMark struct {
	Day SalesRollupDay
	// MarkedAt is compared on clear, so a day re-marked while its rebuild ran is kept.
	MarkedAt time.Time
}

// SalesFactInvoiceCursor is a position in the reconcile sweep's (created_at, id) walk over invoices.
type SalesFactInvoiceCursor struct {
	InvoiceID string
	CreatedAt time.Time
}

// SalesFactSync is the reconcile sweep's persisted progress.
type SalesFactSync struct {
	// Cursor is where the pass in progress resumes; nil when no pass is running.
	Cursor *SalesFactInvoiceCursor
	// PassStartedAt is when the pass in progress (or the last one) began.
	PassStartedAt *time.Time
	// LastCompletedAt is when a pass last reached the newest invoice; nil until the backfill finishes.
	LastCompletedAt *time.Time
}

// SalesRollupDay is one account's UTC day, the unit the rollup sweep rebuilds.
type SalesRollupDay struct {
	AccountID string
	// Day is midnight UTC.
	Day time.Time
}

// SalesRollupSync is the rollup sweep's persisted progress.
type SalesRollupSync struct {
	// Cursor is the next day the pass in progress rebuilds; nil when no pass is running.
	Cursor *SalesRollupDay
	// PassStartedAt is when the pass in progress (or the last one) began.
	PassStartedAt *time.Time
	// LastCompletedAt is when a pass last finished; nil until the backfill finishes.
	LastCompletedAt *time.Time
}
