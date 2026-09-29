package domain

import "time"

// SalesLineFact is one invoice line priced for sales analytics, as stored in sales_line_fact.
//
// Amounts are exact DECIMAL(65,30) strings, nil where the legacy expression is NULL (e.g. an item whose category has no base unit has no base quantity). They stay strings end to end so a refresh can compare and rewrite them without rounding.
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
)

// SalesFactDirtyMark is a scope whose facts are waiting to be recomputed.
type SalesFactDirtyMark struct {
	ScopeType SalesFactScope
	ScopeID   string
	AccountID string
	// MarkedAt is compared on clear, so a mark refreshed while the refresh ran is kept.
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
