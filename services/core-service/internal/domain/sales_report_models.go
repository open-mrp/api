package domain

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/pagination"
)

// SalesReportFilter selects the invoiced sales a report covers. Every list is empty-means-all and they combine with AND.
type SalesReportFilter struct {
	// AccountID is set by the service from the caller's identity.
	AccountID string
	// StartsAt and EndsAt bound the current period by invoice date, inclusive.
	StartsAt time.Time
	EndsAt   time.Time
	// ComparisonStartsAt and ComparisonEndsAt bound an optional second period; both or neither.
	ComparisonStartsAt *time.Time
	ComparisonEndsAt   *time.Time
	// CustomerIDs match the buyer or any child account of it.
	CustomerIDs      []string
	CustomerGroupIDs []string
	ProductLineIDs   []string
	SalesRepIDs      []string
	ItemIDs          []string
}

// HasComparison reports whether a comparison period is set.
func (f SalesReportFilter) HasComparison() bool {
	return f.ComparisonStartsAt != nil && f.ComparisonEndsAt != nil
}

// SalesTotals is invoiced sales over some set of lines. Amounts are exact decimal strings: sums of the stored line amounts, never rounded.
type SalesTotals struct {
	// PeriodStart is the first day of a daily bucket, as midnight UTC of the caller's local day; nil on a whole period.
	PeriodStart *time.Time
	Invoiced    string
	// Cost is nil when the caller may not see cost.
	Cost *string
	// Quantity is invoiced quantity, each line in its item's base unit.
	Quantity     string
	InvoiceCount int64
	LineCount    int64
}

type AnalyzeSalesSummaryParams struct {
	SalesReportFilter
	// TZOffsetMinutes is the caller's offset east of UTC; daily totals are bucketed by the caller's local day.
	TZOffsetMinutes int32
}

// SalesSummary is the current period's totals and daily totals, and the comparison period's when one is set.
type SalesSummary struct {
	Current         SalesTotals
	Daily           []SalesTotals
	Comparison      *SalesTotals
	ComparisonDaily []SalesTotals
}

type AnalyzeSalesBreakdownParams struct {
	SalesReportFilter
	GroupBy constants.SalesBreakdownGroupBy
	Limit   int32
	// Cursor is a page_info cursor from a previous page; nil starts at the largest group.
	Cursor *string
}

// SalesBreakdown is one page of invoiced totals grouped by a dimension, largest current-period total first.
type SalesBreakdown struct {
	Groups   []SalesBreakdownGroup
	PageInfo pagination.PageInfo
}

type SalesBreakdownGroup struct {
	// Key is the id of the customer, item, product line, customer group, sales rep (account user) or discount.
	Key   string
	Label string
	// Description and UnitAbbreviation are set when grouping by product: the item's description and base unit.
	Description      *string
	UnitAbbreviation *string
	Totals           SalesTotals
	// Comparison is set when a comparison period is.
	Comparison *SalesTotals
}

type AnalyzeSalesInvoicesParams struct {
	SalesReportFilter
	Limit int32
	// Cursor is a page_info cursor from a previous page; nil starts at the newest invoice.
	Cursor *string
}

type SalesInvoicePage struct {
	Invoices []SalesInvoiceSummary
	PageInfo pagination.PageInfo
}

type SalesInvoiceSummary struct {
	InvoiceID     string
	InvoiceNumber string
	CustomerID    string
	CustomerName  string
	InvoicedAt    time.Time
	// ItemCount is the number of distinct items on the invoice's matching lines.
	ItemCount int64
	Invoiced  string
}

type ListSalesLinesParams struct {
	SalesReportFilter
	// HasWindow is false for an unbounded listing; StartsAt/EndsAt are then ignored.
	HasWindow bool
	Limit     int32
	// Cursor is a page_info cursor from a previous page; nil starts at the newest line.
	Cursor *string
}

type SalesLinePage struct {
	Lines    []SalesEntry
	PageInfo pagination.PageInfo
}

// ExportSalesLinesParams is what an accepted sales-lines export records on its job and replays in the worker, which has no caller: the service narrows it to what the caller may see before storing it.
type ExportSalesLinesParams struct {
	SalesReportFilter
	// HasWindow is false for an export of every invoiced line; StartsAt/EndsAt are then ignored.
	HasWindow bool
	// HideCost drops the unit cost column, for a sales rep.
	HideCost bool
}
