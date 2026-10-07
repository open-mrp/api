package domain

import (
	"time"

	"github.com/open-mrp/api/shared/pagination"
)

// --- Open Orders ---

// OpenOrderFilter selects the sale lines of an account's open sales orders: issued and not yet completed. Every list is empty-means-all and they combine with AND.
type OpenOrderFilter struct {
	// AccountID is set by the service from the caller's identity.
	AccountID string
	// CustomerIDs match the buyer or any child account of it.
	CustomerIDs      []string
	CustomerGroupIDs []string
	SalesRepIDs      []string
	// ProductLineIDs and ItemIDs choose which of an order's lines count; an order with none left is not open for the report.
	ProductLineIDs []string
	ItemIDs        []string
}

// OpenOrdersSummary is the money on open orders, as exact decimal strings in the price's currency.
type OpenOrdersSummary struct {
	Ordered     string
	BackOrdered string
	// Invoiced is what has been invoiced against the open lines so far.
	Invoiced string
}

type AnalyzeOpenOrderProductsParams struct {
	OpenOrderFilter
	Limit  int32
	Cursor *string
}

// OpenOrderProduct is one item's quantities across the open lines, in the item's base unit, as exact decimal strings.
type OpenOrderProduct struct {
	ItemID      string
	Sku         string
	Description *string
	UnitID      string
	// Unit is the base unit UnitID names, attached by the service.
	Unit                *Unit
	QuantityOrdered     string
	QuantityBackOrdered string
	QuantityInvoiced    string
}

type OpenOrderProductPage struct {
	Products []OpenOrderProduct
	PageInfo pagination.PageInfo
}

type ListOpenOrdersParams struct {
	OpenOrderFilter
	Limit  int32
	Cursor *string
}

// OpenOrder is one open sales order with the lines the filter counts.
type OpenOrder struct {
	ID             string
	Number         string
	Status         string
	IssuedAt       time.Time
	CustomerID     string
	CustomerName   string
	CustomerNumber *string
	ShipToState    *string
	ShipToCountry  *string
	LineCount      int64
	// TotalOrdered is the counted lines' ordered value, an exact decimal string.
	TotalOrdered string
}

type OpenOrderPage struct {
	Orders   []OpenOrder
	PageInfo pagination.PageInfo
}

// OpenOrderLine is one sale line of an order. Quantities are in the item's base unit; the unit price is per the base unit of its denominator's dimension.
type OpenOrderLine struct {
	ID          string
	ItemID      string
	Sku         string
	Description *string
	UnitID      string
	// Unit is the base unit UnitID names, attached by the service.
	Unit                       *Unit
	UnitPrice                  string
	UnitPriceNumeratorUnitID   string
	UnitPriceNumeratorAbbr     string
	UnitPriceDenominatorUnitID string
	UnitPriceDenominatorAbbr   string
	QuantityBackOrdered        string
	QuantityInvoiced           string
	TotalOrdered               string
}

// ExportOpenOrderLinesParams is what an accepted open-order-lines export records on its job and replays in the worker, which has no caller: the service narrows it to what the caller may see before storing it.
type ExportOpenOrderLinesParams struct {
	OpenOrderFilter
	// HideCost drops the unit cost column, for a sales rep.
	HideCost bool
}
